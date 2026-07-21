package boss

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"

	"guacagamblebot/internal/achievement"
	"guacagamblebot/internal/battle"
	"guacagamblebot/internal/components"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/interaction"
	"guacagamblebot/internal/model"
	bosssvc "guacagamblebot/internal/service/boss"
	petsvc "guacagamblebot/internal/service/pets"
	"guacagamblebot/internal/store"
)

type Cog struct {
	store *store.Store
	cfg   *config.Config
	svc   *bosssvc.Service
}

func Register(r *interaction.Router, s *store.Store, cfg *config.Config) {
	c := &Cog{store: s, cfg: cfg, svc: bosssvc.New(s, cfg)}
	r.Slash("boss", "Boss League - Combattez des boss", c.onSlash)
	r.Prefix("boss", c.onPrefix)
	r.Prefix("league", c.onPrefix)
}

func (c *Cog) onSlash(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	embed := c.show(userID, lang)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, nil))
}

func (c *Cog) onPrefix(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message) {
	lang := c.store.GetLanguage(interaction.ToInt64(m.GuildID))
	parts := strings.Fields(m.Content)
	sub := ""
	if len(parts) > 1 {
		sub = parts[1]
	}
	switch sub {
	case "fight":
		embed := c.fight(interaction.ToInt64(m.Author.ID), lang)
		_, _ = s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
			Embeds: []*discordgo.MessageEmbed{embed},
		})
	default:
		embed := c.show(interaction.ToInt64(m.Author.ID), lang)
		_, _ = s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
			Embeds: []*discordgo.MessageEmbed{embed},
		})
	}
}

func (c *Cog) show(userID int64, lang string) *discordgo.MessageEmbed {
	stage, err := c.svc.GetStage(userID)
	if err != nil {
		return components.Embed("❌", "Error loading boss data.", 0xe74c3c)
	}

	if stage >= 5 {
		return components.Embed(
			i18n.T("boss_league.title", lang),
			i18n.T("boss_league.champion", lang),
			0xf1c40f,
		)
	}

	boss := bosssvc.BossLeague[stage]
	bossName := boss.NameEN
	if lang == "fr" {
		bossName = boss.NameFR
	}
	bossDesc := boss.DescEN
	if lang == "fr" {
		bossDesc = boss.DescFR
	}

	statsTxt := fmt.Sprintf("Lvl %d | %s | HP: %d | ATK: %d | DEF: %d | SPD: %d",
		boss.Level, boss.Species, boss.HP, boss.Atk, boss.Defense, boss.Speed)

	rewardsTxt := fmt.Sprintf("💵 **$%d**\n", boss.RewardMoney)
	for item, qty := range boss.RewardItems {
		rewardsTxt += fmt.Sprintf("📦 %s x%d\n", item, qty)
	}

	embed := components.Embed(
		i18n.T("boss_league.title", lang),
		"",
		0x992d22,
	)
	embed.Fields = []*discordgo.MessageEmbedField{
		components.Field(
			i18n.T("boss_league.stage_info", lang, map[string]any{"stage": stage + 1, "name": bossName}),
			fmt.Sprintf("*%s*", bossDesc),
			false,
		),
		components.Field("Characteristics", fmt.Sprintf("🐾 %s", statsTxt), true),
		components.Field(i18n.T("boss_league.rewards_title", lang), rewardsTxt, true),
	}
	embed.Footer = &discordgo.MessageEmbedFooter{Text: i18n.T("boss_league.challenge_footer", lang)}

	return embed
}

func (c *Cog) fight(userID int64, lang string) *discordgo.MessageEmbed {
	stage, err := c.svc.GetStage(userID)
	if err != nil {
		return components.Embed("❌", "Error loading boss data.", 0xe74c3c)
	}

	if stage >= 5 {
		return components.Embed("❌", i18n.T("boss_league.already_champion", lang), 0xe74c3c)
	}

	pet, err := petsvc.New(c.store, c.cfg).GetActivePet(userID)
	if err != nil || pet == nil {
		return components.Embed("❌", i18n.T("boss_league.no_pet", lang), 0xe74c3c)
	}

	if pet.HP <= 0 {
		return components.Embed("❌", i18n.T("boss_league.pet_ko", lang, map[string]any{"name": pet.Nickname}), 0xe74c3c)
	}

	bossCfg := bosssvc.BossLeague[stage]
	bossPet := c.svc.CreateBossPet(bossCfg)

	userBP := petToBattlePet(pet)
	battle.Simulate(userBP, bossPet)
	_ = petsvc.New(c.store, c.cfg).UpdatePet(pet)

	if userBP.HP > 0 && bossPet.HP <= 0 {
		newStage := stage + 1
		_ = c.svc.SetStage(userID, newStage)
		_, _ = c.svc.UpdateBalance(userID, bossCfg.RewardMoney)

		_ = achievement.IncrementStat(c.svc.DB(), userID, "pve_wins", 1)
		unlocks, _ := achievement.CheckAndUnlock(c.svc.DB(), userID)

		desc := fmt.Sprintf("🏆 **Victory!** You defeated **%s**!\n\n💵 +$%d\n",
			bossCfg.NameEN, bossCfg.RewardMoney)
		for item, qty := range bossCfg.RewardItems {
			desc += fmt.Sprintf("📦 %s x%d\n", item, qty)
		}
		if newStage >= 5 {
			desc += "\n" + i18n.T("boss_league.champion", lang)
		}

		embed := components.Embed(i18n.T("boss_league.victory", lang, map[string]any{"boss_name": bossCfg.NameEN}), desc, 0x2ecc71)
		if len(unlocks) > 0 {
			achStr := ""
			for _, a := range unlocks {
				achStr += fmt.Sprintf("🎖️ %s (+%d Glory)\n", a.ID, a.Glory)
			}
			embed.Fields = append(embed.Fields, components.Field("🎖️ Achievements", achStr, false))
		}
		return embed
	}

	return components.Embed(
		i18n.T("boss_league.defeat", lang, map[string]any{"pet_name": pet.Nickname, "boss_name": bossCfg.NameEN}),
		"Train your pet and try again!",
		0xe74c3c,
	)
}

func petToBattlePet(pet *model.UserPet) *battle.BattlePet {
	pt := petsvc.PetTypes[pet.PetType]
	emoji := "🐾"
	if pt != nil {
		emoji = pt.Emoji
	}
	return &battle.BattlePet{
		ID: pet.ID, Nickname: pet.Nickname, Emoji: emoji,
		Level: pet.Level, HP: pet.HP, MaxHP: pet.MaxHP,
		Atk: pet.Atk, Defense: pet.Defense, Speed: pet.Speed,
		DGE: pet.DGE, ACC: pet.ACC, CritC: pet.CritC, CritD: pet.CritD, SpcC: pet.SpcC,
	}
}


