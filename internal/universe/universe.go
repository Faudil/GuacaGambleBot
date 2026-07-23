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
