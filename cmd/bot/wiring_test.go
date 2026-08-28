package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"guacagamblebot/internal/i18n"
)

// The old /help documented 6 of 43 cogs because it was maintained by hand.
// These tests keep the generated one honest: every cog must be filed under a
// category, and every command must have a description in every language.

var (
	aliasImport = regexp.MustCompile(`(?m)^\s*(\w+)\s+"guacagamblebot/internal/(?:cogs/(\w+)|(onboarding))"`)
	tableEntry  = regexp.MustCompile(`(?m)^\s*(\w+)\.Register,`)
	slashCall   = regexp.MustCompile(`r\.Slash(?:WithOptions)?\(\s*"([^"]+)"\s*,\s*"([^"]+)"`)
)

func TestEveryCogIsFiledUnderAHelpCategory(t *testing.T) {
	src, err := os.ReadFile("main.go")
	require.NoError(t, err)

	wired := map[string]bool{}
	for _, m := range tableEntry.FindAllStringSubmatch(string(src), -1) {
		wired[m[1]] = true
	}
	var missing []string
	for _, m := range aliasImport.FindAllStringSubmatch(string(src), -1) {
		alias := m[1]
		if !wired[alias] {
			missing = append(missing, alias)
		}
	}
	assert.Empty(t, missing,
		"these cogs are imported but not filed under a help category in main.go, so their commands would be missing from /help")
}

func TestEveryCommandHasADescriptionInEveryLanguage(t *testing.T) {
	root := filepath.Join("..", "..")
	require.NoError(t, i18n.Load(filepath.Join(root, "locales")))
	langs := i18n.Languages()
	require.NotEmpty(t, langs)

	var checked int
	err := filepath.Walk(filepath.Join(root, "internal"), func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, m := range slashCall.FindAllStringSubmatch(string(src), -1) {
			name, key := m[1], m[2]
			assert.True(t, strings.HasPrefix(key, "cmd.") && strings.HasSuffix(key, ".desc"),
				"%s: /%s should register an i18n key like \"cmd.%s.desc\", got %q", path, name, name, key)
			for _, lang := range langs {
				got := i18n.T(key, lang)
				assert.NotEqual(t, key, got, "%s: /%s has no %s description for key %q", path, name, lang, key)
				// Discord rejects command descriptions longer than 100 characters.
				assert.LessOrEqual(t, len(got), 100, "%s: /%s %s description exceeds Discord's 100-character limit", path, name, lang)
			}
			checked++
		}
		return nil
	})
	require.NoError(t, err)
	assert.Greater(t, checked, 80, "expected to find the bot's slash registrations")
}
