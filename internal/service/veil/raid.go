package veil

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"guacagamblebot/internal/config"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/model"
	"guacagamblebot/internal/store"
)

type Service struct {
	store *store.Store
	cfg   *config.Config
}

func New(s *store.Store, cfg *config.Config) *Service {
	return &Service{store: s, cfg: cfg}
}

func (svc *Service) Store() *store.Store { return svc.store }

func (svc *Service) currentWeekStart() string {
	now := time.Now()
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	monday := now.AddDate(0, 0, -(weekday - 1))
	return monday.Format("2006-01-02")
}

func (svc *Service) ValidateGate(userID int64, lang string) (bool, string) {
	pet, err := svc.getActivePet(userID)
	if err != nil || pet == nil || pet.Level < 40 {
		return false, i18n.T("veil.gate.err_pet", lang)
	}

	char, err := svc.store.EnsureCharacter(userID)
	if err != nil || char.Level < 50 {
		return false, i18n.T("veil.gate.err_char", lang)
	}

	ok, _ := svc.store.HasItem(userID, "veil_key", 1)
	if !ok {
		return false, i18n.T("veil.gate.err_key_detail", lang)
	}

	return true, ""
}

func (svc *Service) getActivePet(userID int64) (*model.UserPet, error) {
	var pets []model.UserPet
	err := svc.store.DB.Where("user_id = ? AND is_active = ?", userID, true).Find(&pets).Error
	if len(pets) > 0 {
		return &pets[0], nil
	}
	return nil, err
}

func (svc *Service) HasItem(userID int64, itemID string, qty int) (bool, error) {
	return svc.store.HasItem(userID, itemID, qty)
}

func (svc *Service) ConsumeVeilKey(userID int64) error {
	return svc.store.DB.Where("user_id = ? AND item_id = 'veil_key'", userID).
		UpdateColumn("quantity", nil).Error
}

func (svc *Service) CreateRaid(leaderID, guildID int64, lang string) (*model.VeilRaid, error) {
	ok, msg := svc.ValidateGate(leaderID, lang)
	if !ok {
		return nil, errors.New(msg)
	}

	ok, err := svc.HasItem(leaderID, "veil_key", 1)
	if err != nil || !ok {
		return nil, errors.New(i18n.T("veil.gate.err_key", lang))
	}

	if err := svc.ConsumeVeilKey(leaderID); err != nil {
		return nil, errors.New(i18n.T("veil.gate.err_consume", lang))
	}

	ids := []int64{leaderID}
	b, _ := json.Marshal(ids)
	states := map[int64]model.VeilPlayerState{
		leaderID: {
			UserID: leaderID,
			HP:     200,
			MaxHP:  200,
			Status: "active",
		},
	}
	psb, _ := json.Marshal(states)

	raid := &model.VeilRaid{
		GuildID:        guildID,
		LeaderID:       leaderID,
		Status:         "forming",
		Phase:          "whispering",
		ParticipantIDs: string(b),
		PlayerStates:   string(psb),
	}
	return raid, svc.store.CreateVeilRaid(raid)
}

func (svc *Service) JoinRaid(raid *model.VeilRaid, userID int64, lang string) error {
	ok, msg := svc.ValidateGate(userID, lang)
	if !ok {
		return errors.New(msg)
	}

	var ids []int64
	store.UnmarshalJSON(raid.ParticipantIDs, &ids)
	if len(ids) >= 6 {
		return errors.New(i18n.T("veil.gate.err_full", lang))
	}
	for _, id := range ids {
		if id == userID {
			return errors.New(i18n.T("veil.gate.err_already_joined", lang))
		}
	}

	ok, err := svc.HasItem(userID, "veil_key", 1)
	if err != nil || !ok {
		return errors.New(i18n.T("veil.gate.err_key", lang))
	}
	if err := svc.ConsumeVeilKey(userID); err != nil {
		return errors.New(i18n.T("veil.gate.err_consume", lang))
	}

	ids = append(ids, userID)
	b, _ := json.Marshal(ids)
	raid.ParticipantIDs = string(b)

	var states map[int64]model.VeilPlayerState
	json.Unmarshal([]byte(raid.PlayerStates), &states)
	states[userID] = model.VeilPlayerState{
		UserID: userID,
		HP:     200,
		MaxHP:  200,
		Status: "active",
	}
	psb, _ := json.Marshal(states)
	raid.PlayerStates = string(psb)

	return svc.store.SaveVeilRaid(raid)
}

func (svc *Service) StartRaid(raid *model.VeilRaid, leaderID int64, lang string) error {
	if raid.LeaderID != leaderID {
		return errors.New(i18n.T("veil.gate.err_not_leader", lang))
	}
	var ids []int64
	store.UnmarshalJSON(raid.ParticipantIDs, &ids)
	if len(ids) < 3 {
		return fmt.Errorf("%s", i18n.T("veil.gate.err_min_players", lang, map[string]any{"count": len(ids)}))
	}

	scaled := ScaleForPlayers(len(ids))
	raid.BossMaxHP = int(float64(1500) * scaled)
	raid.BossHP = raid.BossMaxHP
	raid.BossImage = "bosses/vault_guardian.png"
	raid.Status = "active"
	raid.Phase = "whispering"

	return svc.store.SaveVeilRaid(raid)
}

func (svc *Service) GetPlayerStates(raid *model.VeilRaid) map[int64]model.VeilPlayerState {
	states := map[int64]model.VeilPlayerState{}
	json.Unmarshal([]byte(raid.PlayerStates), &states)
	return states
}

func (svc *Service) SetPlayerStates(raid *model.VeilRaid, states map[int64]model.VeilPlayerState) {
	b, _ := json.Marshal(states)
	raid.PlayerStates = string(b)
}

func (svc *Service) GetParticipants(raid *model.VeilRaid) []int64 {
	var ids []int64
	store.UnmarshalJSON(raid.ParticipantIDs, &ids)
	return ids
}

func GetParticipantsWith(raid *model.VeilRaid) []int64 {
	var ids []int64
	store.UnmarshalJSON(raid.ParticipantIDs, &ids)
	return ids
}

func (svc *Service) EndRaid(raid *model.VeilRaid, outcome string) error {
	raid.Status = outcome
	if err := svc.store.SaveVeilRaid(raid); err != nil {
		return err
	}
	lockout := &model.VeilRaidLockout{
		UserID:    raid.LeaderID,
		WeekStart: svc.currentWeekStart(),
		Cleared:   outcome == "completed",
	}
	for _, pid := range svc.GetParticipants(raid) {
		lockout.UserID = pid
		svc.store.UpsertVeilRaidLockout(lockout)
	}
	return nil
}

func ScaleForPlayers(n int) float64 {
	return 0.5 + float64(n)/6.0
}

func (svc *Service) HasHelperLockout(userID int64) (bool, error) {
	lockout, err := svc.store.GetVeilRaidLockout(userID, svc.currentWeekStart())
	if err != nil || lockout == nil {
		return false, nil
	}
	return lockout.HelpedAt != nil, nil
}

func (svc *Service) MarkHelper(userID int64) error {
	lockout := &model.VeilRaidLockout{
		UserID:    userID,
		WeekStart: svc.currentWeekStart(),
		Cleared:   true,
	}
	now := time.Now()
	lockout.HelpedAt = &now
	return svc.store.UpsertVeilRaidLockout(lockout)
}
