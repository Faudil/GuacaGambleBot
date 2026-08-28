package main

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	helpcog "guacagamblebot/internal/cogs/help"
	"guacagamblebot/internal/i18n"
)

// These assert against the real wiring: every command the bot registers must be
// reachable from /help, in every language.

func TestNoCommandIsMissingFromHelp(t *testing.T) {
	r := buildTestRouter(t)

	var uncategorised []string
	for _, c := range r.CommandList() {
		if c.Category == "" {
			uncategorised = append(uncategorised, c.Name)
		}
	}
	assert.Empty(t, uncategorised, "these commands are registered but filed under no help category, so /help would not show them")
}

func TestEveryHelpCategoryHasCommands(t *testing.T) {
	r := buildTestRouter(t)
	for _, cat := range []string{
		helpcog.CatStart, helpcog.CatEconomy, helpcog.CatCasino, helpcog.CatRPG,
		helpcog.CatActivities, helpcog.CatPets, helpcog.CatWorld, helpcog.CatAdmin,
	} {
		assert.NotEmpty(t, r.Catalog(cat), "category %q renders as an empty section", cat)
	}
}

// Discord rejects an embed field value over 1024 characters and a description
// over 4096, so a category that grows too large would break /help at runtime
// rather than at build time. This is the check that catches it first.
func TestHelpRendersWithinDiscordLimits(t *testing.T) {
	require.NoError(t, i18n.Load(filepath.Join("..", "..", "locales")))
	r := buildTestRouter(t)

	for _, lang := range i18n.Languages() {
		for _, cat := range r.Categories() {
			groups := r.Catalog(cat)

			overviewField := 0
			detail := 0
			for _, g := range groups {
				overviewField += len(g.Name) + 5 // "`/name` "
				detail += len(g.Name) + len(i18n.T(g.DescKey, lang)) + 32
			}
			assert.LessOrEqual(t, overviewField, 1024,
				"[%s] category %q overflows an embed field on the /help overview", lang, cat)
			assert.LessOrEqual(t, detail, 4096,
				"[%s] category %q overflows the embed description on its own page", lang, cat)
		}
	}
}
