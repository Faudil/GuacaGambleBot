package items

type Category string

const (
	Mining      Category = "mining"
	Fishing     Category = "fishing"
	Farming     Category = "farming"
	Archeology  Category = "archeology"
	Food        Category = "food"
	Tools       Category = "tools"
	Materials   Category = "materials"
	Special     Category = "special"
)

type Item struct {
	Name        string
	Emoji       string
	Price       int
	Description string
	EffectType  string
	Droppable   bool
	Category    Category
}

var all = []Item{
	// --- Mining ---
	{Name: "caillou", Emoji: "🪨", Price: 1, Description: "Un caillou tout nul. Sert à rien.", EffectType: "resource", Droppable: true, Category: Mining},
	{Name: "charbon", Emoji: "🪨", Price: 5, Description: "Pas mal pour se réchauffer.", EffectType: "resource", Droppable: true, Category: Mining},
	{Name: "minerai de fer", Emoji: "⛏️", Price: 10, Description: "Utile pour forger des trucs solides.", EffectType: "resource", Droppable: true, Category: Mining},
	{Name: "minerai de cuivre", Emoji: "⛏️", Price: 15, Description: "Utile pour forger des trucs solides.", EffectType: "resource", Droppable: true, Category: Mining},
	{Name: "minerai d'argent", Emoji: "⛏️", Price: 25, Description: "Utile pour forger des trucs solides.", EffectType: "resource", Droppable: true, Category: Mining},
	{Name: "pépite d'or", Emoji: "✨", Price: 50, Description: "Brillant ! Les marchands adorent ça.", EffectType: "resource", Droppable: true, Category: Mining},
	{Name: "platine", Emoji: "✨", Price: 75, Description: "Brillant ! Les marchands adorent ça.", EffectType: "resource", Droppable: true, Category: Mining},
	{Name: "emeraude", Emoji: "💚", Price: 100, Description: "INCROYABLE ! Ça vaut une fortune !", EffectType: "resource", Droppable: true, Category: Mining},
	{Name: "diamant brut", Emoji: "💎", Price: 300, Description: "INCROYABLE ! Ça vaut une fortune !", EffectType: "resource", Droppable: true, Category: Mining},

	// --- Fishing ---
	{Name: "vieille botte", Emoji: "🥾", Price: 1, Description: "Un caillou tout nul. Sert à rien.", EffectType: "resource", Droppable: true, Category: Fishing},
	{Name: "truite", Emoji: "🐟", Price: 10, Description: "Un poisson d'eau douce.", EffectType: "resource", Droppable: true, Category: Fishing},
	{Name: "saumon", Emoji: "🐟", Price: 10, Description: "Parfait pour faire des sushis.", EffectType: "resource", Droppable: true, Category: Fishing},
	{Name: "sardine", Emoji: "🐟", Price: 15, Description: "Un petit poisson d'eau de mer", EffectType: "resource", Droppable: true, Category: Fishing},
	{Name: "carpe", Emoji: "🐟", Price: 25, Description: "Le meilleur poisson d'eau douce.", EffectType: "resource", Droppable: true, Category: Fishing},
	{Name: "poisson-globe", Emoji: "🐡", Price: 50, Description: "Attention, ça pique !", EffectType: "resource", Droppable: true, Category: Fishing},
	{Name: "espadon", Emoji: "🐟", Price: 150, Description: "Un poisson combattant majestueux.", EffectType: "resource", Droppable: true, Category: Fishing},
	{Name: "requin", Emoji: "🦈", Price: 100, Description: "INCROYABLE ! Ça vaut une fortune !", EffectType: "resource", Droppable: true, Category: Fishing},
	{Name: "baleine", Emoji: "🐋", Price: 300, Description: "INCROYABLE ! Ça vaut une fortune !", EffectType: "resource", Droppable: true, Category: Fishing},
	{Name: "tentacule de kraken", Emoji: "🦑", Price: 500, Description: "TU AS PÊCHÉ UN MONSTRE ?!", EffectType: "resource", Droppable: true, Category: Fishing},

	// --- Farming ---
	{Name: "plante pourrie", Emoji: "🌿", Price: 0, Description: "Tu as mal géré ta ferme...", EffectType: "resource", Droppable: true, Category: Farming},
	{Name: "blé", Emoji: "🌾", Price: 5, Description: "Indispensable pour faire du pain.", EffectType: "resource", Droppable: true, Category: Farming},
	{Name: "avoine", Emoji: "🌾", Price: 8, Description: "Parfait pour le petit déjeuner.", EffectType: "resource", Droppable: true, Category: Farming},
	{Name: "maïs", Emoji: "🌽", Price: 12, Description: "Fait aussi du pop-corn !", EffectType: "resource", Droppable: true, Category: Farming},
	{Name: "patate", Emoji: "🥔", Price: 20, Description: "On peut en faire de la vodka...", EffectType: "resource", Droppable: true, Category: Farming},
	{Name: "tomate", Emoji: "🍅", Price: 25, Description: "Un fruit ou un légume ? Le débat continue.", EffectType: "resource", Droppable: true, Category: Farming},
	{Name: "citrouille", Emoji: "🎃", Price: 40, Description: "Parfait pour Halloween.", EffectType: "resource", Droppable: true, Category: Farming},
	{Name: "grain de café", Emoji: "🫘", Price: 60, Description: "L'or noir du matin.", EffectType: "resource", Droppable: true, Category: Farming},
	{Name: "fève de cacao", Emoji: "🫘", Price: 75, Description: "L'ingrédient principal du bonheur (chocolat).", EffectType: "resource", Droppable: true, Category: Farming},
	{Name: "fraise", Emoji: "🍓", Price: 90, Description: "Rouge, sucrée et juteuse.", EffectType: "resource", Droppable: true, Category: Farming},
	{Name: "pomme dorée", Emoji: "🍎", Price: 150, Description: "Elle brille d'une lueur magique.", EffectType: "resource", Droppable: true, Category: Farming},
	{Name: "fruit étoile", Emoji: "⭐", Price: 250, Description: "Un fruit cosmique d'une autre dimension.", EffectType: "resource", Droppable: true, Category: Farming},

	// --- Archeology ---
	{Name: "poussière d'os", Emoji: "🦴", Price: 1, Description: "De la poussière d'os de fossile complètement détruit.", EffectType: "resource", Droppable: true, Category: Archeology},
	{Name: "fossile abîmé", Emoji: "🦴", Price: 50, Description: "Un fossile mal extrait, il a perdu de sa valeur.", EffectType: "resource", Droppable: true, Category: Archeology},
	{Name: "fossile commun", Emoji: "🦴", Price: 150, Description: "Un fossile intact d'animal commun.", EffectType: "resource", Droppable: true, Category: Archeology},
	{Name: "fossile rare", Emoji: "🦴", Price: 300, Description: "Un fossile intact d'animal rare.", EffectType: "resource", Droppable: true, Category: Archeology},
	{Name: "fossile épique", Emoji: "🦴", Price: 500, Description: "Un fossile intact d'animal épique.", EffectType: "resource", Droppable: true, Category: Archeology},
	{Name: "fragment légendaire", Emoji: "🦖", Price: 1000, Description: "Un fragment légendaire d'un T-Rex !", EffectType: "resource", Droppable: true, Category: Archeology},
	{Name: "adn pur", Emoji: "🧬", Price: 3000, Description: "De l'ADN de dinosaure parfaitement conservé. Incroyable !", EffectType: "resource", Droppable: true, Category: Archeology},

	// --- Seeds (Materials) ---
	{Name: "graine de blé", Emoji: "🌱", Price: 2, Description: "À planter pour obtenir du blé (5 min).", EffectType: "resource", Droppable: true, Category: Materials},
	{Name: "graine d'avoine", Emoji: "🌱", Price: 3, Description: "À planter pour obtenir de l'avoine (10 min).", EffectType: "resource", Droppable: true, Category: Materials},
	{Name: "graine de maïs", Emoji: "🌱", Price: 5, Description: "À planter pour obtenir du maïs (30 min).", EffectType: "resource", Droppable: true, Category: Materials},
	{Name: "graine de patate", Emoji: "🌱", Price: 8, Description: "À planter pour obtenir des patates (1h).", EffectType: "resource", Droppable: true, Category: Materials},
	{Name: "graine de tomate", Emoji: "🌱", Price: 10, Description: "À planter pour obtenir des tomates (2h).", EffectType: "resource", Droppable: true, Category: Materials},
	{Name: "graine de citrouille", Emoji: "🌱", Price: 15, Description: "À planter pour obtenir des citrouilles (4h).", EffectType: "resource", Droppable: true, Category: Materials},
	{Name: "graine de café", Emoji: "🌱", Price: 25, Description: "À planter pour obtenir du café (8h).", EffectType: "resource", Droppable: true, Category: Materials},
	{Name: "graine de cacao", Emoji: "🌱", Price: 30, Description: "À planter pour obtenir du cacao (12h).", EffectType: "resource", Droppable: true, Category: Materials},
	{Name: "graine de fraise", Emoji: "🌱", Price: 40, Description: "À planter pour obtenir des fraises (18h).", EffectType: "resource", Droppable: true, Category: Materials},
	{Name: "pépin de pomme dorée", Emoji: "🌱", Price: 75, Description: "À planter pour obtenir des pommes dorées (24h).", EffectType: "resource", Droppable: true, Category: Materials},
	{Name: "pépin de fruit étoile", Emoji: "🌱", Price: 125, Description: "À planter pour obtenir des fruits étoiles (48h).", EffectType: "resource", Droppable: true, Category: Materials},

	// --- Tools ---
	{Name: "bière", Emoji: "🍺", Price: 50, Description: "La boisson du mineur ! Réinitialise le cooldown de !mine.", EffectType: "consumable", Droppable: false, Category: Tools},
	{Name: "café", Emoji: "☕", Price: 50, Description: "Réveille tes sens. Réinitialise le cooldown du !daily.", EffectType: "consumable", Droppable: false, Category: Tools},
	{Name: "arc", Emoji: "🏹", Price: 300, Description: "Aide à la chasse ! Réinitialise le cooldown de !hunt.", EffectType: "consumable", Droppable: false, Category: Tools},
	{Name: "engrais", Emoji: "🧪", Price: 200, Description: "Accélère la pousés des récoltes ! Réinitialise le cooldown de !farm.", EffectType: "consumable", Droppable: false, Category: Tools},
	{Name: "hameçon", Emoji: "🪝", Price: 200, Description: "Attire les poissons ! Réinitialise le cooldown de !fish.", EffectType: "consumable", Droppable: false, Category: Tools},
	{Name: "potion d'oubli", Emoji: "🧪", Price: 2500, Description: "Réinitialise ton familier. Reset son estomac et le remet niveau 10.", EffectType: "consumable", Droppable: false, Category: Tools},
	{Name: "fortune cookie", Emoji: "🥠", Price: 20, Description: "Un biscuit délicieux avec un message prémonitoire.", EffectType: "consumable", Droppable: false, Category: Tools},
	{Name: "aimant rouillé", Emoji: "🧲", Price: 30, Description: "🧲 Utilise-le pour trouver de la petite monnaie par terre.", EffectType: "consumable", Droppable: false, Category: Tools},
	{Name: "aimant", Emoji: "🧲", Price: 50, Description: "🧲 Utilise-le pour trouver de la monnaie par terre.", EffectType: "consumable", Droppable: false, Category: Tools},
	{Name: "aimant électrique", Emoji: "🧲", Price: 500, Description: "🧲 Utilise-le pour trouver un max de monnaie par terre.", EffectType: "consumable", Droppable: false, Category: Tools},
	{Name: "pièce truquée", Emoji: "🪙", Price: 200, Description: "Augmente ta chance. Passe ta probabilité de réussir ton pile ou face à 75%.", EffectType: "consumable", Droppable: false, Category: Tools},
	{Name: "jeton de casino", Emoji: "🎰", Price: 50, Description: "Te donne une nouvelle chance, reset ta limite de !casino", EffectType: "consumable", Droppable: false, Category: Tools},
	{Name: "ticket vip", Emoji: "🎟️", Price: 100, Description: "Te donne une nouvelle chance, reset tes limites de casino et coinflip", EffectType: "consumable", Droppable: false, Category: Tools},
	{Name: "parchemin d'identité", Emoji: "📜", Price: 500, Description: "Change ton surnom sur le serveur aléatoirement.", EffectType: "consumable", Droppable: false, Category: Tools},
	{Name: "thieve's glove", Emoji: "🧤", Price: 20, Description: "Utilise !rob @target avec ces gants.", EffectType: "consumable", Droppable: false, Category: Tools},
	{Name: "ticket à gratter", Emoji: "🎰", Price: 100, Description: "Grattez pour gagner jusqu'à 1000$ instantanément !", EffectType: "consumable", Droppable: false, Category: Tools},
	{Name: "data disk", Emoji: "💾", Price: 50, Description: "Un disque mémoire Zénith corrompu.", EffectType: "consumable", Droppable: false, Category: Tools},
	{Name: "old journal", Emoji: "📖", Price: 30, Description: "Un carnet poussiéreux écrit par un survivant.", EffectType: "consumable", Droppable: false, Category: Tools},

	// --- Special ---
	{Name: "œuf mystère", Emoji: "🥚", Price: 6000, Description: "Un œuf frémissant... Tape !hatch pour l'ouvrir !", EffectType: "consumable", Droppable: false, Category: Special},
	{Name: "œuf saison", Emoji: "🥚", Price: 12000, Description: "Un œuf frémissant... Tape !hatch pour l'ouvrir !", EffectType: "consumable", Droppable: false, Category: Special},
	{Name: "trophée de boss", Emoji: "🏆", Price: 10000, Description: "Un trophée légendaire récompensant le premier joueur à vaincre le boss.", EffectType: "collectible", Droppable: false, Category: Special},
	{Name: "terrain : potager", Emoji: "🌿", Price: 500, Description: "Un lopin de terre fertile pour faire pousser des légumes.", EffectType: "permanent", Droppable: false, Category: Special},
	{Name: "terrain : serre tropicale", Emoji: "🌿", Price: 1000, Description: "Une structure en verre chauffée pour le café et le cacao.", EffectType: "permanent", Droppable: false, Category: Special},
	{Name: "terrain : verger enchanté", Emoji: "🌿", Price: 10000, Description: "Une île flottante magique. Seuls les fruits légendaires y poussent.", EffectType: "permanent", Droppable: false, Category: Special},
}

var byName = func() map[string]*Item {
	m := make(map[string]*Item, len(all))
	for i := range all {
		m[all[i].Name] = &all[i]
	}
	return m
}()

var byCategory = func() map[Category][]Item {
	m := make(map[Category][]Item)
	for _, it := range all {
		m[it.Category] = append(m[it.Category], it)
	}
	return m
}()

func Get(name string) *Item {
	return byName[name]
}

func AllItems() []Item {
	return all
}

func ItemsByCategory(cat Category) []Item {
	return byCategory[cat]
}
