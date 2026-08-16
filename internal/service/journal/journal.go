package journal

import (
	"log/slog"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"guacagamblebot/internal/model"
	"guacagamblebot/internal/store"
)

// Completion describes a step finished during CheckAndAdvance, so the UI can
// celebrate it.
type Completion struct {
	PathID      string
	PathEmoji   string
	StepTextKey string
	Reward      Reward
	AllDone     bool
}

// StepView is the per-step rendering data.
type StepView struct {
	TextKey    string
	Progress   int
	Target     int
	Done       bool
	Discovered bool
}

// PathView is the per-path rendering data.
type PathView struct {
	PathID    string
	Emoji     string
	TitleKey  string
	DescKey   string
	Steps     []StepView
	Completed int
	Total     int
	Rank      int
	Tracked   bool
	AllDone   bool
	HasRumor  bool // current step is surfaced
}

// JournalView is the full journal state for a player.
type JournalView struct {
	Paths           []PathView
	Completions     []Completion
	MasteryUnlocked bool
	MasteryNew      bool
}

// masteryReward is granted once when every path is completed.
var masteryReward = Reward{Money: 0, Crowns: 100, ItemIDs: []string{"mastery_medallion"}}

// The hidden achievement inserted when mastery is unlocked.
const masteryAchievementID = "journal_mastery"

// LeaderboardEntry is one player's standing on a path.
type LeaderboardEntry struct {
	UserID    int64
	StepIndex int
	Progress  int
	UpdatedAt time.Time
}

// Service runs the journal: evaluating checks, granting rewards, ranking.
type Service struct {
	store *store.Store
}

// New wires the journal into the store activity hook and returns a service.
func New(s *store.Store) *Service {
	svc := &Service{store: s}
	s.SetJournalFn(svc.OnActivity)
	return svc
}

// OnActivity is invoked by the store after any recorded activity; it re-checks
// every path so steps complete the moment their milestone is reached.
func (s *Service) OnActivity(userID int64, _ string, _ int) {
	if _, err := s.CheckAndAdvance(userID); err != nil {
		slog.Warn("journal: activity check failed", "user", userID, "error", err)
	}
}

// CheckAndAdvance re-evaluates all paths for the user, advancing and rewarding
// every step whose check now passes. It returns the newly completed steps and
// queues atmospheric scenes on rank-ups.
func (s *Service) CheckAndAdvance(userID int64) ([]Completion, error) {
	var completions []Completion
	oldRank := HighestRank(s.store, userID)
	for _, p := range GetPaths() {
		entry, err := s.ensureEntry(userID, p.ID)
		if err != nil {
			return completions, err
		}
		for entry.StepIndex < len(p.Steps) {
			step := p.Steps[entry.StepIndex]
			progress, target, done := step.Check(s.store, userID)
			if !done {
				s.updateProgress(userID, p.ID, entry.StepIndex, progress)
				break
			}
			if err := s.grantReward(userID, step.Reward); err != nil {
				slog.Warn("journal: reward failed", "user", userID, "path", p.ID, "error", err)
				break
			}
			entry.StepIndex++
			s.updateProgress(userID, p.ID, entry.StepIndex, 0)
			completions = append(completions, Completion{
				PathID: p.ID, PathEmoji: p.Emoji,
				StepTextKey: step.TextKey, Reward: step.Reward,
				AllDone: entry.StepIndex >= len(p.Steps),
			})
			slog.Info("journal: step completed", "user", userID, "path", p.ID,
				"step", entry.StepIndex, "target", target)
		}
	}
	newRank := HighestRank(s.store, userID)
	s.queueRankUpScenes(userID, oldRank, newRank)
	return completions, nil
}

// queueRankUpScenes pushes the Chronicler's introduction once the player
// reaches rank 2 on a path (his reveal only lands on players genuinely
// engaged with the Journal), and a lighter rank-up moment on other rank-ups.
func (s *Service) queueRankUpScenes(userID int64, oldRank, newRank int) {
	if newRank <= oldRank {
		return
	}
	if oldRank < 2 && newRank >= 2 {
		s.store.PushJournalScene(userID, store.JournalScene{
			Key: "journal.chronicler.sighting", DM: true,
		})
		return
	}
	s.store.PushJournalScene(userID, store.JournalScene{
		Key: "journal.rankup", Params: map[string]any{"rank": newRank},
	})
}

// View returns the journal with fresh progress. It also advances any steps
// whose milestones were reached since the last check and unlocks the secret
// Mastery legend when every path is completed.
func (s *Service) View(userID int64) (*JournalView, error) {
	completions, err := s.CheckAndAdvance(userID)
	if err != nil {
		return nil, err
	}
	v := &JournalView{Completions: completions}
	v.MasteryUnlocked = s.IsMasteryUnlocked(userID)
	if !v.MasteryUnlocked {
		v.MasteryNew, err = s.unlockMastery(userID)
		if err != nil {
			return nil, err
		}
		if v.MasteryNew {
			v.MasteryUnlocked = true
		}
	}
	for _, p := range GetPaths() {
		entry, err := s.ensureEntry(userID, p.ID)
		if err != nil {
			return nil, err
		}
		pv := PathView{
			PathID: p.ID, Emoji: p.Emoji, TitleKey: p.TitleKey, DescKey: p.DescKey,
			Total: len(p.Steps), Rank: RankFor(entry.StepIndex, len(p.Steps)),
			Tracked: entry.Tracked, AllDone: entry.StepIndex >= len(p.Steps),
		}
		for i, step := range p.Steps {
			sv := StepView{TextKey: step.TextKey}
			switch {
			case i < entry.StepIndex:
				sv.Done = true
				sv.Discovered = true
				pv.Completed++
			case i == entry.StepIndex:
				// The current step surfaces as a rumor once its Discover check
				// passes (or when it has none); before that it stays a mystery.
				if step.Discover == nil {
					sv.Discovered = true
				} else {
					_, _, ok := step.Discover(s.store, userID)
					sv.Discovered = ok
				}
				if sv.Discovered {
					sv.Progress, sv.Target, _ = step.Check(s.store, userID)
					if sv.Progress >= sv.Target {
						sv.Done = true
					}
				}
				pv.HasRumor = sv.Discovered
			}
			pv.Steps = append(pv.Steps, sv)
		}
		v.Paths = append(v.Paths, pv)
	}
	return v, nil
}

// Track toggles whether a path is pinned in the player's journal.
func (s *Service) Track(userID int64, pathID string) error {
	entry, err := s.ensureEntry(userID, pathID)
	if err != nil {
		return err
	}
	return s.store.DB.Model(&model.UserJournalEntry{}).
		Where("user_id = ? AND path_id = ?", userID, pathID).
		Update("tracked", !entry.Tracked).Error
}

// Leaderboard returns the top players of a path, ordered by steps completed,
// then progress, then who got there first.
func (s *Service) Leaderboard(pathID string, limit int) ([]LeaderboardEntry, error) {
	var rows []model.UserJournalEntry
	if err := s.store.DB.Where("path_id = ?", pathID).
		Order("step_index DESC, progress DESC, updated_at ASC").
		Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]LeaderboardEntry, 0, len(rows))
	for _, r := range rows {
		out = append(out, LeaderboardEntry{
			UserID: r.UserID, StepIndex: r.StepIndex, Progress: r.Progress, UpdatedAt: r.UpdatedAt,
		})
	}
	return out, nil
}

func (s *Service) ensureEntry(userID int64, pathID string) (*model.UserJournalEntry, error) {
	var e model.UserJournalEntry
	err := s.store.DB.Where("user_id = ? AND path_id = ?", userID, pathID).First(&e).Error
	if err == nil {
		return &e, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, err
	}
	return &e, s.store.DB.Create(&model.UserJournalEntry{UserID: userID, PathID: pathID, UpdatedAt: time.Now()}).Error
}

func (s *Service) updateProgress(userID int64, pathID string, stepIndex, progress int) {
	_ = s.store.DB.Model(&model.UserJournalEntry{}).
		Where("user_id = ? AND path_id = ?", userID, pathID).
		Updates(map[string]any{"step_index": stepIndex, "progress": progress, "updated_at": time.Now()}).Error
}

func (s *Service) grantReward(userID int64, r Reward) error {
	if r.Money > 0 {
		if _, err := s.store.UpdateBalance(userID, r.Money); err != nil {
			return err
		}
	}
	if r.Crowns > 0 {
		// GetBalance ensures the user row exists so the column update applies.
		if _, err := s.store.GetBalance(userID); err != nil {
			return err
		}
		if err := s.store.DB.Model(&model.User{}).
			Where("user_id = ?", userID).
			UpdateColumn("crowns", gorm.Expr("crowns + ?", r.Crowns)).Error; err != nil {
			return err
		}
	}
	for _, itemID := range r.ItemIDs {
		if err := s.store.DB.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}, {Name: "item_id"}},
			DoUpdates: clause.Assignments(map[string]any{"quantity": gorm.Expr("quantity + ?", 1)}),
		}).Create(&model.Inventory{UserID: userID, ItemID: itemID, Quantity: 1}).Error; err != nil {
			return err
		}
	}
	return nil
}

// RankFor maps completed/total steps to a rank 0-4.
func RankFor(completed, total int) int {
	switch {
	case completed <= 0:
		return 0
	case total <= 0:
		return 0
	case completed*4 < total: // < 25%
		return 1
	case completed*4 < total*2: // < 50%
		return 2
	case completed*4 < total*3: // < 75%
		return 3
	default:
		return 4
	}
}

// AllPathsDone reports whether the player completed every step of every path.
func (s *Service) AllPathsDone(userID int64) bool {
	for _, p := range GetPaths() {
		var entry model.UserJournalEntry
		if err := s.store.DB.Where("user_id = ? AND path_id = ?", userID, p.ID).First(&entry).Error; err != nil {
			return false
		}
		if entry.StepIndex < len(p.Steps) {
			return false
		}
	}
	return true
}

// RankOf returns the player's rank (0-4) on a single path.
func RankOf(st *store.Store, userID int64, pathID string) int {
	p := GetPath(pathID)
	if p == nil {
		return 0
	}
	var entry model.UserJournalEntry
	if err := st.DB.Where("user_id = ? AND path_id = ?", userID, pathID).First(&entry).Error; err != nil {
		return 0
	}
	return RankFor(entry.StepIndex, len(p.Steps))
}

// HighestRank returns the player's best rank across all paths (0 = none).
func HighestRank(st *store.Store, userID int64) int {
	best := 0
	for _, p := range GetPaths() {
		if r := RankOf(st, userID, p.ID); r > best {
			best = r
		}
	}
	return best
}

// RankedPaths returns the IDs of every path in which the player holds at
// least rank 1, in registry order.
func RankedPaths(st *store.Store, userID int64) []string {
	var out []string
	for _, p := range GetPaths() {
		if RankOf(st, userID, p.ID) >= 1 {
			out = append(out, p.ID)
		}
	}
	return out
}

// MetChronicler reports whether the player already spoke with the Chronicler
// (his one-time intro was revealed).
func MetChronicler(st *store.Store, userID int64) bool {
	var n int64
	st.DB.Table("user_npc_secrets").
		Where("user_id = ? AND npc_id = ? AND secret_id = ?", userID, ChroniclerID, ChroniclerIntroSecret).
		Count(&n)
	return n > 0
}

// IsMasteryUnlocked reports whether the secret Mastery legend was earned.
func (s *Service) IsMasteryUnlocked(userID int64) bool {
	var n int64
	s.store.DB.Model(&model.UserJournalMastery{}).Where("user_id = ?", userID).Count(&n)
	return n > 0
}

// unlockMastery grants the secret Mastery legend once, when every path is
// complete. It is idempotent: returns true only on the first unlock.
func (s *Service) unlockMastery(userID int64) (bool, error) {
	if !s.AllPathsDone(userID) {
		return false, nil
	}
	if s.IsMasteryUnlocked(userID) {
		return false, nil
	}
	if err := s.store.DB.Create(&model.UserJournalMastery{UserID: userID, UnlockedAt: time.Now()}).Error; err != nil {
		return false, err
	}
	if err := s.grantReward(userID, masteryReward); err != nil {
		return false, err
	}
	if err := s.store.DB.Create(&model.UserAchievement{UserID: userID, AchievementID: masteryAchievementID}).Error; err != nil {
		return false, err
	}
	slog.Info("journal: Mastery legend unlocked", "user", userID)
	return true, nil
}
