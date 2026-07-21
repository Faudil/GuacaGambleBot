package elosimulation

import (
	"log"
	"math"
	"time"

	"gorm.io/gorm"

	"guacagamblebot/internal/battle"
	"guacagamblebot/internal/model"
	"guacagamblebot/internal/store"
)

// Run starts a background loop that periodically picks two active pets with
// similar Elo, has them fight, and updates their ratings to normalise the Elo
// distribution. The loop runs until the process exits.
func Run(st *store.Store) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		simulateRound(st)
	}
}

func simulateRound(st *store.Store) {
	p1, p2, err := st.GetRandomActivePetPair(5, 500)
	if err != nil {
		return
	}
	if p2 == nil {
		return
	}

	result := battle.Simulate(toBattlePet(p1), toBattlePet(p2))

	var score float64
	if result.WinnerID == p1.ID {
		score = 1.0
	} else if result.WinnerID == p2.ID {
		score = 0.0
	} else {
		score = 0.5
	}

	K := 32.0
	e1 := 1.0 / (1.0 + math.Pow(10, float64(p2.Elo-p1.Elo)/400))
	e2 := 1.0 / (1.0 + math.Pow(10, float64(p1.Elo-p2.Elo)/400))
	d1 := int(K * (score - e1))
	d2 := int(K * ((1.0 - score) - e2))

	newElo1 := p1.Elo + d1
	newElo2 := p2.Elo + d2
	if newElo1 < 0 {
		newElo1 = 0
	}
	if newElo2 < 0 {
		newElo2 = 0
	}

	txErr := st.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.UserPet{}).
			Where("id = ?", p1.ID).Update("elo", newElo1).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.UserPet{}).
			Where("id = ?", p2.ID).Update("elo", newElo2).Error; err != nil {
			return err
		}
		return nil
	})
	if txErr != nil {
		log.Printf("elosimulation: failed to persist Elo update: %v", txErr)
	}
}

func toBattlePet(p *model.UserPet) *battle.BattlePet {
	return &battle.BattlePet{
		ID:       p.ID,
		Nickname: p.Nickname,
		Level:    p.Level,
		MaxHP:    p.MaxHP,
		HP:       p.HP,
		Atk:      p.Atk,
		Defense:  p.Defense,
		Speed:    p.Speed,
		DGE:      p.DGE,
		ACC:      p.ACC,
		CritC:    p.CritC,
		CritD:    p.CritD,
		SpcC:     p.SpcC,
	}
}
