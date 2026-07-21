package jobs

import (
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/model"
	"guacagamblebot/internal/store"
)

var JobNames = []string{"miner", "fisher", "farmer", "gambler", "crafter", "archeologist", "hunter"}

func XPForLevel(level int) int {
	return 100 * level
}

type JobInfo struct {
	Name  string
	Level int
	XP    int
	Next  int
}

type JobsResult struct {
	Jobs       []JobInfo
	TotalLevel int
}

type Service struct {
	store *store.Store
	cfg   *config.Config
}

func New(s *store.Store, cfg *config.Config) *Service {
	return &Service{store: s, cfg: cfg}
}

func (s *Service) GetJobs(userID int64) (*JobsResult, error) {
	var jobs []model.Job
	if err := s.store.DB.Where("user_id = ?", userID).Find(&jobs).Error; err != nil {
		return nil, err
	}
	jobMap := make(map[string]model.Job, len(jobs))
	for _, j := range jobs {
		jobMap[j.JobName] = j
	}
	result := &JobsResult{}
	total := 0
	for _, name := range JobNames {
		j, ok := jobMap[name]
		lvl := 1
		xp := 0
		if ok {
			lvl = j.Level
			xp = j.XP
		}
		total += lvl
		result.Jobs = append(result.Jobs, JobInfo{
			Name:  name,
			Level: lvl,
			XP:    xp,
			Next:  XPForLevel(lvl),
		})
	}
	result.TotalLevel = total
	return result, nil
}
