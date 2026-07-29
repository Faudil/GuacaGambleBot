package delve

type FlagDef struct {
	Sentence   string
	Epithet    string
	Unlocks    []string
	HiddenFrom []string
}

var FlagManifest = map[string]FlagDef{
	"first_descent": {
		Sentence: "Your first descent into the Undercroft marked the beginning of a legend.",
	},
	"escaped_alive": {
		Sentence: "You emerged from the darkness, battered but breathing. The Undercroft would remember you.",
		Epithet:  "the Survivor",
	},
	"fell_in_battle": {
		Sentence: "The depths claimed you once, but death is not the end for those marked by fate.",
	},
	"fled_from_depths": {
		Sentence: "You ran. The shadows whispered your cowardice, but survival has its own wisdom.",
	},
	"cleared_floor_5": {
		Sentence: "You were among the first to reach Floor 5. The Undercroft acknowledged your strength.",
		Epithet:  "the Deep Walker",
	},
	"cleared_floor_10": {
		Sentence: "Floor 10. The Abyss opened before you, and you did not flinch.",
		Epithet:  "the Abyssal",
	},
	"freed_prisoner": {
		Sentence: "In the darkness, you chose mercy. A life freed because you decided it mattered.",
		Unlocks:  []string{"faction_ally"},
	},
	"betrayed_npc": {
		Sentence: "Gold over compassion. The chains you left behind still echo with broken promises.",
		Epithet:  "the Forsworn",
		Unlocks:  []string{"betrayer_flag"},
	},
	"desecrated_altar": {
		Sentence: "You defiled the sacred and took its power. The dark mark on your soul cannot be washed away.",
		Epithet:  "the Unholy",
		Unlocks:  []string{"dark_mark_carrier"},
	},
	"sacrificed_hp": {
		Sentence: "A piece of your vitality, traded for strength. The altar drank deeply.",
		Epithet:  "the Scarred",
	},
	"solved_riddle": {
		Sentence: "Your wit proved sharper than any blade. The vault opened its secrets to you.",
	},
	"opened_treasure_trap": {
		Sentence: "The chest's curse tested your resolve. You bore its mark and claimed its prize.",
	},
	"disarmed_treasure": {
		Sentence: "Steady hands and sharp eyes. The trap was disarmed, the treasure taken safely.",
		Epithet:  "the Cautious",
	},
	"spared_mimic": {
		Sentence: "You sensed the deception and chose to spare the mimic. A strange friendship may yet bloom.",
		HiddenFrom: []string{"betrayed_npc"},
		Unlocks:    []string{"mimic_ally"},
	},
	"merchant_purchase": {
		Sentence: "The wandering merchant smiled as you traded. Somewhere, a deal is a bond.",
	},
	"used_torch": {
		Sentence: "A torch spent to restore your strength. Light is precious in the darkness.",
	},
	"slept_unprotected": {
		Sentence: "You slept without guard. The Undercroft could have taken you, but it let you wake.",
	},
	"ambushed_while_sleeping": {
		Sentence: "Your rest was shattered by claws in the dark. You survived, but trust is a luxury now.",
	},
	"helped_warden": {
		Sentence: "The Lost Warden bowed their head. 'One day,' they said, 'I will repay this debt.'",
		Unlocks:  []string{"warden_ally"},
	},
	"set_item_collected": {
		Sentence: "A piece of the ancient set now calls you its bearer. The power grows.",
	},
	"full_set_equipped": {
		Sentence: "You bear the complete set. Its legacy awakens, and the ground trembles.",
		Epithet:  "the Chosen of the Set",
	},
	"soulbound_keepsake": {
		Sentence: "A soulbound keepsake found in the depths. It remembers what you did when no one was watching.",
	},
	"mercy_on_enemy": {
		Sentence: "The monster knelt, and you stayed your hand. Some debts cannot be quantified.",
		Unlocks:  []string{"spared_enemy"},
	},
	"defeated_zone_boss": {
		Sentence: "The zone's master recognized your strength. Its defeat reshapes the dungeon's memory of you.",
	},
	"rescued_another": {
		Sentence: "You reached into the darkness and pulled another back. No one is forgotten.",
		Epithet:  "the Savior",
	},
	"touched_by_shadow": {
		Sentence: "The shadows recognize you. Where light falters, you have walked — and the darkness remembers.",
		Unlocks:  []string{"mask_of_malveillance"},
	},
	"mask_of_malveillance": {
		Sentence: "You claimed the Mask of Malveillance from the Gravewarden's grasp. The underworld stirs.",
		Epithet:  "the Shadow-Bearer",
	},
	"tomb_raider": {
		Sentence: "You dared to open the sealed sarcophagus and claim what lay within. The dead do not rest easy.",
	},
	"shrine_blessed": {
		Sentence: "A holy shrine recognized your worth. The blessing lingers like a warm ember in your chest.",
	},
	"shrine_defiled": {
		Sentence: "You defiled a sacred place and took its power. The light dims wherever you walk now.",
	},
	"key_master": {
		Sentence: "You unlocked the path with a key. The clever always find a way.",
	},
}

func GetFlagSentence(flagID string) string {
	if f, ok := FlagManifest[flagID]; ok {
		return f.Sentence
	}
	return ""
}

func GetFlagEpithet(flagID string) string {
	if f, ok := FlagManifest[flagID]; ok {
		return f.Epithet
	}
	return ""
}
