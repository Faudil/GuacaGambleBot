package assets

import (
	"os"
	"path/filepath"

	"github.com/bwmarrin/discordgo"

	"guacagamblebot/internal/components"
)

// Dir is the assets directory, relative to the working directory. It is
// overridable so tests can point it at a temporary directory.
var Dir = "assets"

// Has reports whether an asset file (e.g. "npcs/elara.png") exists in the
// assets directory.
func Has(name string) bool {
	if name == "" {
		return false
	}
	_, err := os.Stat(filepath.Join(Dir, filepath.FromSlash(name)))
	return err == nil
}

// Image sets the embed's image to the given asset. The asset must be uploaded
// with the same message (see Response, EditResponse, Send), otherwise Discord
// cannot resolve the attachment:// URL. Missing assets degrade gracefully:
// the embed is left untouched.
func Image(embed *discordgo.MessageEmbed, name string) {
	if embed == nil || !Has(name) {
		return
	}
	embed.Image = &discordgo.MessageEmbedImage{URL: "attachment://" + filepath.Base(name)}
}

// Thumbnail sets the embed's thumbnail to the given asset. See Image for the
// attachment:// constraint.
func Thumbnail(embed *discordgo.MessageEmbed, name string) {
	if embed == nil || !Has(name) {
		return
	}
	embed.Thumbnail = &discordgo.MessageEmbedThumbnail{URL: "attachment://" + filepath.Base(name)}
}

// File opens an asset and wraps it as a discordgo attachment. It returns nil
// when the asset is missing so callers can fall back to a text-only embed.
func File(name string) *discordgo.File {
	if !Has(name) {
		return nil
	}
	f, err := os.Open(filepath.Join(Dir, filepath.FromSlash(name)))
	if err != nil {
		return nil
	}
	return &discordgo.File{
		Name:   filepath.Base(name),
		Reader: f,
	}
}

// Response builds an interaction response whose embed shows the given asset
// (image) and uploads the file alongside it. A missing asset yields a plain
// response with no image.
func Response(responseType discordgo.InteractionResponseType, embed *discordgo.MessageEmbed, comps []discordgo.MessageComponent, name string) *discordgo.InteractionResponse {
	return responseWith(responseType, embed, comps, name, true)
}

// ResponseThumbnail is Response but renders the asset as a thumbnail.
func ResponseThumbnail(responseType discordgo.InteractionResponseType, embed *discordgo.MessageEmbed, comps []discordgo.MessageComponent, name string) *discordgo.InteractionResponse {
	return responseWith(responseType, embed, comps, name, false)
}

func responseWith(responseType discordgo.InteractionResponseType, embed *discordgo.MessageEmbed, comps []discordgo.MessageComponent, name string, big bool) *discordgo.InteractionResponse {
	resp := components.InteractionResponse(responseType, embed, comps)
	file := File(name)
	if file == nil {
		return resp
	}
	resp.Data.Files = []*discordgo.File{file}
	if embed != nil && embed.Image == nil && embed.Thumbnail == nil {
		if big {
			embed.Image = &discordgo.MessageEmbedImage{URL: "attachment://" + file.Name}
		} else {
			embed.Thumbnail = &discordgo.MessageEmbedThumbnail{URL: "attachment://" + file.Name}
		}
	}
	return resp
}

// EditResponse builds a message edit whose embed shows the given asset and
// re-uploads the file. Discord drops attachments that are not included in an
// edit payload, so every edit that references the attachment must carry it.
func EditResponse(embed *discordgo.MessageEmbed, comps []discordgo.MessageComponent, name string) *discordgo.WebhookEdit {
	edit := components.WebhookEditResponse(embed, comps)
	file := File(name)
	if file == nil {
		return edit
	}
	edit.Files = []*discordgo.File{file}
	if embed != nil && embed.Image == nil && embed.Thumbnail == nil {
		embed.Image = &discordgo.MessageEmbedImage{URL: "attachment://" + file.Name}
	}
	return edit
}

// Send builds a channel message (prefix path) carrying the asset as image. See
// Response for the missing-asset fallback.
func Send(embed *discordgo.MessageEmbed, comps []discordgo.MessageComponent, name string) *discordgo.MessageSend {
	return sendWith(embed, comps, name, true)
}

// SendThumbnail is Send but renders the asset as a thumbnail.
func SendThumbnail(embed *discordgo.MessageEmbed, comps []discordgo.MessageComponent, name string) *discordgo.MessageSend {
	return sendWith(embed, comps, name, false)
}

func sendWith(embed *discordgo.MessageEmbed, comps []discordgo.MessageComponent, name string, big bool) *discordgo.MessageSend {
	send := &discordgo.MessageSend{Components: comps}
	if embed != nil {
		send.Embeds = []*discordgo.MessageEmbed{embed}
	}
	file := File(name)
	if file == nil {
		return send
	}
	send.Files = []*discordgo.File{file}
	if embed != nil && embed.Image == nil && embed.Thumbnail == nil {
		if big {
			embed.Image = &discordgo.MessageEmbedImage{URL: "attachment://" + file.Name}
		} else {
			embed.Thumbnail = &discordgo.MessageEmbedThumbnail{URL: "attachment://" + file.Name}
		}
	}
	return send
}
