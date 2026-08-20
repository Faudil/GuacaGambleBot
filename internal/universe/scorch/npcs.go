package scorch

import "guacagamblebot/internal/universe"

var NPCs = map[string]*universe.NPCData{
	"wexler": {
		ID: "wexler", Name: "Wexler", Emoji: "\U0001f3aa", Color: 0x8B7355,
		Image: "npcs/wexler.png",

		DescriptionEN: "Runs the trading post. Knows everyone by what they carry.",
		DescriptionFR: "Tient le comptoir d'échange. Connaît tout le monde à ce qu'ils transportent.",

		RoleEN: "Buys salvage and sells gear. High reputation improves trade rates and unlocks rarer stock.",
		RoleFR: "Achète la récupération et vend de l'équipement. Haute réputation améliore les taux d'échange et débloque du stock plus rare.",

		AdviceEN: "Check buildings thoroughly. Most people miss the good stuff because they don't look up.",
		AdviceFR: "Fouille les bâtiments à fond. La plupart des gens ratent le bon matos parce qu'ils ne regardent pas en haut.",

		ChatEN: "Sold a pre-collapse coffee maker last week. Still works. Lady paid three days of water for it. I'd have done the same.",
		ChatFR: "J'ai vendu une machine à café d'avant la semaine dernière. Elle marche encore. Une dame a payé trois jours d'eau pour ça. J'aurais fait pareil.",

		HintEN: "(Loves salvage, rare alloys, pre-collapse electronics, anything that still works)",
		HintFR: "(Aime la récupération, les alliages rares, l'électronique d'avant, tout ce qui marche encore)",

		GreetingsEN: []string{"Wexler. You buying or selling? Either way, let's talk.", "Got new stock. Nothing special. But it's clean and it works. Mostly.", "You look like someone who's been out past the gates. Good. Tell me what you saw."},
		GreetingsFR: []string{"Wexler. T'achètes ou tu vends ? Dans les deux cas, on parle.", "Nouveau stock. Rien de spécial. Mais c'est propre et ça marche. En général.", "T'as l'air de quelqu'un qui est allé au-delà des portes. Bien. Dis-moi ce que t'as vu."},
	},
	"riggs": {
		ID: "riggs", Name: "Riggs", Emoji: "\U0001f441\ufe0f\u200d\U0001f5e8\ufe0f", Color: 0x607B8B,
		Image:            "npcs/riggs.png",
		LinkedActivities: []string{"hunting"},

		DescriptionEN: "Settlement security. Patrols the perimeter. He's been out past the treeline more times than anyone.",
		DescriptionFR: "Sécurité de la colonie. Patrouille le périmètre. Il est allé au-delà de la ligne d'arbres plus de fois que quiconque.",

		RoleEN: "Leads patrols and hunts. High reputation improves hunting yields and reveals safe routes.",
		RoleFR: "Dirige les patrouilles et la chasse. Haute réputation améliore les rendements de chasse et révèle des itinéraires sûrs.",

		AdviceEN: "If you hear breathing and you can't see the source, walk. Don't run. Running is a kill instinct trigger for half the things out there.",
		AdviceFR: "Si tu entends respirer sans voir la source, marche. Ne cours pas. Courir déclenche l'instinct de prédation chez la moitié des trucs dehors.",

		ChatEN: "Saw something new today. North perimeter. It stood outside the fence for an hour. Just watching. Then it walked east. Like it had an appointment.",
		ChatFR: "J'ai vu un truc nouveau aujourd'hui. Périmètre nord. Il est resté devant la clôture une heure. Juste à regarder. Puis il est parti vers l'est. Comme s'il avait rendez-vous.",

		HintEN: "(Loves weapons, ammunition, armor parts, patrol reports)",
		HintFR: "(Aime les armes, les munitions, les pièces d'armure, les rapports de patrouille)",

		GreetingsEN: []string{"Riggs. Gate's secure. Nothing on the perimeter. You're good.", "You planning to go out? Take water. Tell someone where. Don't be stupid.", "Back again. Good. I was starting to count you as a loss."},
		GreetingsFR: []string{"Riggs. La porte est sécurisée. Rien sur le périmètre. Tout va bien.", "Tu prévois de sortir ? Prends de l'eau. Dis à quelqu'un où. Fais pas l'idiot.", "De retour. Bien. Je commençais à te compter comme perdu."},
	},
	"mother": {
		ID: "mother", Name: "Mother Glitch", Emoji: "\u2699\ufe0f", Color: 0x8B4789,
		Image:            "npcs/mother.png",
		LinkedActivities: []string{"fishing"},

		DescriptionEN: "Maintains the water filters. Talks to the machines. Water is always clean.",
		DescriptionFR: "Entretient les filtres à eau. Parle aux machines. L'eau est toujours propre.",

		RoleEN: "Filtration and purification. High reputation improves fishing yields and reduces contamination risk.",
		RoleFR: "Filtration et purification. Haute réputation améliore les prises et réduit les risques de contamination.",

		AdviceEN: "If the water tastes sweet, don't drink it. Sweet means something's growing in it.",
		AdviceFR: "Si l'eau a un goût sucré, ne la bois pas. Sucré veut dire que quelque chose pousse dedans.",

		ChatEN: "Filter three is humming again. It does that when it's about to clog. I told it to hold on. It usually listens.",
		ChatFR: "Le filtre trois bourdonne encore. Il fait ça quand il va se boucher. Je lui ai dit de tenir. En général il écoute.",

		HintEN: "(Loves mechanical parts, chemical reagents, old schematics, anything with a working circuit)",
		HintFR: "(Aime les pièces mécaniques, les réactifs chimiques, les vieux schémas, tout ce qui a un circuit qui marche)",

		GreetingsEN: []string{"Mother Glitch. Don't touch the pipes. They're doing their best.", "Water's clean. Tested it myself. Well. Myself and the strips.", "Ah. You. The machines remember you. That's neutral."},
		GreetingsFR: []string{"Mother Glitch. Touche pas aux tuyaux. Ils font de leur mieux.", "L'eau est propre. Je l'ai testée moi-même. Enfin. Moi-même et les bandelettes.", "Ah. Toi. Les machines se souviennent de toi. C'est neutre."},
	},
	"pyke": {
		ID: "pyke", Name: "Pyke", Emoji: "\U0001f52a", Color: 0xCC5500,
		Image: "npcs/pyke.png",

		DescriptionEN: "Former raider. Knows the dead zones. Doesn't sleep without a knife.",
		DescriptionFR: "Ancien pillard. Connaît les zones mortes. Ne dort pas sans un couteau.",

		RoleEN: "Dead zone guide. High reputation unlocks safer expedition routes and better loot from dangerous areas.",
		RoleFR: "Guide des zones mortes. Haute réputation débloque des itinéraires d'expédition plus sûrs et un meilleur butin.",

		AdviceEN: "The dead zones change every season. What was safe last month isn't safe now. Trust nothing. Especially me.",
		AdviceFR: "Les zones mortes changent chaque saison. Ce qui était sûr le mois dernier ne l'est plus. Fais confiance à rien. Surtout pas à moi.",

		ChatEN: "I was with the Wreckers for three years. We hit twelve settlements. Lost count of how many people. The Wreckers are gone now. I'm still here. You figure out why.",
		ChatFR: "J'étais avec les Démolisseurs pendant trois ans. On a frappé douze colonies. Perdu le compte du nombre de gens. Les Démolisseurs ne sont plus là. Moi je suis encore là. Devine pourquoi.",

		HintEN: "(Loves knives, survival gear, maps of dead zones, med supplies)",
		HintFR: "(Aime les couteaux, l'équipement de survie, les cartes des zones mortes, les fournitures médicales)",

		GreetingsEN: []string{"Pyke. You want to go somewhere dangerous? Good. Let's talk price.", "Keep your voice down. Some things hunt by sound.", "Back alive. That's more than most people manage."},
		GreetingsFR: []string{"Pyke. Tu veux aller quelque part de dangereux ? Bien. Parlons prix.", "Baisse la voix. Certaines choses chassent au son.", "De retour vivant. C'est plus que ce que la plupart des gens arrivent à faire."},
	},

	"the_chronicler": {
		ID: "the_chronicler", Name: "The Chronicler", Emoji: "\U0001f56f\ufe0f", Color: 0x2c3e50,
		Image: "npcs/the_chronicler.png",

		DescriptionEN: "A hooded figure who writes in a book that never runs out of pages.",
		DescriptionFR: "Une silhouette encapuchonnée qui écrit dans un livre aux pages inépuisables.",

		RoleEN: "He watches those who walk the paths of the Journal. Speak to him once you have reached rank 2 on a path and defeated the Vault Guardian.",
		RoleFR: "Il observe ceux qui parcourent les voies du Journal. Parle-lui une fois que tu as atteint le rang 2 dans une voie et vaincu le Gardien du Coffre.",

		AdviceEN: "Every path you walk writes a line in my book. Walk them all, and the legend becomes yours.",
		AdviceFR: "Chaque voie que tu parcours écrit une ligne dans mon livre. Parcours-les toutes, et la légende devient tienne.",

		ChatEN: "The ink remembers. So do I.",
		ChatFR: "L'encre se souvient. Moi aussi.",

		GreetingsEN: []string{"I have seen you in the pages.", "The book never forgets a name.", "You walk a path worth writing down."},
		GreetingsFR: []string{"Je t'ai vu dans les pages.", "Le livre n'oublie jamais un nom.", "Tu parcours un chemin qui mérite d'être écrit."},

		QuipsEN: []string{
			"Somewhere, someone is finishing a path you have not begun.",
			"The miners speak of you in the low taverns.",
			"Every rumor you hear is a page someone else has turned.",
			"The Hall of Whispers holds what you seek — if you seek everything.",
			"Legends are not born. They are written, one small step at a time.",
		},
		QuipsFR: []string{
			"Quelque part, quelqu'un achève un chemin que tu n'as pas commencé.",
			"Les mineurs parlent de toi dans les tavernes basses.",
			"Chaque rumeur que tu entends est une page que quelqu'un d'autre a tournée.",
			"La Salle des Murmures garde ce que tu cherches — si tu cherches tout.",
			"Les légendes ne naissent pas. Elles s'écrivent, un petit pas à la fois.",
		},
	},
}
