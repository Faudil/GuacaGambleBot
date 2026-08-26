package interaction

import (
	"encoding/json"
	"math"
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/require"

	"guacagamblebot/internal/achievement"
	"guacagamblebot/internal/i18n"
)

func TestSendAchievementsChunksUnderEmbedLimit(t *testing.T) {
	require.NoError(t, i18n.Load("../../locales"))

	ds, rt := newDeferTestSession(t)
	i := deferTestInteraction()
	i.Member = &discordgo.Member{User: &discordgo.User{ID: "7"}}

	all := achievement.All()
	require.Greater(t, len(all), achievementsPerEmbed, "fixture must span several embeds")

	SendAchievements(&Bot{Session: ds}, &discordgo.InteractionCreate{Interaction: i}, "en", all)

	// SendAchievements fires its follow-ups asynchronously (off the handler's
	// critical path), so wait for the expected batch count to land.
	expected := int(math.Ceil(float64(len(all)) / float64(achievementsPerEmbed)))
	var calls, bodies []string
	require.Eventually(t, func() bool {
		calls, bodies = rt.snapshot()
		return len(calls) == expected
	}, 2*time.Second, 5*time.Millisecond, "unlocks must be chunked into one follow-up per batch")
	for _, body := range bodies {
		var follow struct {
			Embeds []struct {
				Description string `json:"description"`
			} `json:"embeds"`
		}
		require.NoError(t, json.Unmarshal([]byte(body), &follow))
		require.Len(t, follow.Embeds, 1, "every chunk carries exactly one embed")
		require.LessOrEqual(t, len(follow.Embeds[0].Description), 4096,
			"no chunk may exceed Discord's embed description limit")
	}
}
