package items

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCatalogUniqueIDs guards the catalog against duplicate item IDs. The
// byID map is built silently, so a duplicate would overwrite the first entry
// without any error.
func TestCatalogUniqueIDs(t *testing.T) {
	seen := map[string]string{}
	for _, it := range AllItems() {
		if prev, ok := seen[it.ID]; ok {
			t.Errorf("duplicate item id %q (also used by %q)", it.ID, prev)
		}
		seen[it.ID] = it.Name
	}
}

// TestCatalogSources checks that every item carries a known source tag, so the
// balance review (grep "Source:" catalog.go) always covers the whole catalog.
func TestCatalogSources(t *testing.T) {
	valid := map[string]bool{
		SourceShop:        true,
		SourceCraft:       true,
		SourceQuest:       true,
		SourceBoss:        true,
		SourceCriminality: true,
		SourceVeil:        true,
	}
	for _, it := range AllItems() {
		assert.NotEmpty(t, it.Source, "item %q must carry a Source tag", it.ID)
		assert.True(t, valid[it.Source], "item %q has unknown source %q", it.ID, it.Source)
	}
}

// TestCatalogSetReferences resolves every SetID against the set registry and
// makes sure each set's pieces are all present in the catalog.
func TestCatalogSetReferences(t *testing.T) {
	pieces := map[string][]string{}
	for _, it := range AllItems() {
		if it.SetID == "" {
			continue
		}
		set, ok := SetsByName[it.SetID]
		require.True(t, ok, "item %q references unknown set %q", it.ID, it.SetID)
		assert.NotEmpty(t, set.Name, "set %q has a name", it.SetID)
		pieces[it.SetID] = append(pieces[it.SetID], it.ID)
	}
	for id := range SetsByName {
		if SetsByName[id].Procedural {
			continue
		}
		assert.NotEmpty(t, pieces[id], "set %q has no pieces in the catalog", id)
	}
}

// TestCatalogAffixSlots restricts affix slot lists to known equipment slots.
func TestCatalogAffixSlots(t *testing.T) {
	valid := map[string]bool{}
	for _, s := range EquipSlots {
		valid[s] = true
	}
	for _, a := range AffixPool {
		for _, s := range a.Slots {
			assert.True(t, valid[s], "affix %q targets unknown slot %q", a.ID, s)
		}
	}
}

// TestCatalogDelveLoot validates the delve procedural tables: unique IDs,
// known slots and sane rarity floors.
func TestCatalogDelveLoot(t *testing.T) {
	assertUnique := func(name string, ids []string) {
		t.Helper()
		seen := map[string]bool{}
		for _, id := range ids {
			assert.False(t, seen[id], "duplicate %s id %q", name, id)
			seen[id] = true
		}
	}

	baseIDs, prefixIDs, suffixIDs := []string{}, []string{}, []string{}
	for _, b := range DelveBases {
		baseIDs = append(baseIDs, b.ID)
		assert.GreaterOrEqual(t, b.MinRar.Rank(), 0, "base %q has invalid rarity", b.ID)
		assert.Contains(t, EquipSlots, b.EquipSlot, "base %q has unknown slot %q", b.ID, b.EquipSlot)
	}
	for _, p := range DelvePrefixes {
		prefixIDs = append(prefixIDs, p.ID)
		assert.GreaterOrEqual(t, p.MinRar.Rank(), 0, "prefix %q has invalid rarity", p.ID)
	}
	for _, s := range DelveSuffixes {
		suffixIDs = append(suffixIDs, s.ID)
		assert.GreaterOrEqual(t, s.MinRar.Rank(), 0, "suffix %q has invalid rarity", s.ID)
	}
	assertUnique("base", baseIDs)
	assertUnique("prefix", prefixIDs)
	assertUnique("suffix", suffixIDs)
}

// TestCatalogVeilDropsResolve makes sure every veil raid drop ID resolves to a
// catalog item (the veil pool must never drift from the catalog again).
func TestCatalogVeilDropsResolve(t *testing.T) {
	for _, id := range []string{
		"rift_blade", "dechirure_scythe", "rift_cowl", "rift_warden_aegis", "rift_band", "rift_eye",
	} {
		it := Get(id)
		require.NotNil(t, it, "veil drop %q must exist in the catalog", id)
		assert.Equal(t, SourceVeil, it.Source, "veil drop %q must be tagged SourceVeil", id)
		assert.Equal(t, "rift_walker", it.SetID, "veil drop %q must be a Rift Walker piece", id)
	}
}
