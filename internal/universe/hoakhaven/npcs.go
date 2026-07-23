package hoakhaven

import "guacagamblebot/internal/universe"

var NPCs = map[string]*universe.NPCData{
	"elara": {
		ID: "elara", Name: "Elara", Emoji: "\U0001f33f", Color: 0x2ecc71,

		DescriptionEN: "Guardian of the village gardens and stables.",
		DescriptionFR: "Gardienne des jardins du village et des écuries.",

		RoleEN: "She helps with farming and pets. High affinity reduces crop growth time.",
		RoleFR: "Elle vous aide avec le farming et les pets. Une haute affinité réduit le temps de pousse des cultures.",

		AdviceEN: "Match your crops to your land! Greenhouses are perfect for tropical seeds.",
		AdviceFR: "Associez vos cultures à vos terres ! Les serres sont parfaites pour les graines tropicales.",

		ChatEN: "The sprouts are so green today. Don't forget to feed your little companions!",
		ChatFR: "Les pousses sont si vertes aujourd'hui. N'oubliez pas de nourrir vos petits compagnons !",

		HintEN: "(Loves mystery eggs, star fruits, golden apples, seeds and berries)",
		HintFR: "(Aime les œufs mystère, fruits étoiles, pommes dorées, graines et baies)",

		GreetingsEN: []string{"Hello. Take care of the land and animals.", "Happy to see you. My plants are thriving.", "Hello dear friend! Nature itself sings in your presence."},
		GreetingsFR: []string{"Bonjour. Prends soin de la terre et des animaux.", "Ravie de te voir. Mes plantes poussent à merveille.", "Bonjour cher ami ! La nature elle-même chante en ta présence."},
	},
	"thorek": {
		ID: "thorek", Name: "Thorek", Emoji: "\u26cf\ufe0f", Color: 0xe67e22,

		DescriptionEN: "Village blacksmith and miner.",
		DescriptionFR: "Forgeron et mineur du village.",

		RoleEN: "He refines metals and helps miners. High affinity reduces collapse risk.",
		RoleFR: "Il raffine les métaux et aide les mineurs. Une haute affinité réduit le risque d'effondrement.",

		AdviceEN: "Mining is about risk management. Don't dig too deep if your bag is full!",
		AdviceFR: "Miner est une question de gestion de risque. Ne creusez pas trop profond si votre sac est plein !",

		ChatEN: "Clang! Clang! Working on a new pickaxe prototype. Hard work builds character.",
		ChatFR: "Clang ! Clang ! Je travaille sur un nouveau prototype de pioche. Le travail acharné forge le caractère.",

		HintEN: "(Loves gold nuggets, diamonds, platinum and ores)",
		HintFR: "(Aime les pépites d'or, diamants, platine et minerais)",

		GreetingsEN: []string{"What do you want? If you don't have a pickaxe, you're wasting my time.", "Ah, there you are! Found any good ore lately?", "Hello my friend! My forge is always open for a hard worker like you."},
		GreetingsFR: []string{"Qu'est-ce que tu veux ? Si t'as pas de pioche, tu perds mon temps.", "Ah, te voilà ! Trouvé du bon minerai récemment ?", "Bonjour mon ami ! Ma forge est toujours ouverte pour un travailleur comme toi."},
	},
	"irian": {
		ID: "irian", Name: "Irian", Emoji: "\U0001f3a3", Color: 0x3498db,

		DescriptionEN: "Veteran fisherman and guardian of the docks.",
		DescriptionFR: "Pêcheur vétéran et gardien des quais.",

		RoleEN: "He helps fishermen catch rare creatures. High affinity extends the reaction window.",
		RoleFR: "Il aide les pêcheurs à attraper des créatures rares. Une haute affinité étend la fenêtre de réaction.",

		AdviceEN: "Be quick! Ocean fishing has a very tight window, but that's where legendary beasts live.",
		AdviceFR: "Soyez rapide ! La pêche en océan a une fenêtre très serrée, mais c'est là que vivent les bêtes légendaires.",

		ChatEN: "I'm watching the horizon... They say a giant Kraken lurks in the deep waters when the sky darkens.",
		ChatFR: "Je regarde l'horizon... On dit qu'un Kraken géant rôde dans les eaux profondes quand le ciel s'assombrit.",

		HintEN: "(Loves kraken tentacles, whales, sharks and fish)",
		HintFR: "(Aime les tentacules de kraken, baleines, requins et poissons)",

		GreetingsEN: []string{"Shh... You'll scare the fish.", "Hey sailor. Smelled the sea breeze lately?", "Ah, my captain! The tides are favorable today."},
		GreetingsFR: []string{"Chut... Tu vas effrayer les poissons.", "Hé marin. Senti la brise de mer récemment ?", "Ah, mon capitaine ! Les marées sont favorables aujourd'hui."},
	},
	"gamblebot": {
		ID: "gamblebot", Name: "GambleBot", Emoji: "\U0001f916", Color: 0xf1c40f,

		DescriptionEN: "A state-of-the-art robot dealer.",
		DescriptionFR: "Un croupier robot de pointe.",

		RoleEN: "He runs the village Casino. Improving your reputation unlocks up to 20% discount.",
		RoleFR: "Il gère le Casino du village. Améliorer votre réputation débloque jusqu'à 20% de réduction.",

		AdviceEN: "Always double down on 11 in Blackjack, but never bet more than you can afford to lose!",
		AdviceFR: "Toujours doubler sur un 11 au Blackjack, mais ne misez jamais plus que ce que vous pouvez perdre !",

		ChatEN: "Beep boop! Calculating win probability... 99% chance you should play the slot machines.",
		ChatFR: "Bip boop ! Calcul de probabilité de gain... 99% de chance que vous devriez jouer aux machines à sous.",

		HintEN: "(Loves shiny objects, rigged coins and VIP tickets)",
		HintFR: "(Aime les objets brillants, pièces truquées et tickets VIP)",

		GreetingsEN: []string{"Hello human. Have you tried your luck today?", "Good to see you. Looks like you've bet wisely.", "Hey partner! Who are we fleecing today?"},
		GreetingsFR: []string{"Bonjour humain. As-tu tenté ta chance aujourd'hui ?", "Content de te voir. On dirait que tu as misé sagement.", "Hé partenaire ! Qui allons-nous plumer aujourd'hui ?"},
	},
}
