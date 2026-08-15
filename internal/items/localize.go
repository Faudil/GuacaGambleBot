package items

import (
	"guacagamblebot/internal/i18n"
)

// LocalizedName resolves the item's display name for the given language. It is
// looked up in the "itemnames" locale namespace keyed by the item ID; when no
// translation exists the canonical English name is returned.
func LocalizedName(id, lang string) string {
	it := Get(id)
	if it == nil {
		return id
	}
	return it.LocalizedName(lang)
}

// LocalizedDescription resolves the item's description for the given language.
// When no translation exists the canonical English description is returned.
func LocalizedDescription(id, lang string) string {
	it := Get(id)
	if it == nil {
		return id
	}
	return it.LocalizedDescription(lang)
}

// LocalizedName is the item-aware variant of LocalizedName.
func (it *Item) LocalizedName(lang string) string {
	if tr := i18n.T("itemnames."+it.ID+".name", lang); tr != "itemnames."+it.ID+".name" {
		return tr
	}
	return it.Name
}

// LocalizedDescription is the item-aware variant of LocalizedDescription.
func (it *Item) LocalizedDescription(lang string) string {
	if tr := i18n.T("itemnames."+it.ID+".description", lang); tr != "itemnames."+it.ID+".description" {
		return tr
	}
	return it.Description
}