package delve

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"

	"github.com/bwmarrin/discordgo"

	"guacagamblebot/internal/components"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/model"
	"guacagamblebot/internal/store"
)

type RoomType string

const (
	RoomMonster  RoomType = "monster"
	RoomTreasure RoomType = "treasure"
	RoomAltar    RoomType = "altar"
	RoomMerchant RoomType = "merchant"
	RoomPuzzle   RoomType = "puzzle"
	RoomRest     RoomType = "rest"
	RoomNPC      RoomType = "npc"
	RoomEmpty    RoomType = "empty"
	RoomTomb     RoomType = "tomb"
	RoomGarden   RoomType = "garden"
	RoomForge    RoomType = "anvil"
	RoomRift     RoomType = "rift"
	RoomShrine   RoomType = "shrine"
	RoomLocked   RoomType = "locked"
	RoomVaultKey RoomType = "vault_key"
)

type Room struct {
	Type        RoomType
	Description string
	Buttons     []RoomButton
	EventData   string
}

type RoomButton struct {
	Emoji  string
	Label  string
	Action string
	Style  discordgo.ButtonStyle
	Data   string
}

var zoneFloors = []struct {
	MinFloor int
	MaxFloor int
	Zone     string
}{
	{1, 3, "crypt"},
	{4, 6, "fungal_wilds"},
	{7, 9, "forge_district"},
	{10, 99, "abyss"},
}

func ZoneForFloor(floor int) string {
	for _, z := range zoneFloors {
		if floor >= z.MinFloor && floor <= z.MaxFloor {
			return z.Zone
		}
	}
	return "abyss"
}

type roomWeight struct {
	Type   RoomType
	Weight int
}

var zoneRoomTables = map[string][]roomWeight{
	"crypt": {
		{RoomMonster, 28}, {RoomTreasure, 12}, {RoomAltar, 5},
		{RoomMerchant, 8}, {RoomPuzzle, 8}, {RoomRest, 10},
		{RoomNPC, 5}, {RoomEmpty, 12}, {RoomTomb, 8},
		{RoomShrine, 5}, {RoomLocked, 8},
	},
	"fungal_wilds": {
		{RoomMonster, 26}, {RoomTreasure, 10}, {RoomAltar, 6},
		{RoomMerchant, 8}, {RoomPuzzle, 8}, {RoomRest, 8},
		{RoomNPC, 6}, {RoomEmpty, 12}, {RoomGarden, 8},
		{RoomShrine, 5}, {RoomLocked, 8},
	},
	"forge_district": {
		{RoomMonster, 23}, {RoomTreasure, 15}, {RoomAltar, 8},
		{RoomMerchant, 10}, {RoomPuzzle, 10}, {RoomRest, 6},
		{RoomNPC, 5}, {RoomEmpty, 8}, {RoomForge, 8},
		{RoomShrine, 5}, {RoomLocked, 8},
	},
	"abyss": {
		{RoomMonster, 33}, {RoomTreasure, 8}, {RoomAltar, 12},
		{RoomMerchant, 5}, {RoomPuzzle, 5}, {RoomRest, 5},
		{RoomNPC, 8}, {RoomEmpty, 10}, {RoomRift, 8},
		{RoomShrine, 5}, {RoomLocked, 8},
	},
}

var cryptDescs = []string{
	"Water drips from cracked stone faces lining the walls. At the far end, a rusted iron gate blocks the passage.",
	"Fragments of bone crunch beneath your feet. Faint torchlight flickers across faded murals of forgotten kings.",
	"The air is thick with the smell of damp earth and old death. Shadowy alcoves line both sides of the hall.",
	"A cold draft whispers through the cracks in the masonry. Somewhere, water drips in a steady rhythm.",
	"Sconces flicker with green-tinged flames, casting dancing shadows on the vaulted ceiling.",
}

var fungalDescs = []string{
	"Giant mushrooms tower overhead, releasing clouds of glowing spores. The ground squishes with every step.",
	"Strange fungi pulse with bioluminescent light, illuminating a maze of twisting root-tunnels.",
	"The air is heavy with the sweet-sick smell of decay. Phosphorescent moss clings to every surface.",
	"Massive fungal growths have burst through the stonework, their tendrils pulsing with an inner light.",
	"Tiny glowing insects swarm around clusters of shelf fungi, casting the cavern in shifting patterns of light.",
}

var forgeDescs = []string{
	"The heat hits you like a wall. Ancient forges still burn with unnatural flame, their bellows creaking on their own.",
	"Molten metal flows through channels in the floor, casting an orange glow across walls of blackened stone.",
	"The clang of hammer on metal echoes from somewhere deep within. Workbenches hold half-finished weapons, still warm.",
	"Massive gears line the walls, some still turning slowly. The floor trembles with the pulse of hidden machinery.",
	"Rusted chains hang from the ceiling, and the air shimmers with heat. A conveyor belt of cooled slag leads deeper.",
	"A seam of shimmering light runs down the far wall, humming softly. You press a hand to it — the stone feels thin, like something breathes behind a curtain.",
	"Dust lies in strange patterns across the floor: a key shape, a doorway, a line of marks repeated by something patient. A whisper follows you that sounds almost like a name.",
}

var abyssDescs = []string{
	"Reality bends here. The walls shift between stone, flesh, and void. You feel watched by something immense.",
	"Gravity fluctuates as you step forward. The path splits and reforms in ways that defy logic.",
	"Whispers echo from everywhere and nowhere. Fragments of forgotten memories cling to the air like cobwebs.",
	"The floor is a mirror of dark glass. Beneath it, shapes move — vast, slow, indifferent.",
	"Time loses meaning. You step and find yourself somewhere else, as though the room itself forgot you were there.",
}

var monsterDescs = []string{
	"Something stirs in the darkness ahead, dragging claws across stone. It knows you're here.",
	"A low growl rumbles from the shadows. Two pale eyes open, fixing on you with hungry intent.",
	"The creature before you is a twisted mockery of life, its form barely held together by malice.",
	"Chittering echoes from the ceiling as a mass of limbs and eyes descends toward you.",
}

var treasureDescs = []string{
	"A ornate chest sits in the center of the room, covered in dust but untouched by rust.",
	"Sunlight — impossibly — streams from above, illuminating a pedestal with a glowing object.",
	"Coin and gems spill from a cracked urn. Among them, something with a faint magical aura catches your eye.",
}

var altarDescs = []string{
	"A black stone altar dominates the room, its surface stained with something dark. In the center rests an item of power.",
	"Carved from a single piece of obsidian, the altar pulses with a faint heartbeat. An offering awaits.",
}

var merchantDescs = []string{
	"A hunched figure in a tattered cloak tends a stall of curiosities. His smile reveals too many teeth.",
	"\"Welcome, welcome!\" chirps a floating mechanical sphere, its eye glowing blue. \"Care to see my wares?\"",
}

var puzzleDescs = []string{
	"A massive stone door blocks the way forward, covered in engraved symbols. An inscription at eye level poses a question.",
	"The path is sealed by a wall of interlocking crystal gears. A riddle is carved into the frame.",
}

var restDescs = []string{
	"A small alcove offers shelter. The remains of a campfire suggest someone else found safety here before.",
	"A natural spring bubbles up through the floor, steam rising from its warm waters. The air is calm.",
}

var npcDescs = []string{
	"A figure huddles in chains against the wall, their eyes lighting up when they see you.",
	"A wounded adventurer leans against the wall, clutching a bloodied side. \"Please,\" they whisper, \"help me.\"",
}

var tombDescs = []string{
	"A stone sarcophagus dominates the chamber, its lid carved with the visage of a long-forgotten king. Dust motes dance in your torchlight.",
	"Rows of burial niches line the walls, some sealed with crumbling wax, others pried open by earlier thieves. One remains intact.",
	"A massive marble casket sits atop a raised dais. Faint runes glow along its edges, warning of danger — or promising reward.",
}

var gardenDescs = []string{
	"Bioluminescent mushrooms carpet the floor, their caps pulsing with soft light. Strange fruits hang from fibrous stalks.",
	"A patch of glowing flora thrives in the center of the cavern. Some look nourishing; others drip with ominous spores.",
	"Delicate fungal blooms sway in an unseen breeze. The air is thick with a sweet, heady aroma that makes your head spin.",
}

var anvilDescs = []string{
	"An ancient anvil sits cold and silent. Tools lie scattered around it, still in remarkable condition. A forge pit yawns nearby.",
	"The heat of long-extinguished fires lingers. A workbench holds half-finished weapon parts, and a heavy hammer rests on the anvil.",
	"Gears and chains hang from the ceiling above a sturdy anvil. A metal press stands in the corner, its handle worn smooth by use.",
}

var riftDescs = []string{
	"The air distorts and shimmers. A vertical tear in reality hangs before you, pulsing with colors that shouldn't exist.",
	"Gravity warps around a crack in the fabric of space. You feel pulled toward it — and see glimpses of other worlds within.",
	"A wound in the world, bleeding light and shadow. The void whispers through it, offering secrets that burn to know.",
}

var shrineDescs = []string{
	"A small altar of white stone stands in a quiet alcove. Fresh flowers — impossibly — lie upon it, as if placed yesterday.",
	"Carved from a single piece of marble, a shrine to an unknown deity glows with a faint inner light. Someone still tends this place.",
	"Faded icons and burnt-down candles surround a humble shrine. The air here is warm and still, untouched by the dungeon's chill.",
}

var lockedDescs = []string{
	"A heavy iron door blocks the passage, its lock intricate and well-maintained. Beyond it, you sense open space and treasure.",
	"Rusted bars seal a promising corridor. The lock mechanism looks complex but functional — a key would turn it smoothly.",
	"A reinforced door bars your way. Through a crack, you glimpse a glittering chamber beyond. The lock gleams, awaiting a key.",
}

func weightedPick(rng *rand.Rand, weights []roomWeight) RoomType {
	total := 0
	for _, w := range weights {
		total += w.Weight
	}
	roll := rng.Intn(total)
	for _, w := range weights {
		roll -= w.Weight
		if roll < 0 {
			return w.Type
		}
	}
	return weights[0].Type
}

func GenerateRoom(session *model.DelveSession, lang string) Room {
	zone := session.Zone
	seed := session.Seed + int64(session.RoomsCleared)
	rng := rand.New(rand.NewSource(seed))

	weights := zoneRoomTables[zone]
	if weights == nil {
		weights = zoneRoomTables["crypt"]
	}
	roomType := weightedPick(rng, weights)

	desc := roomDescription(roomType, zone, rng, lang)
	btns := roomButtons(roomType, rng)
	eventData := roomEventData(roomType, zone, rng, session.Floor)

	return Room{
		Type:        roomType,
		Description: desc,
		Buttons:     btns,
		EventData:   eventData,
	}
}

// VaultKeyRoom builds the tutorial's Vault Key chamber: the first room shown to
// a player who descends while on the tutorial's delve step. It lets them take
// the Vault Key and leave without risking combat in the depths.
func VaultKeyRoom(lang string) Room {
	return Room{
		Type:        RoomVaultKey,
		Description: i18n.T("delve.room.desc.vault_key", lang),
		Buttons: []RoomButton{
			{Emoji: "🔑", Label: "Take the Vault Key", Action: "key_take", Style: discordgo.SuccessButton, Data: ""},
			{Emoji: "🚪", Label: "Leave the Depths", Action: "floor_leave", Style: discordgo.SecondaryButton, Data: ""},
		},
	}
}

func roomDescription(rt RoomType, zone string, rng *rand.Rand, lang string) string {
	var key string
	switch rt {
	case RoomMonster:
		if zone == "crypt" {
			key = "crypt_monster"
		} else {
			key = fmt.Sprintf("monster_%d", rng.Intn(len(monsterDescs)))
		}
	case RoomTreasure:
		key = fmt.Sprintf("treasure_%d", rng.Intn(len(treasureDescs)))
	case RoomAltar:
		key = fmt.Sprintf("altar_%d", rng.Intn(len(altarDescs)))
	case RoomMerchant:
		key = fmt.Sprintf("merchant_%d", rng.Intn(len(merchantDescs)))
	case RoomPuzzle:
		key = fmt.Sprintf("puzzle_%d", rng.Intn(len(puzzleDescs)))
	case RoomRest:
		key = fmt.Sprintf("rest_%d", rng.Intn(len(restDescs)))
	case RoomTomb:
		key = fmt.Sprintf("tomb_%d", rng.Intn(len(tombDescs)))
	case RoomGarden:
		key = fmt.Sprintf("garden_%d", rng.Intn(len(gardenDescs)))
	case RoomForge:
		key = fmt.Sprintf("anvil_%d", rng.Intn(len(anvilDescs)))
	case RoomRift:
		key = fmt.Sprintf("rift_%d", rng.Intn(len(riftDescs)))
	case RoomShrine:
		key = fmt.Sprintf("shrine_%d", rng.Intn(len(shrineDescs)))
	case RoomLocked:
		key = fmt.Sprintf("locked_%d", rng.Intn(len(lockedDescs)))
	case RoomNPC:
		key = fmt.Sprintf("npc_%d", rng.Intn(len(npcDescs)))
	default:
		switch zone {
		case "crypt":
			key = fmt.Sprintf("crypt_%d", rng.Intn(len(cryptDescs)))
		case "fungal_wilds":
			key = fmt.Sprintf("fungal_wilds_%d", rng.Intn(len(fungalDescs)))
		case "forge_district":
			key = fmt.Sprintf("forge_district_%d", rng.Intn(len(forgeDescs)))
		case "abyss":
			key = fmt.Sprintf("abyss_%d", rng.Intn(len(abyssDescs)))
		default:
			key = "passage_default"
		}
	}
	return i18n.T("delve.room.desc."+key, lang)
}

func roomButtons(rt RoomType, rng *rand.Rand) []RoomButton {
	switch rt {
	case RoomMonster:
		return []RoomButton{
			{Emoji: "⚔️", Label: "Fight", Action: "fight", Style: discordgo.DangerButton, Data: "normal"},
			{Emoji: "🛡️", Label: "Defend", Action: "defend_start", Style: discordgo.PrimaryButton, Data: "defend"},
			{Emoji: "🏃", Label: "Flee", Action: "flee", Style: discordgo.SecondaryButton, Data: ""},
		}
	case RoomTreasure:
		return []RoomButton{
			{Emoji: "🔓", Label: "Disarm", Action: "disarm", Style: discordgo.PrimaryButton, Data: ""},
			{Emoji: "🎲", Label: "Open", Action: "open", Style: discordgo.DangerButton, Data: ""},
			{Emoji: "🚪", Label: "Leave", Action: "leave", Style: discordgo.SecondaryButton, Data: ""},
		}
	case RoomAltar:
		return []RoomButton{
			{Emoji: "💀", Label: "Sacrifice", Action: "sacrifice", Style: discordgo.DangerButton, Data: ""},
			{Emoji: "🔥", Label: "Desecrate", Action: "desecrate", Style: discordgo.DangerButton, Data: ""},
			{Emoji: "↩️", Label: "Pass", Action: "leave", Style: discordgo.SecondaryButton, Data: ""},
		}
	case RoomMerchant:
		return []RoomButton{
			{Emoji: "🛒", Label: "Browse", Action: "merchant_browse", Style: discordgo.PrimaryButton, Data: ""},
			{Emoji: "🤝", Label: "Haggle", Action: "merchant_haggle", Style: discordgo.SuccessButton, Data: ""},
			{Emoji: "🚪", Label: "Leave", Action: "leave", Style: discordgo.SecondaryButton, Data: ""},
		}
	case RoomPuzzle:
		return []RoomButton{
			{Emoji: "⌨️", Label: "Solve", Action: "puzzle_solve", Style: discordgo.PrimaryButton, Data: ""},
			{Emoji: "↩️", Label: "Skip", Action: "leave", Style: discordgo.SecondaryButton, Data: ""},
		}
	case RoomRest:
		return []RoomButton{
			{Emoji: "🔦", Label: "Use Torch", Action: "rest_torch", Style: discordgo.SuccessButton, Data: ""},
			{Emoji: "😴", Label: "Sleep", Action: "rest_sleep", Style: discordgo.PrimaryButton, Data: ""},
			{Emoji: "🩹", Label: "Bandage", Action: "rest_bandage", Style: discordgo.PrimaryButton, Data: ""},
			{Emoji: "↩️", Label: "Leave", Action: "leave", Style: discordgo.SecondaryButton, Data: ""},
		}
	case RoomTomb:
		return []RoomButton{
			{Emoji: "🪦", Label: "Open Sarcophagus", Action: "tomb_open", Style: discordgo.DangerButton, Data: ""},
			{Emoji: "🙏", Label: "Pay Respects", Action: "tomb_respect", Style: discordgo.SuccessButton, Data: ""},
			{Emoji: "↩️", Label: "Pass", Action: "leave", Style: discordgo.SecondaryButton, Data: ""},
		}
	case RoomGarden:
		return []RoomButton{
			{Emoji: "🌿", Label: "Harvest", Action: "garden_harvest", Style: discordgo.PrimaryButton, Data: ""},
			{Emoji: "🔥", Label: "Burn Garden", Action: "garden_burn", Style: discordgo.DangerButton, Data: ""},
			{Emoji: "↩️", Label: "Pass", Action: "leave", Style: discordgo.SecondaryButton, Data: ""},
		}
	case RoomForge:
		return []RoomButton{
			{Emoji: "⚒️", Label: "Temper Weapon", Action: "forge_temper", Style: discordgo.PrimaryButton, Data: ""},
			{Emoji: "🔧", Label: "Scavenge Parts", Action: "forge_scavenge", Style: discordgo.SuccessButton, Data: ""},
			{Emoji: "↩️", Label: "Pass", Action: "leave", Style: discordgo.SecondaryButton, Data: ""},
		}
	case RoomRift:
		return []RoomButton{
			{Emoji: "👁️", Label: "Gaze into the Void", Action: "rift_gaze", Style: discordgo.DangerButton, Data: ""},
			{Emoji: "🌀", Label: "Disturb the Rift", Action: "rift_disturb", Style: discordgo.PrimaryButton, Data: ""},
			{Emoji: "↩️", Label: "Pass", Action: "leave", Style: discordgo.SecondaryButton, Data: ""},
		}
	case RoomShrine:
		return []RoomButton{
			{Emoji: "🙏", Label: "Pray", Action: "shrine_pray", Style: discordgo.PrimaryButton, Data: ""},
			{Emoji: "💰", Label: "Donate Gold", Action: "shrine_donate", Style: discordgo.SuccessButton, Data: ""},
			{Emoji: "💀", Label: "Defile", Action: "shrine_defile", Style: discordgo.DangerButton, Data: ""},
			{Emoji: "↩️", Label: "Pass", Action: "leave", Style: discordgo.SecondaryButton, Data: ""},
		}
	case RoomLocked:
		return []RoomButton{
			{Emoji: "🔑", Label: "Use Key", Action: "locked_key", Style: discordgo.PrimaryButton, Data: ""},
			{Emoji: "💪", Label: "Force Door", Action: "locked_force", Style: discordgo.DangerButton, Data: ""},
			{Emoji: "↩️", Label: "Find Another Way", Action: "leave", Style: discordgo.SecondaryButton, Data: ""},
		}
	case RoomNPC:
		return []RoomButton{
			{Emoji: "🤝", Label: "Help", Action: "npc_help", Style: discordgo.SuccessButton, Data: ""},
			{Emoji: "💰", Label: "Betray", Action: "npc_betray", Style: discordgo.DangerButton, Data: ""},
			{Emoji: "💪", Label: "Intimidate", Action: "npc_intimidate", Style: discordgo.PrimaryButton, Data: ""},
			{Emoji: "🚪", Label: "Ignore", Action: "leave", Style: discordgo.SecondaryButton, Data: ""},
		}
	default:
		return navigationButtons(rng)
	}
}

func navigationButtons(rng *rand.Rand) []RoomButton {
	directions := []RoomButton{
		{Emoji: "⬅️", Label: "Go Left", Action: "nav", Style: discordgo.PrimaryButton, Data: "left"},
		{Emoji: "➡️", Label: "Go Right", Action: "nav", Style: discordgo.PrimaryButton, Data: "right"},
		{Emoji: "⬆️", Label: "Go Forward", Action: "nav", Style: discordgo.PrimaryButton, Data: "forward"},
	}
	rng.Shuffle(len(directions), func(i, j int) {
		directions[i], directions[j] = directions[j], directions[i]
	})
	return directions
}

func roomEventData(rt RoomType, zone string, rng *rand.Rand, floor int) string {
	data := map[string]any{"type": string(rt)}
	switch rt {
	case RoomMonster:
		enemy := GenerateEnemy(zone, floor, rng)
		b, _ := json.Marshal(enemy)
		data["enemy"] = string(b)
	}
	b, _ := json.Marshal(data)
	return string(b)
}

func BuildRoomEmbed(session *model.DelveSession, room *Room, lang string, svc *Service) *discordgo.MessageEmbed {
	zoneName := i18n.T("delve.room.zone."+session.Zone, lang)
	if zoneName == "delve.room.zone."+session.Zone {
		zoneName = session.Zone
	}

	char, _ := svc.store.EnsureCharacter(session.UserID)
	playerLevel := 1
	if char != nil {
		playerLevel = char.Level
	}
	danger := CalcDanger(session.Floor, playerLevel)

	title := i18n.T("delve.room.title", lang, map[string]any{
		"floor": fmt.Sprintf("%d", session.Floor),
		"zone":  zoneName,
	})
	dangerLine := DescribeDanger(danger, lang)
	desc := dangerLine + "\n\n" + room.Description

	color := 0x2ecc71
	switch {
	case danger.Skulls >= 4:
		color = 0xe74c3c
	case danger.Skulls >= 2:
		color = 0xf39c12
	}

	embed := &discordgo.MessageEmbed{
		Title:       title,
		Description: desc,
		Color:       color,
	}

	warningLine := ""
	if danger.IsPunished {
		warningLine = "\n⚠️ " + i18n.T("delve.handler.weakness_warning", lang)
	}
	if session.Torches == 0 {
		if warningLine != "" {
			warningLine += "\n🌑 " + i18n.T("delve.combat.darkness_warning", lang)
		} else {
			warningLine = "\n🌑 " + i18n.T("delve.combat.darkness_warning", lang)
		}
	}

	potionDisplay := i18n.T("delve.room.potions_line", lang, map[string]any{"potions": fmt.Sprintf("%d", session.Potions)})
	hpLine := i18n.T("delve.room.hp_line", lang, map[string]any{
		"hp":       fmt.Sprintf("%d", session.HP),
		"max_hp":   fmt.Sprintf("%d", session.MaxHP),
		"mana":     fmt.Sprintf("%d", session.Mana),
		"max_mana": fmt.Sprintf("%d", session.MaxMana),
	})
	itemsLine := i18n.T("delve.room.items_line", lang, map[string]any{
		"torches": fmt.Sprintf("%d", session.Torches),
		"keys":    fmt.Sprintf("%d", session.Keys),
		"gold":    fmt.Sprintf("%d", session.Gold),
	})
	var effects []string
	json.Unmarshal([]byte(session.StatusEffects), &effects)
	statusLine := ""
	if len(effects) > 0 {
		var displayEffects []string
		for _, e := range effects {
			statusKey := e
			if i := strings.Index(e, ":"); i > 0 {
				statusKey = e[:i]
			}
			displayEffects = append(displayEffects, i18n.T("delve.status."+statusKey, lang))
		}
		statusLine = "\n⚠️ " + strings.Join(displayEffects, ", ")
	}

	pets := svc.DeployedPets(session)
	petLine := ""
	if len(pets) > 0 {
		petLine = "\n" + i18n.T("delve.room.pets_line", lang, map[string]any{"pets": fmt.Sprintf("%d", len(pets))})
	}

	statBlock := hpLine + "\n" + potionDisplay + "  " + itemsLine + warningLine + petLine + statusLine
	embed.Fields = []*discordgo.MessageEmbedField{
		{Name: "\u200b", Value: statBlock, Inline: false},
	}

	return embed
}

func BuildRoomComponents(room *Room, lang string) []discordgo.MessageComponent {
	var rows []discordgo.MessageComponent
	var currentRow []discordgo.MessageComponent

	for _, b := range room.Buttons {
		label := i18n.T("delve.buttons."+b.Action, lang)
		if label == "delve.buttons."+b.Action {
			label = b.Label
		}
		customID := components.Encode("delve", b.Action, b.Data)
		btn := components.Button(b.Emoji+" "+label, customID, b.Style)
		currentRow = append(currentRow, btn)
		if len(currentRow) >= 3 {
			rows = append(rows, components.ActionRow(currentRow...))
			currentRow = nil
		}
	}
	if len(currentRow) > 0 {
		rows = append(rows, components.ActionRow(currentRow...))
	}
	return rows
}

func CombatRoomButtons(lang string, abilities []AbilityStatus, weaponEmoji, weaponName string) []discordgo.MessageComponent {
	var rows []discordgo.MessageComponent
	var currentRow []discordgo.MessageComponent

	for _, as := range abilities {
		a := as.Ability
		emoji := a.Emoji
		label := a.Name
		customID := components.Encode("delve", "combat_"+a.ID)

		if a.ID == "slash" {
			emoji = weaponEmoji
			label = TranslateWeaponName(weaponName, lang)
		} else {
			if tr := i18n.T("delve.abilities."+a.ID, lang); tr != "delve.abilities."+a.ID {
				label = tr
			}
		}

		style := discordgo.PrimaryButton
		switch a.ID {
		case "fireball":
			style = discordgo.DangerButton
		case "mend":
			style = discordgo.SuccessButton
		case "defend":
			style = discordgo.SecondaryButton
		}

		var btn discordgo.MessageComponent
		if as.Unlocked {
			btn = components.Button(emoji+" "+label, customID, style)
		} else {
			lockLabel := fmt.Sprintf("🔒 %s (Lv %d)", i18n.T("delve.abilities."+a.ID, lang), a.UnlockLevel)
			if lockLabel == "🔒 delve.abilities."+a.ID {
				lockLabel = fmt.Sprintf("🔒 %s (Lv %d)", label, a.UnlockLevel)
			}
			btn = components.Button(lockLabel, customID, discordgo.SecondaryButton)
		}

		currentRow = append(currentRow, btn)
		if len(currentRow) >= 3 {
			rows = append(rows, components.ActionRow(currentRow...))
			currentRow = nil
		}
	}
	if len(currentRow) > 0 {
		rows = append(rows, components.ActionRow(currentRow...))
	}

	rows = append(rows, components.ActionRow(
		components.Button("🧪 "+i18n.T("delve.buttons.combat_potion", lang), components.Encode("delve", "combat_potion"), discordgo.PrimaryButton),
		components.Button("🏃 "+i18n.T("delve.buttons.combat_flee", lang), components.Encode("delve", "combat_flee"), discordgo.SecondaryButton),
	))

	return rows
}

func MaybeAddRescueOverlay(room *Room, fallen []model.DelveSession, currentUserID int64, guildID int64, dbStore *store.Store, lang string) {
	var eligible []model.DelveSession
	for _, f := range fallen {
		if f.UserID != currentUserID {
			eligible = append(eligible, f)
		}
	}
	if len(eligible) == 0 {
		return
	}
	if rand.Intn(100) >= 30 {
		return
	}
	if len(eligible) > 2 {
		eligible = eligible[:2]
	}
	room.Description += "\n\n" + i18n.T("delve.rescue_overlay", lang)
	var btns []RoomButton
	for _, f := range eligible {
		btns = append(btns, RoomButton{
			Emoji:  "🤝",
			Label:  i18n.T("delve.room.rescue", lang, map[string]any{"user": fmt.Sprintf("<@%d>", f.UserID)}),
			Action: "rescue",
			Style:  discordgo.SuccessButton,
			Data:   fmt.Sprintf("%d", f.UserID),
		})
	}
	btns = append(btns, RoomButton{
		Emoji:  "🚫",
		Label:  i18n.T("delve.room.ignore", lang),
		Action: "ignore_fallen",
		Style:  discordgo.SecondaryButton,
		Data:   "",
	})
	room.Buttons = append(room.Buttons, btns...)
}
