package sanctuary

import (
	"testing"

	"github.com/bwmarrin/discordgo"

	"guacagamblebot/internal/model"
)

// A fully-built sanctuary (tier 2, pets in sanctuary, room to grow, upgrade
// available) offers 7 actions: Collect, Recall, Retire, Upgrade, Showcase,
// Fusion, Ascend. Discord rejects any action row with more than 5 buttons,
// so buildButtons must spread them across rows instead of packing one row.
func TestBuildButtonsSplitsRowsAtDiscordLimit(t *testing.T) {
	c := &Cog{}
	san := &model.UserSanctuary{Tier: 2}
	rows := c.buildButtons(san, 2, 9, 15, "en", 1)

	total := 0
	for _, r := range rows {
		row, ok := r.(discordgo.ActionsRow)
		if !ok {
			t.Fatalf("expected ActionsRow, got %T", r)
		}
		if len(row.Components) > 5 {
			t.Fatalf("action row has %d buttons, Discord's limit is 5", len(row.Components))
		}
		total += len(row.Components)
	}
	if total != 7 {
		t.Fatalf("expected 7 buttons total, got %d", total)
	}
}
