package items

import (
	"testing"

	"guacagamblebot/internal/i18n"
)

func TestLocalizeNameAndDescription(t *testing.T) {
	if err := i18n.Load("../../locales"); err != nil {
		t.Fatalf("failed to load locales: %v", err)
	}

	it := Get("coal")
	if it == nil {
		t.Fatal("coal item not found")
	}
	if got := it.LocalizedName("fr"); got != "Charbon" {
		t.Errorf("coal fr name = %q", got)
	}
	if got := it.LocalizedName("en"); got != "Coal" {
		t.Errorf("coal en name = %q", got)
	}
	if got := it.LocalizedDescription("fr"); got != "Pas mal pour se réchauffer." {
		t.Errorf("coal fr description = %q", got)
	}
	if got := it.LocalizedDescription("en"); got != "Great for keeping warm." {
		t.Errorf("coal en description = %q", got)
	}

	// French translation that came from the new catalog entries.
	sword := Get("dragon_slayer_sword")
	if sword == nil {
		t.Fatal("dragon_slayer_sword item not found")
	}
	if got := sword.LocalizedName("fr"); got != "Épée Tueuse de Dragon" {
		t.Errorf("dragon_slayer_sword fr name = %q", got)
	}
}

func TestLocalizeFallback(t *testing.T) {
	if err := i18n.Load("../../locales"); err != nil {
		t.Fatalf("failed to load locales: %v", err)
	}
	if got := LocalizedName("definitely_missing_item_xyz", "fr"); got != "definitely_missing_item_xyz" {
		t.Errorf("missing id fallback = %q", got)
	}
}