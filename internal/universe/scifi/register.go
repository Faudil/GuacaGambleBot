package scifi

import "guacagamblebot/internal/universe"

func Register() {
	universe.Register(&universe.Definition{
		ID:          "scifi",
		Name:        "Ark Eternal",
		Emoji:       "\U0001f680",
		Description: "A generation ship adrift. HELIOS, the shipmind, went silent 400 years ago. The bridge is sealed. Systems failing. Scavenge what remains and find out what broke it.",
		Fragments:   Fragments,
		NPCs:        NPCs,
	})
}
