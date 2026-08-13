// Command localehelper is a CLI for editing and validating the locale packs.
//
// Usage:
//
//	localehelper validate                 check key parity across languages
//	localehelper list [namespace]         list keys with a preview
//	localehelper show <key>               print a key in every language
//	localehelper edit <key>               edit a key in every language in $EDITOR
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// execCmd runs cmd as a shell command so EDITOR values with arguments
// (e.g. "code -w") work. Additional args are shell-quoted before appending.
func execCmd(cmd string, args ...string) *exec.Cmd {
	quoted := []string{cmd}
	for _, a := range args {
		quoted = append(quoted, "'"+strings.ReplaceAll(a, "'", `'\''`)+"'")
	}
	return exec.Command("sh", "-c", strings.Join(quoted, " "))
}

const localesDir = "locales"

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		usage()
		os.Exit(2)
	}
	var err error
	switch args[0] {
	case "validate":
		err = cmdValidate(args[1:])
	case "list":
		err = cmdList(args[1:])
	case "show":
		err = cmdShow(args[1:])
	case "edit":
		err = cmdEdit(args[1:])
	case "help", "-h", "--help":
		usage()
	default:
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "localehelper:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Print(`localehelper — edit and validate locale packs

Usage:
  localehelper validate                 check key parity across languages
  localehelper list [namespace]         list keys with a preview
  localehelper show <key>               print a key in every language
  localehelper edit <key>               edit a key in every language in $EDITOR
`)
}

// pack is one language: the merged data plus the per-namespace files
// so edits can be written back to disk.
type pack struct {
	lang string
	data map[string]any
	// files maps a top-level namespace to the raw content of its file.
	files map[string]map[string]any
}

func loadPacks(dir string) (map[string]*pack, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	packs := map[string]*pack{}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		p := &pack{lang: e.Name(), data: map[string]any{}, files: map[string]map[string]any{}}
		langDir := filepath.Join(dir, e.Name())
		fileEntries, err := os.ReadDir(langDir)
		if err != nil {
			return nil, err
		}
		for _, f := range fileEntries {
			if f.IsDir() || !strings.HasSuffix(f.Name(), ".json") {
				continue
			}
			raw, err := os.ReadFile(filepath.Join(langDir, f.Name()))
			if err != nil {
				return nil, err
			}
			var m map[string]any
			if err := json.Unmarshal(raw, &m); err != nil {
				return nil, fmt.Errorf("%s/%s: %w", e.Name(), f.Name(), err)
			}
			for k, v := range m {
				p.data[k] = v
				p.files[k] = m
			}
		}
		packs[e.Name()] = p
	}
	if len(packs) == 0 {
		return nil, fmt.Errorf("no language packs found in %s", dir)
	}
	return packs, nil
}

// leafKeys returns every dotted path whose value is a string or string array.
func leafKeys(m map[string]any, prefix string, out *[]string) {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	for _, k := range ks {
		path := k
		if prefix != "" {
			path = prefix + "." + k
		}
		switch v := m[k].(type) {
		case map[string]any:
			leafKeys(v, path, out)
		case string, []any:
			*out = append(*out, path)
		}
	}
}

func getAt(m map[string]any, path string) any {
	cur := any(m)
	for _, seg := range strings.Split(path, ".") {
		mm, ok := cur.(map[string]any)
		if !ok {
			return nil
		}
		cur, ok = mm[seg]
		if !ok {
			return nil
		}
	}
	return cur
}

func setAt(m map[string]any, path string, v any) {
	segs := strings.Split(path, ".")
	cur := m
	for _, seg := range segs[:len(segs)-1] {
		next, ok := cur[seg].(map[string]any)
		if !ok {
			next = map[string]any{}
			cur[seg] = next
		}
		cur = next
	}
	cur[segs[len(segs)-1]] = v
}

func displayValue(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case []any:
		parts := make([]string, 0, len(t))
		for _, item := range t {
			if s, ok := item.(string); ok {
				parts = append(parts, s)
			}
		}
		return strings.Join(parts, "\n")
	default:
		return fmt.Sprintf("%v", v)
	}
}

func keyExists(packs map[string]*pack, path string) bool {
	for _, p := range packs {
		if getAt(p.data, path) != nil {
			return true
		}
	}
	return false
}

// ---- validate ----

func cmdValidate(args []string) error {
	if len(args) > 1 {
		return fmt.Errorf("validate takes no arguments")
	}
	packs, err := loadPacks(localesDir)
	if err != nil {
		return err
	}

	langs := make([]string, 0, len(packs))
	for l := range packs {
		langs = append(langs, l)
	}
	sort.Strings(langs)

	keySets := map[string]map[string]bool{}
	for _, l := range langs {
		var keys []string
		leafKeys(packs[l].data, "", &keys)
		set := map[string]bool{}
		for _, k := range keys {
			set[k] = true
		}
		keySets[l] = set
	}

	all := map[string]bool{}
	for _, set := range keySets {
		for k := range set {
			all[k] = true
		}
	}

	missing := map[string][]string{}
	for l := range packs {
		for k := range all {
			if !keySets[l][k] {
				missing[l] = append(missing[l], k)
			}
		}
	}

	problems := 0
	for _, l := range langs {
		keys := make([]string, 0, len(keySets[l]))
		for k := range keySets[l] {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		fmt.Printf("%s: %d keys\n", l, len(keys))
	}
	fmt.Println()
	for _, l := range langs {
		for _, k := range missing[l] {
			fmt.Printf("MISSING in %s: %s\n", l, k)
			problems++
		}
	}
	if problems > 0 {
		fmt.Printf("\n%d keys missing in at least one language.\n", problems)
		os.Exit(1)
	}
	fmt.Println("All languages have matching keys.")
	return nil
}

// ---- list ----

func cmdList(args []string) error {
	var filter string
	switch len(args) {
	case 0:
	case 1:
		filter = args[0]
	default:
		return fmt.Errorf("list takes at most one namespace argument")
	}

	packs, err := loadPacks(localesDir)
	if err != nil {
		return err
	}
	langs := make([]string, 0, len(packs))
	for l := range packs {
		langs = append(langs, l)
	}
	sort.Strings(langs)
	primary := packs[langs[0]]

	var keys []string
	leafKeys(primary.data, "", &keys)
	for _, k := range keys {
		if filter != "" && !strings.HasPrefix(k, filter) {
			continue
		}
		preview := displayValue(getAt(primary.data, k))
		if len(preview) > 60 {
			preview = preview[:57] + "..."
		}
		preview = strings.ReplaceAll(preview, "\n", " \\n ")
		fmt.Printf("%-60s %s\n", k, preview)
	}
	return nil
}

// ---- show ----

func cmdShow(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: localehelper show <key>")
	}
	path := args[0]
	packs, err := loadPacks(localesDir)
	if err != nil {
		return err
	}
	if !keyExists(packs, path) {
		return fmt.Errorf("key %q not found in any language", path)
	}
	langs := make([]string, 0, len(packs))
	for l := range packs {
		langs = append(langs, l)
	}
	sort.Strings(langs)

	for _, l := range langs {
		v := getAt(packs[l].data, path)
		fmt.Printf("--- %s ---\n", l)
		if v == nil {
			fmt.Println("(missing)")
			continue
		}
		fmt.Println(displayValue(v))
		fmt.Println()
	}
	return nil
}

// ---- edit ----

func cmdEdit(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: localehelper edit <key>")
	}
	path := args[0]
	packs, err := loadPacks(localesDir)
	if err != nil {
		return err
	}
	if !keyExists(packs, path) {
		return fmt.Errorf("key %q not found in any language", path)
	}

	langs := make([]string, 0, len(packs))
	for l := range packs {
		langs = append(langs, l)
	}
	sort.Strings(langs)

	before := map[string]any{}
	for _, l := range langs {
		if v := getAt(packs[l].data, path); v != nil {
			before[l] = v
		}
	}

	tmp, err := os.CreateTemp("", "localehelper-*.json")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	// Pre-fill missing translations with the first language that has the key
	// so the author can translate them in the same pass.
	fallback := "en"
	if _, ok := before[fallback]; !ok {
		for l := range before {
			fallback = l
			break
		}
	}
	fill := map[string]any{}
	for _, l := range langs {
		if v, ok := before[l]; ok {
			fill[l] = v
		} else {
			fill[l] = before[fallback]
		}
	}
	raw, err := json.MarshalIndent(fill, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	if _, err := tmp.Write(raw); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}
	cmd := execCmd(editor, tmpName)
	fmt.Printf("Editing %s (%s)...\n", path, strings.Join(langs, ", "))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("editor failed: %w", err)
	}

	raw, err = os.ReadFile(tmpName)
	if err != nil {
		return err
	}
	var edited map[string]any
	if err := json.Unmarshal(raw, &edited); err != nil {
		return fmt.Errorf("file is not valid JSON: %w", err)
	}

	changed := false
	for _, l := range langs {
		v, ok := edited[l]
		if !ok {
			continue
		}
		if !sameValue(v, before[l]) {
			ns := strings.Split(path, ".")[0]
			file, ok := packs[l].files[ns]
			if !ok {
				return fmt.Errorf("%s: no file for namespace %q", l, ns)
			}
			setAt(file, path, v)
			changed = true
		}
	}
	if !changed {
		fmt.Println("No changes.")
		return nil
	}

	for _, l := range langs {
		for ns, m := range packs[l].files {
			out, err := json.MarshalIndent(m, "", "  ")
			if err != nil {
				return err
			}
			out = append(out, '\n')
			fn := filepath.Join(localesDir, l, ns+".json")
			if err := os.WriteFile(fn, out, 0o644); err != nil {
				return err
			}
		}
	}
	fmt.Println("Saved.")
	return nil
}

func sameValue(a, b any) bool {
	as, aok := a.(string)
	bs, bok := b.(string)
	if aok && bok {
		return as == bs
	}
	aj, err1 := json.Marshal(a)
	bj, err2 := json.Marshal(b)
	return err1 == nil && err2 == nil && string(aj) == string(bj)
}
