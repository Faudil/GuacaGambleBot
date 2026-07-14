package i18n

import (
	"strings"
	"testing"
)

func TestLoadAndTranslate(t *testing.T) {
	if err := Load("../../locales"); err != nil {
		t.Fatalf("failed to load locales: %v", err)
	}

	if got := T("economy.balance_title", "fr"); got != "🏦 Ma Banque" {
		t.Errorf("fr balance_title = %q", got)
	}
	if got := T("economy.balance_title", "en"); got != "🏦 My Bank" {
		t.Errorf("en balance_title = %q", got)
	}
}

func TestTranslateParams(t *testing.T) {
	if err := Load("../../locales"); err != nil {
		t.Fatalf("failed to load locales: %v", err)
	}
	got := T("economy.rsa_desc", "en", map[string]any{"user": "Bob", "amount": 50})
	if !strings.Contains(got, "Bob") || !strings.Contains(got, "50") {
		t.Errorf("params not substituted: %q", got)
	}
}

func TestFallbackToKey(t *testing.T) {
	if err := Load("../../locales"); err != nil {
		t.Fatalf("failed to load locales: %v", err)
	}
	missing := "economy.this_key_does_not_exist"
	if got := T(missing, "en"); got != missing {
		t.Errorf("expected key fallback, got %q", got)
	}
}

func TestFrenchFallback(t *testing.T) {
	if err := Load("../../locales"); err != nil {
		t.Fatalf("failed to load locales: %v", err)
	}
	// A key present only conceptually via fr should resolve when asked in an
	// unknown language that falls back to fr.
	if got := T("economy.menu_title", "de"); got != "🏦 Économie" {
		t.Errorf("fr fallback failed: %q", got)
	}
}
