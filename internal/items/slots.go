package items

// Equipment slots. The taxonomy is physical: an item's slot is derived from
// what the object is, not from its stats.
//
//   - weapon: held weapons (swords, bows, staves, ...)
//   - armor:  worn protective gear (armor, cloaks, robes, helmets, ...)
//   - jewelry: precious ornaments worn on the body (rings, amulets, pendants,
//     brooches, talismans, crowns, ...)
//   - trinket: every other non-weapon/non-armor piece (charms, masks, badges,
//     seals, orbs, cores, shards, keys, scopes, lockpicks, posters, ...)
const (
	SlotWeapon  = "weapon"
	SlotArmor   = "armor"
	SlotJewelry = "jewelry"
	SlotTrinket = "trinket"
)

// EquipSlots lists every equipment slot in display order.
var EquipSlots = []string{SlotWeapon, SlotArmor, SlotJewelry, SlotTrinket}
