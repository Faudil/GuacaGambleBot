// Command split_locales migrates the legacy flat locale packs
// (locales/en.json, locales/fr.json) into per-namespace files
// (locales/en/<namespace>.json, locales/fr/<namespace>.json).
//
// Usage: go run ./tools/split_locales [locales-dir]
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

func main() {
	dir := "locales"
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}

	files, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		fatal(err)
	}
	if len(files) == 0 {
		fatal(fmt.Errorf("no locale packs found in %s", dir))
	}

	for _, f := range files {
		lang := filepath.Base(f)
		lang = lang[:len(lang)-len(".json")]
		outDir := filepath.Join(dir, lang)

		data, err := os.ReadFile(f)
		if err != nil {
			fatal(err)
		}
		var pack map[string]any
		if err := json.Unmarshal(data, &pack); err != nil {
			fatal(fmt.Errorf("%s: %w", f, err))
		}

		if err := os.MkdirAll(outDir, 0o755); err != nil {
			fatal(err)
		}

		namespaces := make([]string, 0, len(pack))
		for k := range pack {
			namespaces = append(namespaces, k)
		}
		sort.Strings(namespaces)

		for _, ns := range namespaces {
			nsFile := filepath.Join(outDir, ns+".json")
			raw, err := json.MarshalIndent(map[string]any{ns: pack[ns]}, "", "  ")
			if err != nil {
				fatal(err)
			}
			raw = append(raw, '\n')
			if err := os.WriteFile(nsFile, raw, 0o644); err != nil {
				fatal(err)
			}
			fmt.Printf("%s: wrote %s\n", lang, nsFile)
		}

		fmt.Printf("%s: %d namespaces split into %s\n", f, len(pack), outDir)
	}

	// Once all packs have been split, drop the originals.
	for _, f := range files {
		if err := os.Remove(f); err != nil {
			fatal(err)
		}
		fmt.Printf("removed %s\n", f)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "split_locales:", err)
	os.Exit(1)
}
