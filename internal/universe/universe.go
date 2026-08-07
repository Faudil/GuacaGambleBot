package universe

type Category string

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
	if lang == "en" {
		return f.TitleEN
	}
	return f.TitleFR
}

func (f *Fragment) Text(lang string) string {
	if lang == "en" {
		return f.TextEN
	}
	return f.TextFR
}

type ShopItem struct {
	ItemID   string
	MinLevel int
	RepCost  int
	CoinCost int
	Emoji    string
	LabelEN  string
	LabelFR  string
}

type NPCData struct {
	ID          string
	Name        string
	Emoji       string
	Color       int

	DescriptionEN string
	DescriptionFR string
	RoleEN        string
	RoleFR        string
	AdviceEN      string
	AdviceFR      string
	ChatEN        string
	ChatFR        string
	HintEN        string
	HintFR        string
	GreetingsEN   []string
	GreetingsFR   []string

	QuipsEN      []string
	QuipsFR      []string
	QuipsHighEN  []string
	QuipsHighFR  []string
	ShopItems    []ShopItem

	// LinkedActivities are the player activities (e.g. "mining", "fishing",
	// "gambling") that award small reputation points with this NPC.
	LinkedActivities []string
}

func (n *NPCData) Description(lang string) string {
	if lang == "en" {
		return n.DescriptionEN
	}
	return n.DescriptionFR
}

func (n *NPCData) Role(lang string) string {
	if lang == "en" {
		return n.RoleEN
	}
	return n.RoleFR
}

func (n *NPCData) Advice(lang string) string {
	if lang == "en" {
		return n.AdviceEN
	}
	return n.AdviceFR
}

func (n *NPCData) Chat(lang string) string {
	if lang == "en" {
		return n.ChatEN
	}
	return n.ChatFR
}

func (n *NPCData) Hint(lang string) string {
	if lang == "en" {
		return n.HintEN
	}
	return n.HintFR
}

func (n *NPCData) Greetings(lang string) []string {
	if lang == "en" {
		return n.GreetingsEN
	}
	return n.GreetingsFR
}

func (n *NPCData) Quips(lang string) []string {
	if lang == "en" {
		return n.QuipsEN
	}
	return n.QuipsFR
}

func (n *NPCData) QuipsHigh(lang string) []string {
	if lang == "en" {
		return n.QuipsHighEN
	}
	return n.QuipsHighFR
}

type Definition struct {
	ID   string
	Name string
	Emoji string
	Description string

	Fragments []Fragment
	NPCs      map[string]*NPCData

	LocaleOverrideEN map[string]string
	LocaleOverrideFR map[string]string
}
