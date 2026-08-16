package journal

import (
	"math/rand"
	"strconv"
	"time"

	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/store"
)

// The Chronicler: a mysterious NPC who notices players once they reach their
// first journal rank. His NPC definitions live in each universe's npcs.go.
const (
	ChroniclerID          = "the_chronicler"
	ChroniclerIntroSecret = "secret_chronicler_intro"

	// chroniclerSightingKey is the queued scene key for the reveal DM.
	chroniclerSightingKey = "journal.chronicler.sighting"

	// ChroniclerSightingSecret marks that the reveal DM was delivered, so it
	// is never queued or sent twice.
	ChroniclerSightingSecret = "secret_chronicler_sighting"
)

// domainPaths maps activity domains (used by the activity cogs) to the journal
// paths whose rank enables recognition in that domain.
var domainPaths = map[string][]string{
	"mining":     {"prospector"},
	"fishing":    {"prospector"},
	"farm":       {"prospector"},
	"hunt":       {"prospector", "hunter"},
	"archeology": {"prospector", "historian"},
	"casino":     {"highroller"},
	"lotto":      {"highroller"},
	"market":     {"merchant"},
	"delve":      {"champion"},
	"pets":       {"builder"},
	"housing":    {"builder"},
	"lore":       {"historian"},
}

var sceneRandom = rand.New(rand.NewSource(time.Now().UnixNano()))

// sceneRoll is the probability hook (0-100) used by ambient/recognition scenes;
// overridable in tests.
var sceneRoll = func(pct int) bool { return randPct(pct) }

func itoa(n int) string { return strconv.Itoa(n) }

func randPct(n int) bool {
	if n >= 100 {
		return true
	}
	return sceneRandom.Intn(100) < n
}

// SceneLine is the package entry point used by activity cogs. It returns the
// localized atmospheric line to surface after an activity ("" when nothing):
// first a queued scene (Chronicler sighting, rank-up moment), then ambient
// sightings while the Chronicler is still a stranger, then recognition lines
// for players ranked in the domain's paths. dm reports that the line should
// preferably be delivered as a private message.
func SceneLine(st *store.Store, userID int64, domain, lang string) (text string, dm bool) {
	// 1. Queued scenes first.
	if sc, ok := st.PopJournalScene(userID); ok {
		if sc.Key == chroniclerSightingKey {
			markSightingDelivered(st, userID)
		}
		return i18n.T(sc.Key, lang, sc.Params), sc.DM
	}
	// 2. Ambient sightings until the first meeting.
	if !MetChronicler(st, userID) {
		if ready, _ := st.CheckCooldown(userID, "journal_ambient", 45*time.Minute); !ready {
			return "", false
		}
		if !sceneRoll(20) {
			return "", false
		}
		_ = st.SetCooldown(userID, "journal_ambient")
		n := sceneRandom.Intn(3) + 1
		return i18n.T("journal.ambient."+itoa(n), lang), false
	}
	// 3. Recognition for ranked players in this domain.
	pathID := rankedPathFor(st, userID, domain)
	if pathID == "" {
		return "", false
	}
	if ready, _ := st.CheckCooldown(userID, "journal_recognition", 30*time.Minute); !ready {
		return "", false
	}
	if !sceneRoll(20) {
		return "", false
	}
	_ = st.SetCooldown(userID, "journal_recognition")
	rank := RankOf(st, userID, pathID)
	p := GetPath(pathID)
	if p == nil {
		return "", false
	}
	return i18n.T("journal.recognition."+itoa(rank), lang, map[string]any{
		"path": i18n.T(p.TitleKey, lang),
	}), false
}

// rankedPathFor returns the first path linked to the domain in which the player
// holds at least rank 1, or "".
func rankedPathFor(st *store.Store, userID int64, domain string) string {
	for _, pid := range domainPaths[domain] {
		if RankOf(st, userID, pid) >= 1 {
			return pid
		}
	}
	return ""
}
