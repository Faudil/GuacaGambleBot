package scorch

import "guacagamblebot/internal/universe"

func Register() {
	universe.Register(&universe.Definition{
		ID:          "scorch",
		Name:        "Scorch",
		Emoji:       "\u2622\ufe0f",
		Description: "Decades after the collapse. Survivors pick through ruins, trade salvage for water, and keep quiet so the things in the dead zones don't notice them.",
		Fragments:   Fragments,
		NPCs:        NPCs,
	})
}
