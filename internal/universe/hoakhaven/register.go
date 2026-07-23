package hoakhaven

import "guacagamblebot/internal/universe"

func Register() {
	universe.Register(&universe.Definition{
		ID:          "hoakhaven",
		Name:        "HoakHaven",
		Emoji:       "\U0001f3e0",
		Description: "A post-apocalyptic world where survivors rebuild civilization among the ruins of the Zenith. Explore the Strata, uncover Aether-forged mysteries, and restore the Nexus.",
		Fragments:   Fragments,
		NPCs:        NPCs,
	})
}
