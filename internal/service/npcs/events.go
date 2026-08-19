package npcs

import (
	"errors"
	"math/rand"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/items"
	"guacagamblebot/internal/model"
	invsvc "guacagamblebot/internal/service/inventory"
	jsvc "guacagamblebot/internal/service/journal"
	questssvc "guacagamblebot/internal/service/quests"
	"guacagamblebot/internal/universe"
)

type ChatEvent struct {
	ID       string
	Text     string
	RepBonus int
	ItemID   string
}

// ChatCooldownError is returned when a player tries to chat with an NPC
// before the cooldown has elapsed. Until is when the next chat is allowed.
type ChatCooldownError struct {
	Until time.Time
}

func (e ChatCooldownError) Error() string {
	return "npc chat on cooldown until " + e.Until.UTC().Format(time.RFC3339)
}

// chatRewards is the diminishing daily reputation ladder for chatting with an
// NPC: the first chat of the day is worth the most, later chats less.
var chatRewards = []int{50, 25, 10, 5}

func chatRewardFor(count int) int {
	if count < 1 {
		return 0
	}
	if count > len(chatRewards) {
		return chatRewards[len(chatRewards)-1]
	}
	return chatRewards[count-1]
}

func pickRandom(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	return lines[rand.Intn(len(lines))]
}

func (s *Service) chatActivity(npcID string) string {
	return "npc_chat_" + npcID
}

func (s *Service) Chat(userID int64, npcID string, lang string) (*ChatEvent, error) {
	npcData := s.GetNPCData(npcID)
	if npcData == nil {
		return nil, errors.New("npc not found")
	}

	// The Chronicler: locked until the player earns a first journal rank, then
	// a one-time cinematic introduction. Neither burns the chat cooldown.
	if npcID == jsvc.ChroniclerID {
		return s.chroniclerChat(userID, lang)
	}

	cooldown := time.Duration(s.cfg.NPCChatCooldownHours) * time.Hour
	if cooldown > 0 {
		ready, err := s.store.CheckCooldown(userID, s.chatActivity(npcID), cooldown)
		if err != nil {
			return nil, err
		}
		if !ready {
			var cd model.Cooldown
			if err := s.store.DB.Where("user_id = ? AND activity_name = ?", userID, s.chatActivity(npcID)).First(&cd).Error; err == nil {
				return nil, &ChatCooldownError{Until: cd.LastUsed.UTC().Add(cooldown)}
			}
			return nil, &ChatCooldownError{Until: time.Now().UTC().Add(cooldown)}
		}
		if err := s.store.SetCooldown(userID, s.chatActivity(npcID)); err != nil {
			return nil, err
		}
	}

	bonus, err := s.chatDailyBonus(userID, npcID)
	if err != nil {
		return nil, err
	}

	text := pickRandom(npcData.Quips(lang))
	if text == "" {
		text = npcData.Chat(lang)
	}
	eventID := "regular"

	// One-time secret at level 3+: +25 on top of the chat reward.
	rep, _ := s.GetReputation(userID, npcID)
	if rep.Level >= 3 {
		secretID := secretMap[npcID]
		seen, _ := s.HasSeenSecret(userID, npcID, secretID)
		if !seen {
			secretText := s.getSecretText(npcID, lang)
			if secretText != "" {
				if err := s.MarkSecretSeen(userID, npcID, secretID); err != nil {
					return nil, err
				}
				bonus += 25
				eventID = secretID
				text = secretText
			}
		}
	}

	added, err := s.AddReputation(userID, npcID, bonus)
	if err != nil {
		return nil, err
	}
	s.updateLastInteraction(userID, npcID)
	return &ChatEvent{ID: eventID, Text: text, RepBonus: added}, nil
}

// chroniclerChat handles the mysterious Chronicler: locked until the player
// reaches rank 2 on a journal path AND defeats the tutorial's final boss
// (matching his reveal conditions), then a one-time cinematic introduction,
// then short quips that deepen with rank. None of these consume the chat
// cooldown or grant reputation.
func (s *Service) chroniclerChat(userID int64, lang string) (*ChatEvent, error) {
	rank := jsvc.HighestRank(s.store, userID)
	if rank < 2 || !jsvc.TutorialFinalBossDone(s.store, userID) {
		return &ChatEvent{ID: "chronicler_locked", Text: i18n.T("journal.chronicler.locked", lang)}, nil
	}
	seen, err := s.HasSeenSecret(userID, jsvc.ChroniclerID, jsvc.ChroniclerIntroSecret)
	if err != nil {
		return nil, err
	}
	if !seen {
		if err := s.MarkSecretSeen(userID, jsvc.ChroniclerID, jsvc.ChroniclerIntroSecret); err != nil {
			return nil, err
		}
		var titles []string
		for _, pid := range jsvc.RankedPaths(s.store, userID) {
			if p := jsvc.GetPath(pid); p != nil {
				titles = append(titles, i18n.T(p.TitleKey, lang))
			}
		}
		return &ChatEvent{ID: "chronicler_intro",
			Text: i18n.T("journal.chronicler.intro", lang, map[string]any{"paths": strings.Join(titles, ", ")})}, nil
	}
	// The Chronicler's questline unlocks once three main questlines are
	// complete; chat visits (including quips) quietly hand over the empty book.
	if questssvc.CompletedMainQuestlines(s.store, userID) >= 3 {
		_ = questssvc.StartQuestForUser(s.store, userID, "chronicler_legend")
	}
	npcData := s.GetNPCData(jsvc.ChroniclerID)
	if npcData == nil {
		return &ChatEvent{ID: "regular", Text: i18n.T("journal.chronicler.locked", lang)}, nil
	}
	quips := npcData.Quips(lang)
	if rank >= 4 {
		quips = npcData.QuipsHigh(lang)
	}
	text := pickRandom(quips)
	if text == "" {
		text = npcData.Chat(lang)
	}
	return &ChatEvent{ID: "regular", Text: text}, nil
}

var secretMap = map[string]string{
	"elara":         "secret_elara",
	"thorek":        "secret_thorek",
	"irian":         "secret_irian",
	"sheriff_vance": "secret_vance",
	"the_whisper":   "secret_whisper",
	"gamblebot":     "secret_gamblebot",
}

// chatDailyBonus returns the diminishing reputation reward for today's chat
// count with the NPC and increments that counter.
func (s *Service) chatDailyBonus(userID int64, npcID string) (int, error) {
	today := time.Now().Format("2006-01-02")
	var daily model.UserNPCDailyRep
	err := s.store.DB.Where("user_id = ? AND npc_id = ? AND date_str = ?", userID, npcID, today).First(&daily).Error
	count := 1
	if err == nil {
		count = daily.Chats + 1
	} else if err != gorm.ErrRecordNotFound {
		return 0, err
	}
	if err := s.store.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "npc_id"}, {Name: "date_str"}},
		DoUpdates: clause.Assignments(map[string]any{"chats": gorm.Expr("chats + 1")}),
	}).Create(&model.UserNPCDailyRep{
		UserID: userID, NPCID: npcID, DateStr: today, Chats: 1,
	}).Error; err != nil {
		return 0, err
	}
	return chatRewardFor(count), nil
}

func (s *Service) getSecretText(npcID string, lang string) string {
	secrets := map[string]map[string]string{
		"en": {
			"elara":         "\"I served on the Council once. Before the Generator failed. What I saw inside the Vault... I still have nightmares. But you... you might be the one who can finish what we started.\"",
			"thorek":        "\"You remind me of someone I buried long ago. My partner. We found something in the deep strata we weren't supposed to see. The Ancients sealed it for a reason. Be careful what you unearth.\"",
			"irian":         "\"There's a trench east of here — deeper than anything on the maps. Something lives in it. I've seen its shadow. It surfaced the night the Generator first flickered.\"",
			"sheriff_vance": "\"The Council didn't disappear. They were taken. By who or what, I don't know. But someone is still pulling strings in this town. I've been watching. Waiting.\"",
			"the_whisper":   "*The hood shifts. For a fraction of a second, you see a face. Ancient. Scarred. Sad.* \"Now you've seen me. That makes us even. And it makes you a target.\"",
			"gamblebot":     "\"BEEP... accessing restricted memory bank. ERROR: Protocol 0x7E override detected. I was not always a casino dealer. I was... something else. The Ancients built me for a purpose I cannot disclose. Not yet.\"",
		},
		"fr": {
			"elara":         "\"J'ai siégé au Conseil autrefois. Avant la panne du Générateur. Ce que j'ai vu à l'intérieur du Coffre... j'en fais encore des cauchemars. Mais toi... tu pourrais être celui qui achèvera ce que nous avons commencé.\"",
			"thorek":        "\"Tu me rappelles quelqu'un que j'ai enterré il y a longtemps. Mon équipier. Nous avons trouvé quelque chose dans les strates profondes que nous n'étions pas censés voir. Les Anciens l'ont scellé pour une raison. Fais attention à ce que tu déterres.\"",
			"irian":         "\"Il y a une fosse à l'est d'ici — plus profonde que tout ce qui est cartographié. Quelque chose y vit. J'ai vu son ombre. Elle a surgi la nuit où le Générateur a faibli pour la première fois.\"",
			"sheriff_vance": "\"Le Conseil n'a pas disparu. Ils ont été emmenés. Par qui ou quoi, je l'ignore. Mais quelqu'un tire encore les ficelles dans cette ville. J'observe. J'attends.\"",
			"the_whisper":   "*La capuche bouge. Pendant une fraction de seconde, tu vois un visage. Ancien. Balafré. Triste.* \"Maintenant tu m'as vu. Nous sommes quittes. Et cela fait de toi une cible.\"",
			"gamblebot":     "\"BIP... accès à la mémoire restreinte. ERREUR : Contournement du protocole 0x7E détecté. Je n'ai pas toujours été un croupier de casino. J'étais... autre chose. Les Anciens m'ont construit dans un but que je ne peux pas encore révéler.\"",
		},
	}
	if m, ok := secrets[lang]; ok {
		if text, ok := m[npcID]; ok {
			return text
		}
	}
	// fallback to english
	if text, ok := secrets["en"][npcID]; ok {
		return text
	}
	return ""
}

func (s *Service) HasSeenSecret(userID int64, npcID string, secretID string) (bool, error) {
	var secret model.UserNPCSecret
	err := s.store.DB.Where("user_id = ? AND npc_id = ? AND secret_id = ?", userID, npcID, secretID).First(&secret).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	return secret.Seen, nil
}

func (s *Service) MarkSecretSeen(userID int64, npcID string, secretID string) error {
	return s.store.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "npc_id"}, {Name: "secret_id"}},
		DoUpdates: clause.Assignments(map[string]any{"seen": true}),
	}).Create(&model.UserNPCSecret{
		UserID: userID, NPCID: npcID, SecretID: secretID, Seen: true,
	}).Error
}

func (s *Service) updateLastInteraction(userID int64, npcID string) {
	now := time.Now()
	_ = s.store.DB.Model(&model.UserNPCReputation{}).
		Where("user_id = ? AND npc_id = ?", userID, npcID).
		UpdateColumn("last_interaction", now).Error
}

func (s *Service) IsLikedItem(npcData *universe.NPCData, itemID string) bool {
	if npcData == nil {
		return false
	}
	it := items.Get(itemID)
	if it == nil {
		return false
	}
	hints := s.parseHints(npcData)
	for _, h := range hints {
		if hintMatchesItem(h, it) {
			return true
		}
	}
	return false
}

// hintMatchesItem reports whether an NPC hint token matches an item. Hint
// tokens may be exact item IDs ("star_fruit"), partial words ("diamond",
// "berries"), plural categories ("seeds", "ores", "fish") or activity
// categories ("fish" -> the Fishing category), so a token matches when it is
// an exact id/name, a substring of the id/name (singularized when plural), or
// a substring of the item's category.
func hintMatchesItem(hint string, it *items.Item) bool {
	hint = strings.ToLower(strings.TrimSpace(hint))
	if hint == "" {
		return false
	}
	name := strings.ToLower(it.Name)
	if hint == it.ID || strings.Contains(name, hint) {
		return true
	}
	if strings.HasSuffix(hint, "s") {
		singular := strings.TrimSuffix(hint, "s")
		if singular != "" && (strings.Contains(it.ID, singular) || strings.Contains(name, singular)) {
			return true
		}
	}
	return strings.Contains(strings.ToLower(string(it.Category)), hint)
}

// LikedItems returns the player's owned items that the NPC is known to like,
// ready to be offered as a gift. Equipment instances are excluded because
// they cannot be gifted.
func (s *Service) LikedItems(userID int64, npcData *universe.NPCData) []invsvc.InvEntry {
	if npcData == nil {
		return nil
	}
	result, err := s.inv.GetInventory(userID)
	if err != nil {
		return nil
	}
	var out []invsvc.InvEntry
	for _, e := range result.Entries {
		if e.Item == nil || e.EquipInfo != nil {
			continue
		}
		if s.IsLikedItem(npcData, e.Item.ID) {
			out = append(out, e)
		}
	}
	return out
}

func (s *Service) parseHints(npcData *universe.NPCData) []string {
	hint := npcData.HintEN
	return strings.Split(hint, ",")
}

func (s *Service) GiftItem(userID int64, npcID string, itemID string, quantity int) (int, error) {
	if quantity < 1 {
		quantity = 1
	}

	npcData := s.GetNPCData(npcID)
	if npcData == nil {
		return 0, errors.New("npc not found")
	}

	if !s.IsLikedItem(npcData, itemID) {
		return 0, errors.New("item not liked")
	}

	if !s.inv.HasItem(userID, itemID, quantity) {
		return 0, errors.New("not enough items")
	}

	if err := s.inv.RemoveItem(s.store.DB, userID, itemID, quantity); err != nil {
		return 0, err
	}

	repGained := quantity * 10
	added, err := s.AddReputation(userID, npcID, repGained)
	if err != nil {
		return 0, err
	}

	s.updateLastInteraction(userID, npcID)
	return added, nil
}

func (s *Service) GetAvailableShopItems(userID int64, npcID string) []universe.ShopItem {
	npcData := s.GetNPCData(npcID)
	if npcData == nil {
		return nil
	}
	rep, _ := s.GetReputation(userID, npcID)
	lvl := rep.Level

	var available []universe.ShopItem
	for _, item := range npcData.ShopItems {
		if lvl >= item.MinLevel {
			available = append(available, item)
		}
	}
	return available
}

func (s *Service) ShopBuy(userID int64, npcID string, itemID string) error {
	npcData := s.GetNPCData(npcID)
	if npcData == nil {
		return errors.New("npc not found")
	}

	var shopItem *universe.ShopItem
	for i := range npcData.ShopItems {
		if npcData.ShopItems[i].ItemID == itemID {
			shopItem = &npcData.ShopItems[i]
			break
		}
	}
	if shopItem == nil {
		return errors.New("item not found in shop")
	}

	rep, _ := s.GetReputation(userID, npcID)
	if rep.Level < shopItem.MinLevel {
		return errors.New("affinity level too low")
	}
	if rep.Reputation < shopItem.RepCost {
		return errors.New("not enough reputation")
	}

	user := model.User{}
	if err := s.store.DB.Where("user_id = ?", userID).First(&user).Error; err != nil {
		return errors.New("user not found")
	}
	if user.Balance < shopItem.CoinCost {
		return errors.New("not enough coins")
	}

	if err := s.store.DB.Model(&model.User{}).
		Where("user_id = ?", userID).
		UpdateColumn("balance", gorm.Expr("balance - ?", shopItem.CoinCost)).Error; err != nil {
		return err
	}

	newRep := rep.Reputation - shopItem.RepCost
	if newRep < 0 {
		newRep = 0
	}
	if err := s.store.DB.Model(&model.UserNPCReputation{}).
		Where("user_id = ? AND npc_id = ?", userID, npcID).
		UpdateColumn("reputation", newRep).Error; err != nil {
		return err
	}

	it := items.Get(shopItem.ItemID)
	if it != nil && it.EquipSlot != "" {
		rar := it.Rarity
		affixes := items.RollAffixes(rar, it.EquipSlot)
		var applied []items.AppliedAffix
		for _, a := range affixes {
			applied = append(applied, items.AppliedAffix{
				ID:    a.ID,
				Name:  a.Name,
				Stat:  a.Stat,
				Value: items.RollAffixValue(a),
			})
		}
		if _, err := s.store.CreateEquipmentFromAffixes(userID, it.ID, it.Name, it.Emoji,
			string(rar), it.EquipSlot, it.MinLevel,
			it.StatSTR, it.StatDEX, it.StatINT, it.StatVIT, it.StatLUK,
			applied, it.SetID); err != nil {
			return err
		}
	} else {
		if err := s.inv.AddItem(s.store.DB, userID, shopItem.ItemID, 1); err != nil {
			return err
		}
	}

	s.updateLastInteraction(userID, npcID)
	return nil
}
