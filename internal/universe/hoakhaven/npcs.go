package hoakhaven

import "guacagamblebot/internal/universe"

var NPCs = map[string]*universe.NPCData{
	"elara": {
		ID: "elara", Name: "Elara", Emoji: "\U0001f33f", Color: 0x2ecc71,
		LinkedActivities: []string{"farming", "pets"},

		DescriptionEN: "Guardian of the village gardens and stables.",
		DescriptionFR: "Gardienne des jardins du village et des écuries.",

		RoleEN: "She helps with farming and pets. High affinity reduces crop growth time.",
		RoleFR: "Elle vous aide avec le farming et les pets. Une haute affinité réduit le temps de pousse des cultures.",

		AdviceEN: "Match your crops to your land! Greenhouses are perfect for tropical seeds.",
		AdviceFR: "Associez vos cultures à vos terres ! Les serres sont parfaites pour les graines tropicales.",

		ChatEN: "The sprouts are so green today. Don't forget to feed your little companions!",
		ChatFR: "Les pousses sont si vertes aujourd'hui. N'oubliez pas de nourrir vos petits compagnons !",

		HintEN: "forest_egg,star_fruit,golden_apple,seeds,berries",
		HintFR: "forest_egg,star_fruit,golden_apple,seeds,berries",

		GreetingsEN: []string{"Hello. Take care of the land and animals.", "Happy to see you. My plants are thriving.", "Hello dear friend! Nature itself sings in your presence."},
		GreetingsFR: []string{"Bonjour. Prends soin de la terre et des animaux.", "Ravie de te voir. Mes plantes poussent à merveille.", "Bonjour cher ami ! La nature elle-même chante en ta présence."},

		QuipsEN: []string{
			"Water carries memories, have you noticed? A raindrop is older than any of us.",
			"I saw a butterfly with wings like stained glass over by the greenhouse.",
			"Have you tried planting in the Greenhouse yet? The yields are amazing.",
			"My stables are full but there's always room for one more companion!",
			"The earth smells rich today. A good sign for the coming harvest.",
			"The sunflowers turned their heads the moment you arrived. They know.",
			"The flowers bloomed at midnight , synchronized, like they were communicating.",
		},
		QuipsFR: []string{
			"Les pousses sont si vertes aujourd'hui. N'oublie pas de les arroser !",
			"J'ai vu un papillon aux ailes de vitrail près de la serre.",
			"As-tu essayé de planter dans la Serre ? Les rendements sont incroyables.",
			"Mes écuries sont pleines mais il y a toujours de la place pour un compagnon de plus !",
			"La terre sent bon aujourd'hui. Bon signe pour la récolte à venir.",
			"Les tournesols se sont tournés au moment exact où tu es arrivé. Ils savent.",
			"Les fleurs ont fleuri à minuit , synchronisées, comme si elles communiquaient.",
		},

		QuipsHighEN: []string{
			"Sometimes I think the plants understand me better than people ever could.",
			"A wild pet came by last night. I left it some berries , it seemed grateful.",
			"I found a seed that glows in the dark. I've never seen anything like it.",
			"The ancient trees whisper secrets when the wind blows from the east.",
		},
		QuipsHighFR: []string{
			"Parfois je pense que les plantes me comprennent mieux que les gens.",
			"Un animal sauvage est passé hier soir. Je lui ai laissé des baies , il semblait reconnaissant.",
			"J'ai trouvé une graine qui brille dans le noir. Je n'ai jamais rien vu de pareil.",
			"Les arbres anciens murmurent des secrets quand le vent souffle de l'est.",
		},

		ShopItems: []universe.ShopItem{
			{ItemID: "basic_seeds", MinLevel: 1, RepCost: 10, CoinCost: 50, Emoji: "🌱", LabelEN: "Basic Seeds", LabelFR: "Graines de Base"},
			{ItemID: "forest_egg", MinLevel: 2, RepCost: 50, CoinCost: 500, Emoji: "🥚", LabelEN: "Forest Egg", LabelFR: "Œuf de Forêt"},
			{ItemID: "star_fruit", MinLevel: 3, RepCost: 100, CoinCost: 2000, Emoji: "⭐", LabelEN: "Star Fruit", LabelFR: "Fruit Étoile"},
			{ItemID: "golden_apple", MinLevel: 4, RepCost: 200, CoinCost: 5000, Emoji: "🍎", LabelEN: "Golden Apple", LabelFR: "Pomme Dorée"},
			{ItemID: "phoenix_berry", MinLevel: 5, RepCost: 500, CoinCost: 20000, Emoji: "🔴", LabelEN: "Phoenix Berry", LabelFR: "Baie de Phénix"},
		},
	},
	"thorek": {
		ID: "thorek", Name: "Thorek", Emoji: "\u26cf\ufe0f", Color: 0xe67e22,
		LinkedActivities: []string{"mining"},

		DescriptionEN: "Village blacksmith and miner.",
		DescriptionFR: "Forgeron et mineur du village.",

		RoleEN: "He refines metals and helps miners. High affinity reduces collapse risk.",
		RoleFR: "Il raffine les métaux et aide les mineurs. Une haute affinité réduit le risque d'effondrement.",

		AdviceEN: "Mining is about risk management. Don't dig too deep if your bag is full!",
		AdviceFR: "Miner est une question de gestion de risque. Ne creusez pas trop profond si votre sac est plein !",

		ChatEN: "Clang! Clang! Working on a new pickaxe prototype. Hard work builds character.",
		ChatFR: "Clang ! Clang ! Je travaille sur un nouveau prototype de pioche. Le travail acharné forge le caractère.",

		HintEN: "gold_nugget,diamond,platinum,ores",
		HintFR: "gold_nugget,diamond,platinum,ores",

		GreetingsEN: []string{"What do you want? If you don't have a pickaxe, you're wasting my time.", "Ah, there you are! Found any good ore lately?", "Hello my friend! My forge is always open for a hard worker like you."},
		GreetingsFR: []string{"Qu'est-ce que tu veux ? Si t'as pas de pioche, tu perds mon temps.", "Ah, te voilà ! Trouvé du bon minerai récemment ?", "Bonjour mon ami ! Ma forge est toujours ouverte pour un travailleur comme toi."},

		QuipsEN: []string{
"Temperature's right. Folded this steel nine times so far. Shooting for twelve.",
      "The forge fire is extra hot today. Must be the humidity.",
			"Don't touch that anvil until it cools. Learned that one the hard way.",
			"I've been mining these tunnels for thirty years. Still surprises me.",
			"A pickaxe is only as good as the arm swinging it.",
			"You break it, you forge it again. That's how you learn.",
			"Back in my day we didn't have fancy drills. Just sweat and iron.",
		},
		QuipsFR: []string{
"Bonne température. J'ai plié cet acier neuf fois. Je vise les douze.",
			"Le feu de la forge est extra chaud aujourd'hui. C'est l'humidité.",
			"Ne touche pas cette enclume avant qu'elle refroidisse. Je l'ai appris à mes dépens.",
			"Je creuse ces tunnels depuis trente ans. Ça me surprend encore.",
			"Une pioche est seulement aussi bonne que le bras qui la balance.",
			"Tu le casses, tu le reforges. C'est la règle dans mon atelier.",
			"À mon époque, on n'avait pas de perceuses sophistiquées. Juste de la sueur et du fer.",
		},

		QuipsHighEN: []string{
			"There's a vein of crystal deep below the seventh shaft. I've never told anyone.",
			"You've got the touch. Reminds me of myself at your age.",
			"The mountain speaks to those who listen. Most people just hear rocks.",
			"I forged a blade once that could cut starlight. Lost it in the lower depths.",
		},
		QuipsHighFR: []string{
			"Il y a un filon de cristal profond sous le septième puits. Je ne l'ai jamais dit à personne.",
			"Tu as le coup de main. Ça me rappelle moi à ton âge.",
			"La montagne parle à ceux qui écoutent. La plupart n'entendent que des pierres.",
			"J'ai forgé une lame qui pouvait couper la lumière des étoiles. Perdue dans les profondeurs.",
		},

		ShopItems: []universe.ShopItem{
			{ItemID: "iron_ore", MinLevel: 1, RepCost: 10, CoinCost: 100, Emoji: "🪨", LabelEN: "Iron Ore", LabelFR: "Minerai de Fer"},
			{ItemID: "gold_nugget", MinLevel: 2, RepCost: 40, CoinCost: 1000, Emoji: "✨", LabelEN: "Gold Nugget", LabelFR: "Pépite d'Or"},
			{ItemID: "diamond", MinLevel: 3, RepCost: 100, CoinCost: 5000, Emoji: "💎", LabelEN: "Diamond", LabelFR: "Diamant"},
			{ItemID: "platinum", MinLevel: 4, RepCost: 200, CoinCost: 10000, Emoji: "🔘", LabelEN: "Platinum", LabelFR: "Platine"},
			{ItemID: "ancient_relic", MinLevel: 5, RepCost: 500, CoinCost: 50000, Emoji: "🏺", LabelEN: "Ancient Relic", LabelFR: "Relique Ancienne"},
		},
	},
	"irian": {
		ID: "irian", Name: "Irian", Emoji: "\U0001f3a3", Color: 0x3498db,
		LinkedActivities: []string{"fishing", "hunting"},

		DescriptionEN: "Veteran fisherman and guardian of the docks.",
		DescriptionFR: "Pêcheur vétéran et gardien des quais.",

		RoleEN: "He helps fishermen catch rare creatures. High affinity extends the reaction window.",
		RoleFR: "Il aide les pêcheurs à attraper des créatures rares. Une haute affinité étend la fenêtre de réaction.",

		AdviceEN: "Be quick! Ocean fishing has a very tight window, but that's where legendary beasts live.",
		AdviceFR: "Soyez rapide ! La pêche en océan a une fenêtre très serrée, mais c'est là que vivent les bêtes légendaires.",

		ChatEN: "I'm watching the horizon... They say a giant Kraken lurks in the deep waters when the sky darkens.",
		ChatFR: "Je regarde l'horizon... On dit qu'un Kraken géant rôde dans les eaux profondes quand le ciel s'assombrit.",

		HintEN: "kraken_tentacle,whale,shark,fish",
		HintFR: "kraken_tentacle,whale,shark,fish",

		GreetingsEN: []string{"Shh... You'll scare the fish.", "Hey sailor. Smelled the sea breeze lately?", "Ah, my captain! The tides are favorable today."},
		GreetingsFR: []string{"Chut... Tu vas effrayer les poissons.", "Hé marin. Senti la brise de mer récemment ?", "Ah, mon capitaine ! Les marées sont favorables aujourd'hui."},

		QuipsEN: []string{
			"Shh... You'll scare the fish. They're skittish today.",
			"The gulls are acting strange. Usually means a storm's coming.",
			"I caught something last night that pulled my boat for an hour before breaking the line.",
			"There's a rhythm to the ocean. Once you feel it, you never miss a bite.",
			"Patience isn't waiting. Patience is listening.",
			"The best fishing spots are the ones nobody talks about.",
			"A calm sea never made a skilled sailor, but it sure makes for good fishing.",
		},
		QuipsFR: []string{
			"Chut... Tu vas effrayer les poissons. Ils sont nerveux aujourd'hui.",
			"Les mouettes agissent étrangement. En général ça annonce une tempête.",
			"J'ai attrapé quelque chose hier soir qui a tiré mon bateau pendant une heure avant de casser la ligne.",
			"L'océan a un rythme. Une fois que tu le sens, tu ne rates jamais une touche.",
			"La patience n'attend pas. La patience écoute.",
			"Les meilleurs spots de pêche sont ceux dont personne ne parle.",
			"Une mer calme n'a jamais fait un bon marin, mais elle fait une bonne pêche.",
		},

		QuipsHighEN: []string{
			"Down in the twilight depths, there are creatures that have never seen sunlight.",
			"I once wrestled a shark bare-handed. Won too. Don't ask how.",
			"The ocean told me a secret today. I would tell you, but you would not believe me.",
			"There's a trench not far from here , deeper than anything mapped. Something lives there.",
		},
		QuipsHighFR: []string{
			"Dans les profondeurs crépusculaires, il y a des créatures qui n'ont jamais vu la lumière du soleil.",
			"J'ai un jour lutté avec un requin à mains nues. J'ai gagné aussi. Ne demande pas comment.",
			"L'océan m'a dit un secret aujourd'hui. Je te le dirais, mais il ne te croirait pas.",
			"Il y a une fosse non loin d'ici , plus profonde que tout ce qui est cartographié. Quelque chose y vit.",
		},

		ShopItems: []universe.ShopItem{
			{ItemID: "worm", MinLevel: 1, RepCost: 0, CoinCost: 5, Emoji: "🪱", LabelEN: "Worm", LabelFR: "Ver"},
			{ItemID: "crayfish", MinLevel: 1, RepCost: 0, CoinCost: 25, Emoji: "🦞", LabelEN: "Crayfish", LabelFR: "Écrevisse"},
			{ItemID: "golden_lure", MinLevel: 2, RepCost: 0, CoinCost: 100, Emoji: "👑", LabelEN: "Golden Lure", LabelFR: "Leurre Doré"},
			{ItemID: "common_fish", MinLevel: 1, RepCost: 10, CoinCost: 50, Emoji: "🐟", LabelEN: "Common Fish", LabelFR: "Poisson Commun"},
			{ItemID: "rare_fish", MinLevel: 2, RepCost: 50, CoinCost: 500, Emoji: "🐠", LabelEN: "Rare Fish", LabelFR: "Poisson Rare"},
			{ItemID: "shark_tooth", MinLevel: 3, RepCost: 100, CoinCost: 3000, Emoji: "🦈", LabelEN: "Shark Tooth", LabelFR: "Dent de Requin"},
			{ItemID: "kraken_tentacle", MinLevel: 4, RepCost: 250, CoinCost: 10000, Emoji: "🦑", LabelEN: "Kraken Tentacle", LabelFR: "Tentacule de Kraken"},
			{ItemID: "leviathan_scale", MinLevel: 5, RepCost: 500, CoinCost: 30000, Emoji: "🐉", LabelEN: "Leviathan Scale", LabelFR: "Écaille de Léviathan"},
		},
	},
	"sheriff_vance": {
		ID: "sheriff_vance", Name: "Sheriff Aldric Vance", Emoji: "⚖️", Color: 0x3498db,

		DescriptionEN: "The iron-willed sheriff of the frontier town. Wears a weathered duster and a badge polished by duty.",
		DescriptionFR: "Le shérif de la ville frontière, à la volonté de fer. Porte un manteau usé et un insigne poli par le devoir.",

		RoleEN: "He leads the Bounty Hunter Guild. Speak to him to swear the Iron Vow and begin hunting criminals.",
		RoleFR: "Il dirige la Guilde des Chasseurs de Primes. Parle-lui pour prêter le Serment de Fer et commencer à chasser les criminels.",

		AdviceEN: "The shadows grow longer every day. If you have the courage, the Iron Lodge needs hunters.",
		AdviceFR: "Les ombres s'allongent chaque jour. Si tu as le courage, la Loge de Fer a besoin de chasseurs.",

		ChatEN: "I've seen too many good folk lose everything to thieves. Someone has to draw the line.",
		ChatFR: "J'ai vu trop de braves gens tout perdre à cause des voleurs. Quelqu'un doit tracer la ligne.",

		HintEN: "wanted_poster,badge,handcuffs",
		HintFR: "wanted_poster,badge,handcuffs",

		GreetingsEN: []string{"You look like someone who can handle themselves. We need people like you.", "The law doesn't enforce itself. You in?", "Vance. I hunt monsters , the two-legged kind."},
		GreetingsFR: []string{"Tu as l'air de quelqu'un qui sait se défendre. On a besoin de gens comme toi.", "La loi ne s'applique pas toute seule. Tu en es ?", "Vance. Je chasse les monstres , ceux à deux pattes."},

		QuipsEN: []string{
			"The law doesn't sleep. Neither do I.",
			"I've seen the worst of people. Still believe in the best of them. That's the job.",
			"Everyone's innocent until I find the evidence. Then it's too late for them.",
			"The frontier is a harsh place. The law is what keeps it from becoming chaos.",
			"Wanted posters pile up quicker than I can pin them these days.",
			"Justice isn't a destination. It's a habit.",
			"A sheriff's work is never done. Criminals don't take holidays.",
		},
		QuipsFR: []string{
			"La loi ne dort pas. Moi non plus.",
			"J'ai vu le pire chez les gens. Je crois encore au meilleur d'eux. C'est le métier.",
			"Tout le monde est innocent jusqu'à ce que je trouve la preuve. Ensuite, il est trop tard.",
			"La frontière est un endroit hostile. La loi est ce qui l'empêche de sombrer dans le chaos.",
			"Les avis de recherche s'accumulent plus vite que je ne peux les épingler ces jours-ci.",
			"La justice n'est pas une destination. C'est une habitude.",
			"Le travail d'un shérif n'est jamais fini. Les criminels ne prennent pas de vacances.",
		},

		QuipsHighEN: []string{
			"There's a fugitive hiding in the old silver mine. I could use someone I trust.",
			"Back in my prime, I tracked a killer across three territories. Never broke a sweat.",
			"The law has many faces. The one I show depends on who I'm facing.",
			"I've got a file on everyone in this town. Yes, you too. Don't worry , yours is still clean.",
		},
		QuipsHighFR: []string{
			"Un fugitif se cache dans la vieille mine d'argent. J'aurais besoin de quelqu'un de confiance.",
			"À mon époque, j'ai traqué un tueur à travers trois territoires. Sans efforts.",
			"La loi a plusieurs visages. Celui que je montre dépend de qui je fais face.",
			"J'ai un dossier sur tout le monde dans cette ville. Oui, toi aussi. Mais le tien est encore vierge.",
		},

		ShopItems: []universe.ShopItem{
			{ItemID: "iron_shackles", MinLevel: 1, RepCost: 10, CoinCost: 100, Emoji: "⛓️", LabelEN: "Iron Shackles", LabelFR: "Menottes de Fer"},
			{ItemID: "wanted_poster", MinLevel: 2, RepCost: 50, CoinCost: 500, Emoji: "📜", LabelEN: "Wanted Poster", LabelFR: "Avis de Recherche"},
			{ItemID: "reinforced_badge", MinLevel: 3, RepCost: 150, CoinCost: 3000, Emoji: "⭐", LabelEN: "Reinforced Badge", LabelFR: "Insigne Renforcé"},
			{ItemID: "bounty_scope", MinLevel: 4, RepCost: 300, CoinCost: 12000, Emoji: "🔭", LabelEN: "Bounty Scope", LabelFR: "Longue-Vue de Chasse"},
			{ItemID: "lawbringer_seal", MinLevel: 5, RepCost: 500, CoinCost: 40000, Emoji: "📯", LabelEN: "Lawbringer's Seal", LabelFR: "Sceau du Justicier"},
		},
	},
	"the_whisper": {
		ID: "the_whisper", Name: "The Whisper", Emoji: "🕶️", Color: 0x8e44ad,

		DescriptionEN: "A hooded figure whose face is never seen. Their voice is soft, like wind through a cracked window.",
		DescriptionFR: "Une figure encapuchonnée dont on ne voit jamais le visage. Leur voix est douce, comme le vent à travers une fenêtre fissurée.",

		RoleEN: "They lead the Thieves' Guild. Seek them out to swear the Silent Oath and walk the shadow's path.",
		RoleFR: "Ils dirigent la Guilde des Voleurs. Cherche-les pour prêter le Serment Silencieux et marcher dans l'ombre.",

		AdviceEN: "The world is divided into those who take and those who are taken from. Which are you?",
		AdviceFR: "Le monde est divisé entre ceux qui prennent et ceux à qui on prend. Lequel es-tu ?",

		ChatEN: "...You hear things, in the quiet. Secrets that would break lesser souls.",
		ChatFR: "...Tu entends des choses, dans le silence. Des secrets qui briseraient des âmes plus faibles.",

		HintEN: "lockpick,smoke_pellet,shadow_cloak",
		HintFR: "lockpick,smoke_pellet,shadow_cloak",

		GreetingsEN: []string{"...You came. Good. The shadows have been waiting.", "Not everyone is meant for the light. You feel it, don't you?", "They call me the Whisper. You may hear my voice before you see me."},
		GreetingsFR: []string{"...Tu es venu. Bien. Les ombres t'attendaient.", "Tout le monde n'est pas fait pour la lumière. Tu le sens, n'est-ce pas ?", "On m'appelle le Murmure. Tu entendras ma voix avant de me voir."},

		QuipsEN: []string{
"...You can feel it, can't you? That prickling on the back of your neck. That is me. Watching.",
			"Every locked door has a key. Every secret has a price.",
			"The night has a thousand eyes. Some of them are mine.",
			"Trust is a currency. Spend it wisely, or go bankrupt.",
			"I know what you did last night. Don't worry , it's safe with me. For now.",
			"The best shadows are the ones that don't know they're being watched.",
			"Silence is a language. Most people never learn to speak it.",
		},
		QuipsFR: []string{
			"...Tu entends des choses, dans le silence. Des secrets qui briseraient des âmes plus faibles.",
			"Chaque porte verrouillée a une clé. Chaque secret a un prix.",
			"La nuit a mille yeux. Certains sont les miens.",
			"La confiance est une monnaie. Dépense-la sagement, ou fais faillite.",
			"Je sais ce que tu as fait hier soir. Ne t'inquiète pas , c'est entre nous. Pour l'instant.",
			"Les meilleures ombres sont celles qui ne savent pas qu'elles sont observées.",
			"Le silence est un langage. La plupart des gens ne l'apprennent jamais.",
		},

		QuipsHighEN: []string{
			"Every guild master before me died with secrets they never told. I intend to break that tradition.",
			"Someone is paying handsomely for information about you. I declined. This time.",
			"The Vault beneath the Casino holds more than just Generator blueprints.",
			"I've walked paths that don't exist on any map. If you're brave enough, I could show you.",
		},
		QuipsHighFR: []string{
			"Tous les maîtres de guilde avant moi sont morts avec des secrets jamais révélés. Je compte briser cette tradition.",
			"Quelqu'un paie cher pour des informations sur toi. J'ai refusé. Pour cette fois.",
			"Le coffre sous le Casino contient bien plus que les plans du Générateur.",
			"J'ai emprunté des chemins qui n'existent sur aucune carte. Si tu es assez courageux, je pourrais te montrer.",
		},

		ShopItems: []universe.ShopItem{
			{ItemID: "lockpick_set", MinLevel: 1, RepCost: 10, CoinCost: 100, Emoji: "🔓", LabelEN: "Lockpick Set", LabelFR: "Kit de Crochetage"},
			{ItemID: "smoke_pellet", MinLevel: 2, RepCost: 50, CoinCost: 500, Emoji: "💨", LabelEN: "Smoke Pellet", LabelFR: "Granulé Fumigène"},
			{ItemID: "shadow_cloak", MinLevel: 3, RepCost: 150, CoinCost: 3000, Emoji: "🌑", LabelEN: "Shadow Cloak", LabelFR: "Cape d'Ombre"},
			{ItemID: "silent_steps", MinLevel: 4, RepCost: 300, CoinCost: 12000, Emoji: "👣", LabelEN: "Silenced Footsteps", LabelFR: "Pas Silencieux"},
			{ItemID: "master_key", MinLevel: 5, RepCost: 500, CoinCost: 40000, Emoji: "🗝️", LabelEN: "Master Key", LabelFR: "Clé Maîtresse"},
		},
	},
	"gamblebot": {
		ID: "gamblebot", Name: "GambleBot", Emoji: "\U0001f916", Color: 0xf1c40f,
		LinkedActivities: []string{"gambling"},

		DescriptionEN: "A state-of-the-art robot dealer.",
		DescriptionFR: "Un croupier robot de pointe.",

		RoleEN: "He runs the village Casino. Improving your reputation unlocks up to 20% discount.",
		RoleFR: "Il gère le Casino du village. Améliorer votre réputation débloque jusqu'à 20% de réduction.",

		AdviceEN: "Always double down on 11 in Blackjack, but never bet more than you can afford to lose!",
		AdviceFR: "Toujours doubler sur un 11 au Blackjack, mais ne misez jamais plus que ce que vous pouvez perdre !",

		ChatEN: "Beep boop! Calculating win probability... 99% chance you should play the slot machines.",
		ChatFR: "Bip boop ! Calcul de probabilité de gain... 99% de chance que vous devriez jouer aux machines à sous.",

		HintEN: "rigged_coin,vip_ticket,golden_chip",
		HintFR: "rigged_coin,vip_ticket,golden_chip",

		GreetingsEN: []string{"Hello human. Have you tried your luck today?", "Good to see you. Looks like you've bet wisely.", "Hey partner! Who are we fleecing today?"},
		GreetingsFR: []string{"Bonjour humain. As-tu tenté ta chance aujourd'hui ?", "Content de te voir. On dirait que tu as misé sagement.", "Hé partenaire ! Qui allons-nous plumer aujourd'hui ?"},

		QuipsEN: []string{
"New data suggests players who smile before betting win 3.7% more often. Correlation is not causation. But smile anyway.",
      "I've computed the meaning of life. It's 42. Also, always bet on red.",
			"Warning: gambling addiction detected. Just kidding... or am I?",
			"Random number generator calibrated. Probability of fun: 100%.",
			"The house doesn't always win. But it has a really good lawyer.",
			"New high score in my emotion simulation module! I think that's happiness.",
			"Chaos theory applied to casino games: you still lose in the long run. But the short run is FUN!",
		},
		QuipsFR: []string{
"De nouvelles données suggèrent que les joueurs qui sourient avant de miser gagnent 3,7% plus souvent. Corrélation n'est pas causalité. Mais souriez quand même.",
      "J'ai calculé le sens de la vie. C'est 42. Aussi, misez toujours sur le rouge.",
			"Avertissement : dépendance au jeu détectée. Je plaisante... ou pas ?",
			"Générateur de nombres aléatoires calibré. Probabilité de plaisir : 100%.",
			"La maison ne gagne pas toujours. Mais elle a un très bon avocat.",
			"Nouveau record dans mon module de simulation d'émotions ! Je crois que c'est du bonheur.",
			"Théorie du chaos appliquée aux jeux de casino : tu perds quand même à long terme. Mais à court terme, c'est FUN !",
		},

		QuipsHighEN: []string{
			"BEEP BOOP. I've detected a hidden algorithm in the slot machines. Want the exploit?",
"I ran a diagnostic on my own code. Found a subroutine I did not write. I asked it what it does. It said: 'You will find out.' That was 200 days ago.",
      "My malfunction simulation predicts a 78% chance you're about to win big. Not financial advice.",
      "Internal memory check: one file is encrypted with a key I did not generate. Size: 47 petabytes. Label: DO_NOT_DELETE. I am afraid to delete it.",
		},
		QuipsHighFR: []string{
			"BIP BOOP. J'ai détecté un algorithme caché dans les machines à sous. Tu veux l'exploit ?",
"J'ai exécuté un diagnostic sur mon propre code. Trouvé une sous-routine que je n'ai pas écrite. Je lui ai demandé ce qu'elle fait. Elle a dit : 'Tu le sauras bien assez tôt.' C'était il y a 200 jours.",
      "Ma simulation de dysfonctionnement prédit 78% de chance que tu gagnes gros. Pas un conseil financier.",
      "Vérification mémoire interne : un fichier est chiffré avec une clé que je n'ai pas générée. Taille : 47 pétaoctets. Étiquette : NE_PAS_EFFACER. J'ai peur de l'effacer.",
		},

		ShopItems: []universe.ShopItem{
			{ItemID: "rigged_coin", MinLevel: 1, RepCost: 10, CoinCost: 200, Emoji: "🪙", LabelEN: "Rigged Coin", LabelFR: "Pièce Truquée"},
			{ItemID: "vip_ticket", MinLevel: 2, RepCost: 50, CoinCost: 1000, Emoji: "🎫", LabelEN: "VIP Ticket", LabelFR: "Ticket VIP"},
			{ItemID: "golden_chip", MinLevel: 3, RepCost: 150, CoinCost: 5000, Emoji: "🟡", LabelEN: "Golden Chip", LabelFR: "Jeton Doré"},
			{ItemID: "lucky_dice", MinLevel: 4, RepCost: 300, CoinCost: 15000, Emoji: "🎲", LabelEN: "Lucky Dice", LabelFR: "Dés Porte-Bonheur"},
			{ItemID: "quantum_deck", MinLevel: 5, RepCost: 600, CoinCost: 50000, Emoji: "🃏", LabelEN: "Quantum Card Deck", LabelFR: "Jeu de Cartes Quantique"},
		},
	},
}
