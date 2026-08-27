package jobs

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"guacagamblebot/internal/config"
	"guacagamblebot/internal/model"
	"guacagamblebot/internal/store"
	"guacagamblebot/internal/testutil"
)

func testService(t *testing.T) (*Service, *store.Store) {
	d := testutil.NewDB(t)
	cfg := &config.Config{StartingBalance: 100, DailyAmount: 50}
	s := store.New(d, cfg)
	return New(s, cfg), s
}

func TestGetJobsDefaults(t *testing.T) {
	svc, _ := testService(t)
	res, err := svc.GetJobs(1)
	require.NoError(t, err)
	assert.Len(t, res.Jobs, len(JobNames))
	assert.Equal(t, len(JobNames), res.TotalLevel) // each defaults to level 1
}

func TestGetJobsWithData(t *testing.T) {
	svc, st := testService(t)
	err := st.DB.Create(&model.Job{UserID: 1, JobName: "miner", Level: 5, XP: 200}).Error
	require.NoError(t, err)

	res, err := svc.GetJobs(1)
	require.NoError(t, err)

	var miner *JobInfo
	for i, j := range res.Jobs {
		if j.Name == "miner" {
			miner = &res.Jobs[i]
			break
		}
	}
	require.NotNil(t, miner)
	assert.Equal(t, 5, miner.Level)
	assert.Equal(t, 200, miner.XP)
	assert.Equal(t, 300, miner.Next)
}

func TestGetJobsTotalLevel(t *testing.T) {
	svc, st := testService(t)
	for _, name := range JobNames {
		st.DB.Create(&model.Job{UserID: 1, JobName: name, Level: 3, XP: 0})
	}

	res, err := svc.GetJobs(1)
	require.NoError(t, err)
	assert.Equal(t, len(JobNames)*3, res.TotalLevel)
}

func TestXPForLevel(t *testing.T) {
	assert.Equal(t, 100, XPForLevel(1))
	assert.Equal(t, 300, XPForLevel(5))
	assert.Equal(t, 50, XPForLevel(0))
}
