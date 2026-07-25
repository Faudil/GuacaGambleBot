package delve

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"

	"guacagamblebot/internal/components"
)

type ChronicleEntry struct {
	Date     string
	RunNumber int
	Sentence string
	Loot     string
}

func BuildChronicle(userID int64, svc *Service, lang string) ([]*discordgo.MessageEmbed, error) {
	flags, err := svc.store.GetDelveFlags(userID)
	if err != nil {
		return nil, err
	}
	history, err := svc.store.GetDelveRunHistory(userID)
	if err != nil {
		return nil, err
	}

	if len(flags) == 0 && len(history) == 0 {
		return []*discordgo.MessageEmbed{
			components.Embed("📖 Personal Chronicle", "No chronicle entries yet. Begin your journey with `!delve start`!", 0x9b59b6),
		}, nil
	}

	totalRuns := len(history)

	var entries []string
	epithets := []string{}
	seenEpithets := map[string]bool{}

	for _, f := range flags {
		sentence := GetFlagSentence(f.FlagID)
		if sentence == "" {
			continue
		}
		epithet := GetFlagEpithet(f.FlagID)
		if epithet != "" && !seenEpithets[epithet] {
			epithets = append(epithets, epithet)
			seenEpithets[epithet] = true
		}
		dateStr := f.EarnedAt.Format("Jan 2, 2006")
		entry := fmt.Sprintf("**%s** — %s", dateStr, sentence)
		entries = append(entries, entry)
	}

	var pages []*discordgo.MessageEmbed

	title := "📖 Personal Chronicle"
	desc := &strings.Builder{}
	desc.WriteString(fmt.Sprintf("*%d descent(s) into the Undercroft*\n\n", totalRuns))
	if len(epithets) > 0 {
		desc.WriteString(fmt.Sprintf("**Known as:** %s\n\n", strings.Join(epithets, ", ")))
	}

	if len(entries) == 0 {
		desc.WriteString("*No notable events recorded yet. Venture deeper.*")
	} else {
		for _, e := range entries {
			desc.WriteString(e + "\n\n")
		}
	}

	embed := &discordgo.MessageEmbed{
		Title:       title,
		Description: desc.String(),
		Color:       0x9b59b6,
		Footer:      &discordgo.MessageEmbedFooter{Text: fmt.Sprintf("Your legend grows with every descent.")},
	}
	pages = append(pages, embed)

	return pages, nil
}
