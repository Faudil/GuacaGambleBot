package assets

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/require"
)

func setup(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	old := Dir
	Dir = dir
	t.Cleanup(func() { Dir = old })
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "npcs"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "npcs", "elara.png"), []byte("png"), 0o644))
	return dir
}

func TestHas(t *testing.T) {
	setup(t)
	require.True(t, Has("npcs/elara.png"))
	require.False(t, Has("npcs/missing.png"))
	require.False(t, Has(""))
}

func TestImageAndThumbnail(t *testing.T) {
	setup(t)
	embed := &discordgo.MessageEmbed{}
	Image(embed, "npcs/elara.png")
	require.Equal(t, "attachment://elara.png", embed.Image.URL)

	embed2 := &discordgo.MessageEmbed{}
	Thumbnail(embed2, "npcs/elara.png")
	require.Equal(t, "attachment://elara.png", embed2.Thumbnail.URL)

	// Missing assets leave the embed untouched.
	embed3 := &discordgo.MessageEmbed{}
	Image(embed3, "npcs/missing.png")
	require.Nil(t, embed3.Image)
}

func TestResponse(t *testing.T) {
	setup(t)
	embed := &discordgo.MessageEmbed{Title: "t"}
	resp := Response(discordgo.InteractionResponseUpdateMessage, embed, nil, "npcs/elara.png")
	require.Len(t, resp.Data.Files, 1)
	require.Equal(t, "elara.png", resp.Data.Files[0].Name)
	require.Equal(t, "attachment://elara.png", embed.Image.URL)

	thumb := ResponseThumbnail(discordgo.InteractionResponseUpdateMessage, &discordgo.MessageEmbed{Title: "t"}, nil, "npcs/elara.png")
	require.Len(t, thumb.Data.Files, 1)
	require.Equal(t, "attachment://elara.png", thumb.Data.Embeds[0].Thumbnail.URL)
}

func TestResponseMissingAsset(t *testing.T) {
	setup(t)
	embed := &discordgo.MessageEmbed{Title: "t"}
	resp := Response(discordgo.InteractionResponseUpdateMessage, embed, nil, "npcs/missing.png")
	require.Nil(t, resp.Data.Files)
	require.Nil(t, embed.Image)
	require.Nil(t, embed.Thumbnail)
	require.NotNil(t, resp.Data.Embeds, "plain embed must survive a missing asset")
}

func TestEditResponse(t *testing.T) {
	setup(t)
	embed := &discordgo.MessageEmbed{Title: "t"}
	edit := EditResponse(embed, nil, "npcs/elara.png")
	require.Len(t, edit.Files, 1)
	require.Equal(t, "attachment://elara.png", embed.Image.URL)
}

func TestSend(t *testing.T) {
	setup(t)
	embed := &discordgo.MessageEmbed{Title: "t"}
	send := Send(embed, nil, "npcs/elara.png")
	require.Len(t, send.Files, 1)
	require.Equal(t, "attachment://elara.png", embed.Image.URL)

	sendThumb := SendThumbnail(&discordgo.MessageEmbed{Title: "t"}, nil, "npcs/elara.png")
	require.Len(t, sendThumb.Files, 1)
	require.Equal(t, "attachment://elara.png", sendThumb.Embeds[0].Thumbnail.URL)
}

func TestSendMissingAsset(t *testing.T) {
	setup(t)
	embed := &discordgo.MessageEmbed{Title: "t"}
	send := Send(embed, nil, "npcs/missing.png")
	require.Nil(t, send.Files)
	require.Nil(t, embed.Image)
	require.Len(t, send.Embeds, 1)
}

func TestDirDefault(t *testing.T) {
	require.Equal(t, "assets", Dir)
}
