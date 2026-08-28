package interaction

import (
	"github.com/bwmarrin/discordgo"

	"guacagamblebot/internal/i18n"
)

// Command is one registered slash command as the help screen sees it. The
// router builds this list as the cogs register, so /help is generated from the
// commands that actually exist rather than from a list maintained by hand —
// which is how the old help screen came to document 6 of 43 cogs.
//
// DescKey is an i18n key, not a sentence. Several names may share one key: that
// is how aliases are declared (both "economy" and "eco" carry
// "cmd.economy.desc"), and the name matching the key is the canonical one.
type Command struct {
	Name     string
	DescKey  string
	Category string
}

// CommandGroup is one logical command with the alternate names it answers to,
// as rendered on a help screen.
type CommandGroup struct {
	Name      string
	Aliases   []string
	DescKey   string
	HasPrefix bool
}

// Categorize runs register with every command it registers filed under
// category. Categories are assigned at the wiring site rather than inside each
// cog so the whole taxonomy stays reviewable in one table.
func (r *Router) Categorize(category string, register func()) {
	prev := r.category
	r.category = category
	register()
	r.category = prev
}

// Catalog returns the commands registered under category, in registration
// order, with aliases folded into their canonical command.
func (r *Router) Catalog(category string) []CommandGroup {
	var groups []CommandGroup
	index := map[string]int{}
	for _, c := range r.commands {
		if c.Category != category {
			continue
		}
		if i, ok := index[c.DescKey]; ok {
			g := &groups[i]
			// The name that matches the description key is the canonical one;
			// anything else is an alias for it.
			if canonicalName(c.DescKey) == c.Name {
				g.Aliases = append(g.Aliases, g.Name)
				g.Name = c.Name
			} else {
				g.Aliases = append(g.Aliases, c.Name)
			}
			g.HasPrefix = g.HasPrefix || r.HasPrefix(c.Name)
			continue
		}
		index[c.DescKey] = len(groups)
		groups = append(groups, CommandGroup{
			Name:      c.Name,
			DescKey:   c.DescKey,
			HasPrefix: r.HasPrefix(c.Name),
		})
	}
	return groups
}

// Categories returns every category that has at least one command, in the order
// the cogs were wired.
func (r *Router) Categories() []string {
	seen := map[string]bool{}
	var out []string
	for _, c := range r.commands {
		if c.Category == "" || seen[c.Category] {
			continue
		}
		seen[c.Category] = true
		out = append(out, c.Category)
	}
	return out
}

// Commands returns every registered command, uncategorised ones included. It
// backs the startup check that no command is missing its help metadata.
func (r *Router) CommandList() []Command { return r.commands }

// HasPrefix reports whether name also works as a `!name` message command.
func (r *Router) HasPrefix(name string) bool {
	_, ok := r.prefix[name]
	return ok
}

// canonicalName extracts the command name from a "cmd.<name>.desc" key.
func canonicalName(descKey string) string {
	if len(descKey) > len("cmd.")+len(".desc") &&
		descKey[:4] == "cmd." && descKey[len(descKey)-5:] == ".desc" {
		return descKey[4 : len(descKey)-5]
	}
	return ""
}

// slashDef builds the Discord command definition for a registration. The
// description is resolved from the locale packs so Discord's own command picker
// shows each user the description in their client language, instead of the
// mix of French and English literals the registrations used to carry.
func slashDef(name, descKey string, options []*discordgo.ApplicationCommandOption) *discordgo.ApplicationCommand {
	localizations := map[discordgo.Locale]string{discordgo.French: i18n.T(descKey, "fr")}
	return &discordgo.ApplicationCommand{
		Name:                     name,
		Description:              i18n.T(descKey, "en"),
		DescriptionLocalizations: &localizations,
		Options:                  options,
	}
}
