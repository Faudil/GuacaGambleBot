package delve

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"

	"guacagamblebot/internal/components"
	"guacagamblebot/internal/i18n"
)

type ChronicleEntry struct {
	Date      string
	RunNumber int
	Sentence  string
	Loot      string
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
			components.Embed(i18n.T("delve.chronicle.title", lang), i18n.T("delve.chronicle.empty", lang), components.ColorArcane),
		}, nil
	}

	totalRuns := len(history)

	var entries []string
	epithets := []string{}
	seenEpithets := map[string]bool{}

	for _, f := range flags {
		sentence := GetFlagSentence(f.FlagID, lang)
		if sentence == "" {
			continue
		}
		epithet := GetFlagEpithet(f.FlagID, lang)
		if epithet != "" && !seenEpithets[epithet] {
			epithets = append(epithets, epithet)
			seenEpithets[epithet] = true
		}
		dateStr := f.EarnedAt.Format("Jan 2, 2006")
		entry := fmt.Sprintf("**%s** — %s", dateStr, sentence)
		entries = append(entries, entry)
	}

	var pages []*discordgo.MessageEmbed

	title := i18n.T("delve.chronicle.title", lang)
	desc := &strings.Builder{}
	desc.WriteString(i18n.T("delve.chronicle.runs", lang, map[string]any{"n": fmt.Sprintf("%d", totalRuns)}) + "\n\n")
	if len(epithets) > 0 {
		desc.WriteString(i18n.T("delve.chronicle.known_as", lang, map[string]any{"names": strings.Join(epithets, ", ")}) + "\n\n")
	}

	if len(entries) == 0 {
		desc.WriteString(i18n.T("delve.chronicle.no_events", lang))
	} else {
		for _, e := range entries {
			desc.WriteString(e + "\n\n")
		}
	}

	embed := &discordgo.MessageEmbed{
		Title:       title,
		Description: desc.String(),
		Color:       components.ColorArcane,
		Footer:      &discordgo.MessageEmbedFooter{Text: i18n.T("delve.chronicle.footer", lang)},
	}
	pages = append(pages, embed)

	return pages, nil
}
