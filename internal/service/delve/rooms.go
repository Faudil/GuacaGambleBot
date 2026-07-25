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
		{RoomMonster, 30}, {RoomTreasure, 15}, {RoomAltar, 5},
		{RoomMerchant, 8}, {RoomPuzzle, 10}, {RoomRest, 12},
		{RoomNPC, 5}, {RoomEmpty, 15},
	},
	"fungal_wilds": {
		{RoomMonster, 28}, {RoomTreasure, 12}, {RoomAltar, 8},
		{RoomMerchant, 10}, {RoomPuzzle, 8}, {RoomRest, 10},
		{RoomNPC, 8}, {RoomEmpty, 16},
	},
	"forge_district": {
		{RoomMonster, 25}, {RoomTreasure, 18}, {RoomAltar, 10},
		{RoomMerchant, 12}, {RoomPuzzle, 12}, {RoomRest, 8},
		{RoomNPC, 5}, {RoomEmpty, 10},
	},
	"abyss": {
		{RoomMonster, 35}, {RoomTreasure, 10}, {RoomAltar, 15},
		{RoomMerchant, 5}, {RoomPuzzle, 5}, {RoomRest, 5},
		{RoomNPC, 10}, {RoomEmpty, 15},
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

func pickRandom[T any](items []T, rng *rand.Rand) T {
	return items[rng.Intn(len(items))]
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

	desc := roomDescription(roomType, zone, rng)
	btns := roomButtons(roomType, rng)
	eventData := roomEventData(roomType, zone, rng, session.Floor)

	return Room{
		Type:        roomType,
		Description: desc,
		Buttons:     btns,
		EventData:   eventData,
	}
}

func roomDescription(rt RoomType, zone string, rng *rand.Rand) string {
	var desc string
	switch rt {
	case RoomMonster:
		if zone == "crypt" {
			desc = "Bones rattle in the corners as a shambling figure rises to block your path."
		} else {
			desc = pickRandom(monsterDescs, rng)
		}
	case RoomTreasure:
		desc = pickRandom(treasureDescs, rng)
	case RoomAltar:
		desc = pickRandom(altarDescs, rng)
	case RoomMerchant:
		desc = pickRandom(merchantDescs, rng)
	case RoomPuzzle:
		desc = pickRandom(puzzleDescs, rng)
	case RoomRest:
		desc = pickRandom(restDescs, rng)
	case RoomNPC:
		desc = pickRandom(npcDescs, rng)
	default:
		switch zone {
		case "crypt":
			desc = pickRandom(cryptDescs, rng)
		case "fungal_wilds":
			desc = pickRandom(fungalDescs, rng)
		case "forge_district":
			desc = pickRandom(forgeDescs, rng)
		case "abyss":
			desc = pickRandom(abyssDescs, rng)
		default:
			desc = "The passage stretches ahead into darkness."
		}
	}
	return desc
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
			{Emoji: "↩️", Label: "Leave", Action: "leave", Style: discordgo.SecondaryButton, Data: ""},
		}
	case RoomNPC:
		return []RoomButton{
			{Emoji: "🤝", Label: "Help", Action: "npc_help", Style: discordgo.SuccessButton, Data: ""},
			{Emoji: "💰", Label: "Betray", Action: "npc_betray", Style: discordgo.DangerButton, Data: ""},
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
	zoneDisplay := map[string]string{
		"crypt":          "Crypt",
		"fungal_wilds":   "Fungal Wilds",
		"forge_district": "Forge District",
		"abyss":          "The Abyss",
	}

	zoneName := zoneDisplay[session.Zone]
	if zoneName == "" {
		zoneName = session.Zone
	}

	title := fmt.Sprintf("🧱 The Undercroft · Floor %d · %s", session.Floor, zoneName)
	desc := room.Description

	embed := &discordgo.MessageEmbed{
		Title:       title,
		Description: desc,
		Color:       0x2ecc71,
	}

	hpLine := fmt.Sprintf("⚔️ HP: %d/%d    🔥 Mana: %d/%d", session.HP, session.MaxHP, session.Mana, session.MaxMana)
	itemsLine := fmt.Sprintf("🔦 Torches: %d    🗝️ Keys: %d    💰 Gold: %d", session.Torches, session.Keys, session.Gold)
	var effects []string
	json.Unmarshal([]byte(session.StatusEffects), &effects)
	statusLine := ""
	if len(effects) > 0 {
		statusLine = "\n⚠️ " + strings.Join(effects, ", ")
	}

	pets := svc.DeployedPets(session)
	petLine := ""
	if len(pets) > 0 {
		petLine = fmt.Sprintf("\n🐾 Pets deployed: %d", len(pets))
	}

	embed.Fields = []*discordgo.MessageEmbedField{
		{Name: "\u200b", Value: hpLine + "\n" + itemsLine + petLine + statusLine, Inline: false},
	}

	return embed
}

func BuildRoomComponents(room *Room) []discordgo.MessageComponent {
	var rows []discordgo.MessageComponent
	var currentRow []discordgo.MessageComponent

	for _, b := range room.Buttons {
		label := b.Emoji + " " + i18n.T("delve.buttons."+b.Action, "en")
		if label == "delve.buttons."+b.Action {
			label = b.Emoji + " " + b.Label
		}
		customID := components.Encode("delve", b.Action, b.Data)
		btn := components.Button(b.Emoji+" "+b.Label, customID, b.Style)
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

func CombatRoomButtons() []discordgo.MessageComponent {
	return []discordgo.MessageComponent{
		components.ActionRow(
			components.Button("⚔️ Slash", components.Encode("delve", "combat_slash"), discordgo.PrimaryButton),
			components.Button("🔥 Fireball", components.Encode("delve", "combat_fireball"), discordgo.DangerButton),
			components.Button("🛡️ Defend", components.Encode("delve", "combat_defend"), discordgo.SuccessButton),
		),
		components.ActionRow(
			components.Button("🧪 Use Potion", components.Encode("delve", "combat_potion"), discordgo.PrimaryButton),
			components.Button("🏃 Flee", components.Encode("delve", "combat_flee"), discordgo.SecondaryButton),
		),
	}
}

func MaybeAddRescueOverlay(room *Room, fallen []model.DelveSession, currentUserID int64, guildID int64, dbStore *store.Store) {
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
	room.Description += "\n\n*A faint voice calls from the shadows... someone needs help!*"
	var btns []RoomButton
	for _, f := range eligible {
		btns = append(btns, RoomButton{
			Emoji:  "🤝",
			Label:  fmt.Sprintf("Rescue <@%d> (-1🔥)", f.UserID),
			Action: "rescue",
			Style:  discordgo.SuccessButton,
			Data:   fmt.Sprintf("%d", f.UserID),
		})
	}
	btns = append(btns, RoomButton{
		Emoji:  "🚫",
		Label:  "Ignore them",
		Action: "ignore_fallen",
		Style:  discordgo.SecondaryButton,
		Data:   "",
	})
	room.Buttons = append(room.Buttons, btns...)
}
