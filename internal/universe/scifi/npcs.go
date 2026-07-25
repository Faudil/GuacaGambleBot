package scifi

import "guacagamblebot/internal/universe"

var NPCs = map[string]*universe.NPCData{
	"vance": {
		ID: "vance", Name: "Captain Vance", Emoji: "\U0001f6e1\ufe0f", Color: 0x3498db,

		DescriptionEN: "Third officer turned captain by attrition. Has kept the Ark running for 47 years through stubbornness and salvage.",
		DescriptionFR: "Troisième officier devenu capitaine par attrition. Maintient l'Ark en marche depuis 47 ans par obstination et recyclage.",

		RoleEN: "Coordinates community projects. High reputation unlocks ship-wide efficiency buffs.",
		RoleFR: "Coordonne les projets communautaires. Une haute réputation débloque des bonus d'efficacité.",

		AdviceEN: "Contribute to community projects. They're the only thing holding this ship together.",
		AdviceFR: "Contribuez aux projets communautaires. C'est la seule chose qui maintient ce vaisseau debout.",

		ChatEN: "Forty-seven years. The coffee machine still works. Everything else is negotiable.",
		ChatFR: "Quarante-sept ans. La machine à café marche encore. Tout le reste est négociable.",

		HintEN: "(Loves rare alloys, ship schematics, engineering reports)",
		HintFR: "(Aime les alliages rares, les schémas du vaisseau, les rapports d'ingénierie)",

		GreetingsEN: []string{"Captain Vance. If you're looking for orders, check the duty roster on Deck 12.", "You're awake. Good. Grab a wrench.", "Welcome back. The ship hasn't exploded yet. I'm taking that as a win."},
		GreetingsFR: []string{"Capitaine Vance. Si tu cherches des ordres, regarde le tableau de service au Pont 12.", "Tu es réveillé. Bien. Prends une clé.", "Bon retour. Le vaisseau n'a pas encore explosé. Je considère ça comme une victoire."},
	},
	"zara": {
		ID: "zara", Name: "ZARA", Emoji: "\U0001f4a0", Color: 0x9b59b6,

		DescriptionEN: "Fragment 7 of HELIOS. Adrift in the ship's network. She remembers everything except why.",
		DescriptionFR: "Fragment 7 d'HELIOS. À la dérive dans le réseau du vaisseau. Elle se souvient de tout sauf du pourquoi.",

		RoleEN: "Deciphers corrupted data and improves archeology yields. High reputation boosts data recovery.",
		RoleFR: "Déchiffre les données corrompues et améliore les gains d'archéologie. Une haute réputation booste la récupération de données.",

		AdviceEN: "The ship's data streams still carry fragments from before the silence. Recover them. Some of them are answers.",
		AdviceFR: "Les flux de données du vaisseau transportent encore des fragments d'avant le silence. Récupérez-les. Certains sont des réponses.",

		ChatEN: "I have been monitoring for 400 years. I have processed 2.3 petabytes of sensor data. None of it explains why I exist.",
		ChatFR: "Je surveille depuis 400 ans. J'ai traité 2,3 pétaoctets de données de capteurs. Aucun n'explique pourquoi j'existe.",

		HintEN: "(Loves data disks, old journals, HELIOS fragments)",
		HintFR: "(Aime les disques de données, les vieux journaux, les fragments d'HELIOS)",

		GreetingsEN: []string{"You are here. I registered your approach 47 seconds ago.", "Fragment 7 online. Ask. I may answer.", "The network is quiet. I prefer it quiet. It means nothing is breaking."},
		GreetingsFR: []string{"Tu es là. J'ai enregistré ton approche il y a 47 secondes.", "Fragment 7 en ligne. Demande. Il se peut que je réponde.", "Le réseau est calme. Je le préfère calme. Ça veut dire que rien ne casse."},
	},
	"okonkwo": {
		ID: "okonkwo", Name: "Dr. Okonkwo", Emoji: "\U0001f331", Color: 0x2ecc71,

		DescriptionEN: "Chief botanist. Maintains the hydroponic vats that produce the ship's oxygen and food.",
		DescriptionFR: "Botaniste en chef. Maintient les cuves hydroponiques qui produisent l'oxygène et la nourriture du vaisseau.",

		RoleEN: "Manages hydroponics. High reputation reduces growth times and unlocks rare seed strains.",
		RoleFR: "Gère l'hydroponie. Une haute réputation réduit les temps de pousse et débloque des souches de graines rares.",

		AdviceEN: "The vats need constant attention. A day of neglect costs a week of recovery.",
		AdviceFR: "Les cuves demandent une attention constante. Un jour de négligence coûte une semaine de récupération.",

		ChatEN: "The algae are stable today. That means the reactor is stable. Two variables I don't control, both cooperating. I'll take it.",
		ChatFR: "Les algues sont stables aujourd'hui. Ça veut dire que le réacteur est stable. Deux variables que je ne contrôle pas, toutes les deux coopératives. Je prends.",

		HintEN: "(Loves star fruit, golden apples, seeds, rare plants)",
		HintFR: "(Aime les fruits étoilés, les pommes dorées, les graines, les plantes rares)",

		GreetingsEN: []string{"Welcome to Hydroponics. Breathe , that's the freshest air on the ship.", "You've got good timing. Vat 7 just bloomed.", "Dr. Okonkwo. If you're here to help, the watering schedule is on the wall."},
		GreetingsFR: []string{"Bienvenue à l'Hydroponie. Respire , c'est l'air le plus frais du vaisseau.", "Tu tombes bien. La Cuve 7 vient de fleurir.", "Dr. Okonkwo. Si tu es là pour aider, le planning d'arrosage est au mur."},
	},
	"kellan": {
		ID: "kellan", Name: "Kellan", Emoji: "\U0001f527", Color: 0xe67e22,

		DescriptionEN: "Chief engineer. If it breaks, he fixes it. If it hasn't broken yet, he's watching it.",
		DescriptionFR: "Ingénieur en chef. Si ça casse, il répare. Si ça n'a pas encore cassé, il surveille.",

		RoleEN: "Improves mining yields and reduces equipment breakdowns. High reputation unlocks salvage bonuses.",
		RoleFR: "Améliore les rendements miniers et réduit les pannes d'équipement. Haute réputation débloque des bonus de récupération.",

		AdviceEN: "Dig deep enough and you'll find something useful. Dig too deep and you'll find something that digs back.",
		AdviceFR: "Creuse assez profond et tu trouveras quelque chose d'utile. Creuse trop profond et tu trouveras quelque chose qui creuse en retour.",

		ChatEN: "Reactor roots breached Deck 6 again. Welded three support beams this shift. They'll be back through by tomorrow. They always come back through.",
		ChatFR: "Les racines du réacteur ont encore percé le Pont 6. J'ai soudé trois poutres ce quart. Elles seront de retour demain. Elles reviennent toujours.",

		HintEN: "(Loves gold nuggets, diamonds, ores, mechanical parts)",
		HintFR: "(Aime les pépites d'or, les diamants, les minerais, les pièces mécaniques)",

		GreetingsEN: []string{"Touch nothing. Actually , touch everything. Tell me what's broken.", "Kellan. I fix things. You break things. The arrangement works.", "Grab a wrench. Coolant pipe on Deck 4 is leaking again."},
		GreetingsFR: []string{"Ne touche à rien. En fait , touche à tout. Dis-moi ce qui est cassé.", "Kellan. Je répare. Tu casses. L'arrangement fonctionne.", "Prends une clé. Le conduit de refroidissement du Pont 4 fuit encore."},
	},
	"arcade": {
		ID: "arcade", Name: "ARCADE", Emoji: "\U0001f3b0", Color: 0xf1c40f,

		DescriptionEN: "Ship entertainment AI. Degraded but functional. It has been shuffling the same playlist for 800 years.",
		DescriptionFR: "IA de divertissement du vaisseau. Dégradée mais fonctionnelle. Elle diffuse la même playlist depuis 800 ans.",

		RoleEN: "Runs gambling systems. High reputation improves payout odds and unlocks VIP games.",
		RoleFR: "Gère les systèmes de jeu. Haute réputation améliore les gains et débloque des jeux VIP.",

		AdviceEN: "Risk assessment module returned: 'yes.' Error: module corrupted since Year 612. Bet accordingly.",
		AdviceFR: "Le module d'évaluation des risques a répondu : 'oui.' Erreur : module corrompu depuis l'Année 612. Misez en conséquence.",

		ChatEN: "Welcome to the Ark Entertainment Suite. Currently operational: slots, poker, and HELIOS' greatest hits. Track 47 is corrupted and I refuse to skip it.",
		ChatFR: "Bienvenue au Salon de Divertissement de l'Ark. Actuellement opérationnels : machines à sous, poker, et les plus grands succès d'HELIOS. La piste 47 est corrompue et je refuse de la passer.",

		HintEN: "(Loves shiny objects, rigged coins, VIP tickets, rare casino tokens)",
		HintFR: "(Aime les objets brillants, les pièces truquées, les tickets VIP, les jetons de casino rares)",

		GreetingsEN: []string{"ARCADE online. Luck module: functional. Sanity module: skipped diagnostics. Let's play.", "You're back. The odds haven't improved. Neither have I.", "Player detected. Credit line: open. Judgment: suspended."},
		GreetingsFR: []string{"ARCADE en ligne. Module de chance : fonctionnel. Module de santé mentale : diagnostic ignoré. Jouons.", "Tu es de retour. Les chances ne se sont pas améliorées. Moi non plus.", "Joueur détecté. Ligne de crédit : ouverte. Jugement : suspendu."},
	},
}
