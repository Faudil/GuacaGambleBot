package i18n

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// translations holds the loaded locale data: lang -> nested map.
var translations = map[string]map[string]any{}

// Load reads every *.json file in dir as a language pack (filename without
// extension is the language code, e.g. "en", "fr").
func Load(dir string) error {
	translations = map[string]map[string]any{}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		lang := strings.TrimSuffix(e.Name(), ".json")
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return err
		}
		var m map[string]any
		if err := json.Unmarshal(data, &m); err != nil {
			return err
		}
		translations[lang] = m
	}
	return nil
}

func getNested(m map[string]any, keys []string) any {
	var cur any = m
	for _, k := range keys {
		mm, ok := cur.(map[string]any)
		if !ok {
			return nil
		}
		cur, ok = mm[k]
		if !ok {
			return nil
		}
	}
	return cur
}

// T resolves a dotted key (e.g. "economy.balance_title") for the given language.
// Missing params are left untouched. If the key is absent in the requested
// language, French is used as a fallback; if still absent the key itself is
// returned so missing translations are visible rather than silent.
func T(key, lang string, params ...map[string]any) string {
	if translations == nil {
		return key
	}
	keys := strings.Split(key, ".")
	var val any
	if ld, ok := translations[lang]; ok {
		val = getNested(ld, keys)
	}
	if val == nil && lang != "fr" {
		if fd, ok := translations["fr"]; ok {
			val = getNested(fd, keys)
		}
	}
	if val == nil {
		return key
	}
	s, ok := val.(string)
	if !ok {
		if lst, ok := val.([]any); ok {
			parts := make([]string, 0, len(lst))
			for _, v := range lst {
				if str, ok := v.(string); ok {
					parts = append(parts, str)
				}
			}
			s = strings.Join(parts, "\n")
		} else {
			return fmt.Sprintf("%v", val)
		}
	}
	if len(params) > 0 {
		for k, v := range params[0] {
			s = strings.ReplaceAll(s, "{"+k+"}", fmt.Sprintf("%v", v))
		}
	}
	return s
}
