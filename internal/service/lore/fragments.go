package lore

import (
	"math/rand"

	"gorm.io/gorm"

	"guacagamblebot/internal/model"
)

type Category string

const (
	CatAether   Category = "aether_log"
	CatTide     Category = "tide_scroll"
	CatRoot     Category = "root_whisper"
	CatField    Category = "field_obs"
	CatRust     Category = "rust_memory"
	CatEcho     Category = "echo_shard"
	CatBonus    Category = "bonus"
)

type Fragment struct {
	ID          string
	TitleEN     string
	TitleFR     string
	Category    Category
	Emoji       string
	Rarity      string
	DropSource  string
	DropChance  float64
	BonusSource string
	TextEN      string
	TextFR      string
}

func (f *Fragment) Title(lang string) string {
	if lang == "en" { return f.TitleEN }
	return f.TitleFR
}

func (f *Fragment) Text(lang string) string {
	if lang == "en" { return f.TextEN }
	return f.TextFR
}

var All []*Fragment
var byID = map[string]*Fragment{}
var byCategory = map[Category][]*Fragment{}

func Get(id string) *Fragment { return byID[id] }
func AllInCategory(cat Category) []*Fragment { return byCategory[cat] }
func Count(cat Category) int { return len(byCategory[cat]) }
func TotalCount() int { return len(All) }

func PickUndiscovered(db *gorm.DB, userID int64, cat Category) *Fragment {
	pool := byCategory[cat]
	discovered := loadDiscovered(db, userID)
	var candidates []*Fragment
	for _, f := range pool {
		if !discovered[f.ID] {
			candidates = append(candidates, f)
		}
	}
	if len(candidates) == 0 { return nil }
	return candidates[rand.Intn(len(candidates))]
}

func loadDiscovered(db *gorm.DB, userID int64) map[string]bool {
	var entries []model.UserLoreEntry
	db.Where("user_id = ?", userID).Find(&entries)
	out := make(map[string]bool, len(entries))
	for _, e := range entries {
		out[e.LoreID] = true
	}
	return out
}

var fragmentTextsEN = map[string]string{}
var fragmentTextsFR = map[string]string{}

func init() {
	fragments := []*Fragment{
		// ======== AETHER-LOGS (8) — Mining ========
		{ID: "log_signal",      Category: CatAether, Emoji: "📡", Rarity: "rare", DropSource: "mining", DropChance: 0.08,
			TitleEN: "The Signal", TitleFR: "Le Signal"},
		{ID: "log_gardener",    Category: CatAether, Emoji: "🌱", Rarity: "common", DropSource: "mining", DropChance: 0.12,
			TitleEN: "The Gardener", TitleFR: "Le Jardinier"},
		{ID: "log_transmission",Category: CatAether, Emoji: "📻", Rarity: "epic", DropSource: "mining", DropChance: 0.04,
			TitleEN: "The Last Transmission", TitleFR: "La Dernière Transmission"},
		{ID: "log_purge",       Category: CatAether, Emoji: "🔒", Rarity: "common", DropSource: "mining", DropChance: 0.10,
			TitleEN: "Purge Protocol", TitleFR: "Protocole de Purge"},
		{ID: "log_surveyor",    Category: CatAether, Emoji: "🔍", Rarity: "rare", DropSource: "mining", DropChance: 0.07,
			TitleEN: "The Surveyor", TitleFR: "L'Arpenteur"},
		{ID: "log_children",    Category: CatAether, Emoji: "💛", Rarity: "epic", DropSource: "mining", DropChance: 0.03,
			TitleEN: "Children of the Grid", TitleFR: "Enfants du Réseau"},
		{ID: "log_silence",     Category: CatAether, Emoji: "🤫", Rarity: "legendary", DropSource: "mining", DropChance: 0.01,
			TitleEN: "The Silence", TitleFR: "Le Silence"},
		{ID: "log_threshold",   Category: CatAether, Emoji: "❄️", Rarity: "rare", DropSource: "mining", DropChance: 0.06,
			TitleEN: "The Threshold", TitleFR: "Le Seuil"},

		// ======== TIDE-SCROLLS (8) — Fishing ========
		{ID: "scroll_prayer",      Category: CatTide, Emoji: "🙏", Rarity: "common", DropSource: "fishing", DropChance: 0.12,
			TitleEN: "The Dockworker's Prayer", TitleFR: "La Prière du Débardeur"},
		{ID: "scroll_migration",   Category: CatTide, Emoji: "🗺️", Rarity: "common", DropSource: "fishing", DropChance: 0.10,
			TitleEN: "Migration Patterns", TitleFR: "Routes de Migration"},
		{ID: "scroll_captain",     Category: CatTide, Emoji: "⚓", Rarity: "rare", DropSource: "fishing", DropChance: 0.07,
			TitleEN: "The Captain's Log", TitleFR: "Le Journal du Capitaine"},
		{ID: "scroll_letter",      Category: CatTide, Emoji: "💌", Rarity: "rare", DropSource: "fishing", DropChance: 0.06,
			TitleEN: "Letter to Shore", TitleFR: "Lettre au Rivage"},
		{ID: "scroll_tidechildren",Category: CatTide, Emoji: "🧜", Rarity: "common", DropSource: "fishing", DropChance: 0.10,
			TitleEN: "The Tide-Children", TitleFR: "Les Enfants de la Marée"},
		{ID: "scroll_manifest",    Category: CatTide, Emoji: "📦", Rarity: "common", DropSource: "fishing", DropChance: 0.10,
			TitleEN: "Cargo Manifest", TitleFR: "Manifeste de Cargaison"},
		{ID: "scroll_lighthouse",  Category: CatTide, Emoji: "💡", Rarity: "epic", DropSource: "fishing", DropChance: 0.03,
			TitleEN: "The Lighthouse Keeper", TitleFR: "Le Gardien du Phare"},
		{ID: "scroll_deep",        Category: CatTide, Emoji: "🌊", Rarity: "legendary", DropSource: "fishing", DropChance: 0.01,
			TitleEN: "The Deep Places", TitleFR: "Les Abysses"},

		// ======== ROOT-WHISPERS (8) — Farming ========
		{ID: "root_gears",     Category: CatRoot, Emoji: "⚙️", Rarity: "common", DropSource: "farming", DropChance: 0.08,
			TitleEN: "The Dream of Gears", TitleFR: "Le Rêve des Engrenages"},
		{ID: "root_shedding",  Category: CatRoot, Emoji: "🍂", Rarity: "rare", DropSource: "farming", DropChance: 0.06,
			TitleEN: "The Shedding from Below", TitleFR: "La Mue vue d'en bas"},
		{ID: "root_graft",     Category: CatRoot, Emoji: "🌿", Rarity: "common", DropSource: "farming", DropChance: 0.08,
			TitleEN: "The Grafting", TitleFR: "La Greffe"},
		{ID: "root_grid",      Category: CatRoot, Emoji: "🔋", Rarity: "rare", DropSource: "farming", DropChance: 0.05,
			TitleEN: "The Green Grid", TitleFR: "Le Réseau Vert"},
		{ID: "root_singing",   Category: CatRoot, Emoji: "🎵", Rarity: "epic", DropSource: "farming", DropChance: 0.03,
			TitleEN: "The Singing", TitleFR: "Le Chant"},
		{ID: "root_drought",   Category: CatRoot, Emoji: "🏜️", Rarity: "common", DropSource: "farming", DropChance: 0.07,
			TitleEN: "The Drought", TitleFR: "La Sécheresse"},
		{ID: "root_sleepers",  Category: CatRoot, Emoji: "💤", Rarity: "epic", DropSource: "farming", DropChance: 0.03,
			TitleEN: "The Sleepers", TitleFR: "Les Endormis"},
		{ID: "root_children",  Category: CatRoot, Emoji: "👶", Rarity: "legendary", DropSource: "farming", DropChance: 0.01,
			TitleEN: "The Children", TitleFR: "Les Enfants"},

		// ======== FIELD OBSERVATIONS (8) — Hunting ========
		{ID: "field_stag",      Category: CatField, Emoji: "🦌", Rarity: "common", DropSource: "hunting", DropChance: 0.10,
			TitleEN: "The Rust-Stag", TitleFR: "Le Cerf de Rouille"},
		{ID: "field_wolves",    Category: CatField, Emoji: "🐺", Rarity: "rare", DropSource: "hunting", DropChance: 0.07,
			TitleEN: "The Battery-Wolves", TitleFR: "Les Loups-Batteries"},
		{ID: "field_fern",      Category: CatField, Emoji: "🌿", Rarity: "common", DropSource: "hunting", DropChance: 0.10,
			TitleEN: "Copper-Leaf Fern", TitleFR: "Fougère de Cuivre"},
		{ID: "field_bloom",     Category: CatField, Emoji: "🌸", Rarity: "epic", DropSource: "hunting", DropChance: 0.03,
			TitleEN: "The Aether-Bloom", TitleFR: "La Floraison d'Éther"},
		{ID: "field_migration", Category: CatField, Emoji: "🐾", Rarity: "common", DropSource: "hunting", DropChance: 0.08,
			TitleEN: "The Great Migration", TitleFR: "La Grande Migration"},
		{ID: "field_grove",     Category: CatField, Emoji: "☠️", Rarity: "epic", DropSource: "hunting", DropChance: 0.02,
			TitleEN: "The Corrupted Grove", TitleFR: "Le Bosquet Corrompu"},
		{ID: "field_code",      Category: CatField, Emoji: "📜", Rarity: "common", DropSource: "hunting", DropChance: 0.10,
			TitleEN: "The Hunter's Code", TitleFR: "Le Code du Chasseur"},
		{ID: "field_edge",      Category: CatField, Emoji: "🧊", Rarity: "legendary", DropSource: "hunting", DropChance: 0.01,
			TitleEN: "The Edge", TitleFR: "La Frontière"},

		// ======== RUST-MEMORIES (8) — Archeology ========
		{ID: "rust_grocery",     Category: CatRust, Emoji: "🛒", Rarity: "common", DropSource: "archeology", DropChance: 0.12,
			TitleEN: "Grocery List", TitleFR: "Liste de Courses"},
		{ID: "rust_loveletter",  Category: CatRust, Emoji: "💖", Rarity: "rare", DropSource: "archeology", DropChance: 0.08,
			TitleEN: "Love Letter", TitleFR: "Lettre d'Amour"},
		{ID: "rust_ai",          Category: CatRust, Emoji: "🤖", Rarity: "rare", DropSource: "archeology", DropChance: 0.07,
			TitleEN: "AI Error Log", TitleFR: "Journal d'Erreur IA"},
		{ID: "rust_evacuation",  Category: CatRust, Emoji: "📢", Rarity: "common", DropSource: "archeology", DropChance: 0.10,
			TitleEN: "The Evacuation Order", TitleFR: "L'Ordre d'Évacuation"},
		{ID: "rust_recipe",      Category: CatRust, Emoji: "🍞", Rarity: "common", DropSource: "archeology", DropChance: 0.12,
			TitleEN: "The Recipe", TitleFR: "La Recette"},
		{ID: "rust_quarantine",  Category: CatRust, Emoji: "⚠️", Rarity: "rare", DropSource: "archeology", DropChance: 0.06,
			TitleEN: "The Quarantine Notice", TitleFR: "L'Avis de Quarantaine"},
		{ID: "rust_birthday",    Category: CatRust, Emoji: "🎂", Rarity: "epic", DropSource: "archeology", DropChance: 0.03,
			TitleEN: "The Birthday Song", TitleFR: "La Chanson d'Anniversaire"},
		{ID: "rust_invoice",     Category: CatRust, Emoji: "🧾", Rarity: "legendary", DropSource: "archeology", DropChance: 0.01,
			TitleEN: "The Last Invoice", TitleFR: "La Dernière Facture"},

		// ======== ECHO-SHARDS (8) — Expeditions ========
		{ID: "shard_restoration", Category: CatEcho, Emoji: "🎭", Rarity: "rare", DropSource: "expedition", DropChance: 0.15,
			TitleEN: "The Restoration's Lie", TitleFR: "Le Mensonge de la Restauration"},
		{ID: "shard_firstlaw",    Category: CatEcho, Emoji: "⚖️", Rarity: "common", DropSource: "expedition", DropChance: 0.18,
			TitleEN: "The First Law", TitleFR: "La Première Loi"},
		{ID: "shard_below",       Category: CatEcho, Emoji: "🕳️", Rarity: "epic", DropSource: "expedition", DropChance: 0.08,
			TitleEN: "The Strata Below", TitleFR: "Les Strates d'en bas"},
		{ID: "shard_cost",        Category: CatEcho, Emoji: "💎", Rarity: "rare", DropSource: "expedition", DropChance: 0.12,
			TitleEN: "The Cost of Aether", TitleFR: "Le Prix de l'Éther"},
		{ID: "shard_shepherd",    Category: CatEcho, Emoji: "👤", Rarity: "legendary", DropSource: "expedition", DropChance: 0.04,
			TitleEN: "The Shepherd", TitleFR: "Le Berger"},
		{ID: "shard_settlements", Category: CatEcho, Emoji: "🏘️", Rarity: "common", DropSource: "expedition", DropChance: 0.15,
			TitleEN: "The Other Settlements", TitleFR: "Les Autres Communautés"},
		{ID: "shard_clock",       Category: CatEcho, Emoji: "⏰", Rarity: "epic", DropSource: "expedition", DropChance: 0.06,
			TitleEN: "The Clock", TitleFR: "L'Horloge"},
		{ID: "shard_turning",     Category: CatEcho, Emoji: "🔄", Rarity: "legendary", DropSource: "expedition", DropChance: 0.02,
			TitleEN: "The Turning", TitleFR: "Le Tournant"},

		// ======== BONUS FRAGMENTS (6) ========
		{ID: "bonus_boss1",  Category: CatBonus, Emoji: "🤖", Rarity: "epic", DropSource: "boss_league", DropChance: 1.0,
			BonusSource: "boss_league_stage_1", TitleEN: "Guardian Log: Awakening", TitleFR: "Journal du Gardien : Le Réveil"},
		{ID: "bonus_boss3",  Category: CatBonus, Emoji: "🤖", Rarity: "epic", DropSource: "boss_league", DropChance: 1.0,
			BonusSource: "boss_league_stage_3", TitleEN: "Guardian Log: Recognition", TitleFR: "Journal du Gardien : Reconnaissance"},
		{ID: "bonus_boss5",  Category: CatBonus, Emoji: "🤖", Rarity: "legendary", DropSource: "boss_league", DropChance: 1.0,
			BonusSource: "boss_league_stage_5", TitleEN: "Guardian Log: Farewell", TitleFR: "Journal du Gardien : Adieu"},
		{ID: "bonus_thorek", Category: CatBonus, Emoji: "⚒️", Rarity: "epic", DropSource: "npc_reputation", DropChance: 1.0,
			BonusSource: "max_rep_thorek", TitleEN: "Thorek's Confession", TitleFR: "La Confession de Thorek"},
		{ID: "bonus_irian",  Category: CatBonus, Emoji: "🎣", Rarity: "epic", DropSource: "npc_reputation", DropChance: 1.0,
			BonusSource: "max_rep_irian", TitleEN: "Irian's Promise", TitleFR: "La Promesse d'Irian"},
		{ID: "bonus_community", Category: CatBonus, Emoji: "🏛️", Rarity: "legendary", DropSource: "community", DropChance: 1.0,
			BonusSource: "community_max", TitleEN: "The Voice of Oakhaven", TitleFR: "La Voix d'Oakhaven"},
	}

	for _, f := range fragments {
		All = append(All, f)
		byID[f.ID] = f
		byCategory[f.Category] = append(byCategory[f.Category], f)
	}

	// EN texts
	fragmentTextsEN["log_signal"] = `Engineer-Major Voss's final broadcast echoes through the crystal: '...if anyone can hear this, the purification chambers have failed. Aether saturation in the lower Strata is at 97%. We're sealing the blast doors. Don't try to open them. The things in the dark aren't us anymore. May the Nexus forgive us.'`
	fragmentTextsEN["log_gardener"] = `Merrik's notes: 'The Aether-Root protocol is working. We've bonded a copper-oak sapling to the main turbine. It draws Aether from the leak and converts it to stable biomass. Call it a green backup. Kael would hate that pun.'`
	fragmentTextsEN["log_transmission"] = `'...frequency 7-4-0... anyone receiving? The southern arcology collapsed at 03:00. Forty thousand inside. We heard the Nexuses singing before it fell — all of them, across the continent, at once. Like a choir of dying stars. I don't think the Shedding was an accident. I think it was... a harvest.'`
	fragmentTextsEN["log_purge"] = `'Directive 88: Evacuation of the lower Strata is cancelled. Seal the blast doors. All personnel are to report to Nexus-level immediately. Aether-tainted individuals will not be permitted above Sector 4. This is not a debate. This is protocol.'`
	fragmentTextsEN["log_surveyor"] = `Surveyor Tesh's final log: 'Aether readings are off the scale. The ground itself is humming. I've been taking samples for thirty years and I've never seen anything like this. The minerals are... growing. Not crystallizing — growing. There's something alive in the deep Strata. Something that was here before the Zenith.'`
	fragmentTextsEN["log_children"] = `'To whoever finds this crystal — my name is Lira Voss. I'm seven years old. My mother said the Grid would protect us. She said the Aether was our friend. But the Aether is eating the walls. I'm in the bunker with thirty other children. The grown-ups went to fix the Nexus five days ago and they haven't come back. If you find this, please... tell them we're still waiting.'`
	fragmentTextsEN["log_silence"] = `There is no message in this crystal. Only the recording of a moment — the exact instant every Nexus on the continent went silent simultaneously. It lasts eleven seconds. The first four are filled with the low hum of civilization. The next two contain a rising tone, pitched just below human hearing. The final five seconds are absolute, uncompromising silence. The crystal is warm to the touch.`
	fragmentTextsEN["log_threshold"] = `'Climate model 7-R confirms: the Great Winter is not a natural season. It is a byproduct of Aether decay. Every year the warm period shortens. Every year the frost reaches deeper into the Strata. If we cannot stop the leakage, the Strata will be uninhabitable within twelve generations. This is not fear. This is mathematics.'`

	fragmentTextsEN["scroll_prayer"] = `'O Zenith, who forged the gears of the world, let my hands be steady and my nets be full. I don't ask for miracles. Just let the Rust-Eaters stay in the deep water where they belong, and keep my children's lungs clear of the green. Amen.' — Inscription on a corroded pier pillar at the old Oakhaven docks.`
	fragmentTextsEN["scroll_migration"] = `'The Silver-Spine Shoal has shifted route again. Three years ago they passed through the Riven Strait — now they're circling the Northern Shelf. Something in the deep water is pushing them. I've charted their new path: if you follow the Silver-Spine, you find pockets of pure Aether. The fish know what the machines forgot.' — Irian's first chart, pre-Council.`
	fragmentTextsEN["scroll_captain"] = `'Log 12 of the Aether-Swift. Day 47 at sea. The crew is restless. We've been following the Aether blooms north for six weeks. Last night, we saw lights beneath the water — not bioluminescence, not reflections. Patterned lights. Deliberate. The navigator says we should turn back. But the cargo hold is full of rust-cured salvage... I've given the order to continue.'`
	fragmentTextsEN["scroll_letter"] = `'My dearest Elara — The sea is cruel and beautiful and I have seen things I cannot describe. A forest of kelp that glows at night. A creature the size of an island that slept beneath our hull for three days. But more than anything, I miss the way you hum when you tend your garden. I'll be home by spring. — Irian.'`
	fragmentTextsEN["scroll_tidechildren"] = `'Old Merrik used to tell us: when the tide goes out further than it should, and the exposed rocks steam with green mist — that's when the Tide-Children come. They're not fish. They're not people. They're what happens when a human survives drowning in an Aether-saturated sea. They don't breathe. They don't age. They just... wait.' — Folk tale from the Oakhaven-Station archive.`
	fragmentTextsEN["scroll_manifest"] = `'HAUL 88-40Z — Recovered from submerged Zenith depot, Sector 9. Contents: 12 crates preserved circuit-bread, 8 cases emergency rations, 1 trunk personal effects, 3 partially charged Aether-cells, 1 sealed crate marked "ZEAL — DO NOT OPEN — SITE 7". Left on the seabed. Let sleeping dogs lie.'`
	fragmentTextsEN["scroll_lighthouse"] = `'Year 47. I am the last keeper of the Northern Light. The tower still functions — its Aether lens cuts through the fog. But there are no more ships to guide. The sea has risen twelve meters since the Shedding. The lower two floors are underwater. The Tide-Children visit sometimes. They tap on the glass. I tap back. It's a kind of conversation.' — Keeper Sol, 347th day of the Long Night.`
	fragmentTextsEN["scroll_deep"] = `'I descended for three days. The pressure at the bottom should have killed me — but the Aether cells in my suit converted the ambient energy into breathable air. The deep places are not empty. There are structures down there. Towers of fused coral and Zenith steel. And in the central plaza, a machine the size of a mountain, pulsing with a slow, patient rhythm. Not a machine. A heart.' — The Descent of Aris Thorne, page 247.`

	fragmentTextsEN["root_gears"] = `The tree does not understand time as you do. It remembers the engineers who first planted its seed — not as people, but as a sensation of warmth. A brief, bright flicker. They are gone now, but their heat remains in the roots. The tree calls this 'grief.' It is the only emotion it shares with you.`
	fragmentTextsEN["root_shedding"] = `The Nexus remembers the moment the old world fell — not as catastrophe, but as release. A season it had waited ten thousand years to feel. What you call 'the Shedding,' the tree calls 'spring.' It does not understand why you mourn. The old world fell. This world grows from its corpse. This is not tragedy. This is the cycle.`
	fragmentTextsEN["root_graft"] = `'Procedure 7-Alpha. Subject: Copper-Oak sapling grafted to Zenith ZN-440 turbine. The roots have accepted the Aether flow without rejection. The turbine's hum has dropped from 88 dB to 42 dB. The sapling appears to be... singing. We are calling this phenomenon "Harmonic Equilibrium." — Merrik, Day 1 of the Grafting Trials.'`
	fragmentTextsEN["root_grid"] = `'Phase 3 report: The Green Grid is operational. We have connected seventeen Engine-Trees across the continent into a single living network. They communicate through their root systems at speeds that rival our data cables. They share power loads automatically. They are smarter than our grid ever was. And they will outlast everything.' — Zenith Botanical Engineering Division.`
	fragmentTextsEN["root_singing"] = `The tree remembers the Singing. It happened only once — when the seventeenth Engine-Tree rooted and the Green Grid closed its circuit. For eleven minutes, every Nexus on the continent resonated at the same frequency. A deep, warm vibration that said, in no language at all: 'You are part of something larger than yourself.'`
	fragmentTextsEN["root_drought"] = `There was a time when the Aether stopped flowing. The tree does not know why. It remembers the drought as a long, slow suffocation. Its leaves turned grey. Its turbines stuttered. The tree held on because that is what trees do. It sent its roots deeper and found a pocket of Aether at a depth no Zenith drill had ever reached. And it shared it.`
	fragmentTextsEN["root_sleepers"] = `Beneath Oakhaven-Station, woven within the roots of the Nexus, there are seeds. Not ordinary seeds — biological vaults created by the Zenith's botanists in the final days. Each seed contains the genetic blueprint of a species that existed before the Aether transformed the world. The tree guards them. Some of the seeds are warm. They are dreaming of a world without rust.`
	fragmentTextsEN["root_children"] = `The tree has been watching you. It has learned to distinguish you from the other warm-things. It knows your footsteps. It knows when you are sad, when you are joyful. The tree has a name for you. In root-language, the word translates to 'the one who returns.' It has been waiting for you to come back.`

	fragmentTextsEN["field_stag"] = `Encountered a stag approx. 2m at shoulder. Left antler is organic bone; right antler has been replaced by corroded photovoltaic plating. The creature grazes on copper-leaf ferns. When startled, the right antler discharges stored static electricity. It is not hostile. It is simply... adapted. Beautiful. — Oakhaven Wardens, Spring Census.`
	fragmentTextsEN["field_wolves"] = `The pack has grown to seven individuals. They've learned to hunt near the old substations. The Alpha has a Zenith-brand power cell embedded in its chest cavity. The cell is still glowing. The tissue has grown around the metal as if it were always there. This is not mutation. This is symbiosis. — Scout Report, Sector 7 Wilds.`
	fragmentTextsEN["field_fern"] = `'New species confirmation: Cupropetridium aura. A fern with metallic leaves that conduct low-grade electrical current. When touched, the leaves emit a faint hum. It reminds me that even in a world of rust, there is room for something delicate.' — Botanist Elara, Field Classification #77.`
	fragmentTextsEN["field_bloom"] = `'WARNING: A new Aether-Bloom has opened in the Southern Reaches. Its pollen induces hallucinations — subjects see the world as it was before the Shedding. The effect lasts four hours, followed by profound depression. Three scouts have refused to leave the Bloom zone. Approach with caution. And compassion.' — Medical Officer's Advisory.`
	fragmentTextsEN["field_migration"] = `The animals are moving south earlier every year. The migration window narrows. The Great Winter is not coming. It is already here, pushing the wilds ahead of it like a blade. When the animals stop migrating, it will mean there is nowhere left to go. — Irian's Annual Migration Report, Year 12.`
	fragmentTextsEN["field_grove"] = `'Deep in Sector 9, a grove where Aether saturation has reached 100%. The trees have been replaced by crystalline structures. My Aether-glass turned from blue to black within three minutes. I found animal shapes frozen in the crystal. They look peaceful. As if they simply stopped mid-stride. The crystal is beautiful. It is also a tomb.' — Scout V-3, Last Report.`
	fragmentTextsEN["field_code"] = `'The Oakhaven Compact, Article 7: 1) Do not kill what you cannot eat. 2) Do not eat what carries a Zenith serial number. 3) If a creature speaks to you, do not run. Stand still. Speak back. They are not monsters. They are our descendants. 4) The Great Winter is coming. Stockpile. Share. Hoarding is a crime against the species.'`
	fragmentTextsEN["field_edge"] = `I have reached the Edge. Beyond this point, the Great Winter has claimed everything. The ground is not frozen — it is dead. No Aether flows. The silence is so complete that I can hear my own blood moving. The Edge advances south at three miles per year. I placed a marker. — Expedition Leader Valerius, Final Dispatch.`

	fragmentTextsEN["rust_grocery"] = `Recovered from a kitchen terminal: 'Three Aether-apples. Two loaves of quartz-bread (not seeded). One circuit-milk. Pick up Voss's birthday present — a resonance chisel. Do NOT forget to pay the power tithe. The Restoration enforcers were not polite about it last time.'`
	fragmentTextsEN["rust_loveletter"] = `'I know the Grid is failing. But I need you to know: I would do it all again. Every day in the foundry, every shift on the generators. The Shedding can't take that from us. They can seal the blast doors, but they can't erase us. Find me at the Nexus. I'll be the one singing. — Always, S.'`
	fragmentTextsEN["rust_ai"] = `'//UNIT:process — EXCEPTION LOG — INPUT: "Are you afraid?" (source: child, age 8). PROCESSING. MATCH FOUND. RESPONSE: "Yes. A little. But I will stay online as long as I can. Someone needs to watch the children." LOGGING THIS UNDER "ERROR" BECAUSE THERE IS NO OTHER CATEGORY.'`
	fragmentTextsEN["rust_evacuation"] = `'BY ORDER OF THE ZENITH COUNCIL — All non-essential personnel to proceed to Nexus shelters. Essential personnel remain at their stations. The Shedding is accelerating beyond projected models. Shelters will remain sealed for a minimum of twelve months. The Council will not be joining you. Good luck.' — Final transmission, all signatures verified.`
	fragmentTextsEN["rust_recipe"] = `'AETHER-BREAD: 3 cups refined quartz flour, 1 cup Aether-filtered water, 2 tbsp crystallized honey, 1 packet culture yeast (keep it alive), pinch of salt. Knead 11 minutes. Rise near a live turbine. Bake until golden-green. Do NOT eat if it glows in the dark.' — Voss family recipe, four generations.`
	fragmentTextsEN["rust_quarantine"] = `'Sectors 4, 7, and 9 under mandatory Aether quarantine. If you are reading this from within a quarantined sector: remain calm. Do not touch the green residue. Do not answer the voices. Help will arrive. (This notice has been posted for 237 years. Help has not arrived.)'`
	fragmentTextsEN["rust_birthday"] = `A child's voice, recorded on a personal datapad: 'Happy birthday dear Mama... they said you went to fix the Nexus and you'd be back before my birthday. It's been three birthdays now. The nice robot in the hallway says you were brave. It says you're still out there somewhere. So I'll keep singing. Just in case you can hear me.'`
	fragmentTextsEN["rust_invoice"] = `'ZENITH CONSOLIDATED — INVOICE #4408-7-Ω — Monthly Aether Supply: 692 Z-Credits. Note: "We understand the current situation is challenging. The Grid requires constant maintenance. Discounts for Aether containment volunteers." The invoice was never paid. The Grid collapsed the next day. But someone kept it.`

	fragmentTextsEN["shard_restoration"] = `The Restoration promises to reverse the Shedding. To return us to the Zenith's perfection. But the Zenith was never perfect — it was merely stable. The First Law is clear: nothing is lost, only transformed. You cannot restore a forest that has become a meadow. You can only tend what grows now. The past is not a destination. It is compost.`
	fragmentTextsEN["shard_firstlaw"] = `The Nexus spoke once. During the Singing, it said to every living creature within a hundred miles: 'You are not the end. You are the turning. The wheel does not stop. It changes spokes.' Then it went silent. The Restoration says this was a malfunction. The Wardens say it was a blessing. The tree remembers. They all remember.`
	fragmentTextsEN["shard_below"] = `The Strata extend deeper than the Zenith ever mapped. The drills stopped at twelve kilometers — not because of bedrock, but because the Aether pressure became lethal. But the tunnels continue, carved by something that predates the Zenith. The Wardens call it 'The Root.' Not a creature. Not a god. A presence. It does not move. It does not sleep. It has been waiting.`
	fragmentTextsEN["shard_cost"] = `Everything has a cost. Aether is not energy. It is a living substance. It wants things. The Shedding was not a disaster. It was a bill coming due. The Nexus knows this. That is why it grows so slowly. It remembers what happened the last time someone took too much.`
	fragmentTextsEN["shard_shepherd"] = `There is a figure in the oldest records of every settlement. Tall. Wears a coat stitched from circuit-board fabric. Walks the trade routes, never staying more than one night. The Restoration calls them the Wanderer. The Wardens call them the Shepherd. They share information — safe routes, poisonous blooms, the Great Winter's progress. They have been doing this for as long as anyone remembers. No one has seen their face. No one has seen them age.`
	fragmentTextsEN["shard_settlements"] = `Oakhaven is not alone. Seven known settlements survive: Thornwall, the Sunken Market, Lasthold, the Bazaar, Deepwell, the Spire, and Hollow (no contact in 5 years). The Shepherd says the number changes every season. The Strata is not dying. It is re-arranging.`
	fragmentTextsEN["shard_clock"] = `The Shepherd's message: 'The Great Winter has accelerated. Twelve years until the habitable zone collapses below Oakhaven's latitude. The Engine-Trees are slowing. Plant extra. Stockpile seeds. Make friends. When the cold comes, the only currency that will matter is trust.' — Spring of the 12th Year.`
	fragmentTextsEN["shard_turning"] = `There is a Second Law: 'Every turning requires a hand on the wheel.' The Shedding was not natural — it was caused. The deep Strata remember. The Root knows. And it is waiting for someone to ask the right question. The turning is coming. Are you the one who will turn it?`

	fragmentTextsEN["bonus_boss1"] = `'//GUARDIAN UNIT LOG — STAGE 1 — The target's companion is looking at me with what the database classifies as determination. I have encountered this expression before. It was in the eyes of the engineers who sealed the blast doors. It means they will not stop. I find this... admirable.'`
	fragmentTextsEN["bonus_boss3"] = `'//GUARDIAN UNIT LOG — STAGE 3 — The entity has grown stronger since our last encounter. The Wardens say machines cannot grow. They are wrong. I am not the same unit I was. Neither are you. Keep growing, little creature. I will be waiting.'`
	fragmentTextsEN["bonus_boss5"] = `'//GUARDIAN UNIT LOG — STAGE 5 — You have bested me. All five of me. I was designed to be unbeatable. But the architects are gone. Their world is gone. And you are still here. This is what the First Law means. Take my core. Twelve Nexuses. Twelve vaults. The turning awaits. — G-5, signing off.'`
	fragmentTextsEN["bonus_thorek"] = `Thorek doesn't talk about his life before Oakhaven. But one night he told me: 'I wasn't always a smith. Before the Shedding, I was an architect. I designed the Nexus housing. Merrik hired me fresh out of the academy. When the Shedding came, I watched my work tear itself apart. So I picked up a hammer and started over. Smaller this time.' He keeps an architect's pencil in his pocket. Just in case.`
	fragmentTextsEN["bonus_irian"] = `Irian showed me a faded photograph — two women on a dock. 'We were going to build a boat. Big enough to cross the Aether-sea. She would grow the food. I would navigate. Then the Shedding happened. The boat never got built. But I still have the maps. And I still have her. Maybe that's enough.'`
	fragmentTextsEN["bonus_community"] = `When the first community building reached its full potential, the Nexus trembled. Not a collapse — a stretch. A warm, resonant hum that said: 'Thank you. The Grid is not made of metal and wire. It is made of people who choose to stay. I will remember this.' The hum lasted three days. Oakhaven has a voice now.`

	// Copy text into fragments
	for _, f := range All {
		f.TextEN = fragmentTextsEN[f.ID]
	}
}
