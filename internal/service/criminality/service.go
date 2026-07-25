package criminality

import (
	"math/rand"
	"sync"

	"guacagamblebot/internal/config"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/store"
)

type Service struct {
	store *store.Store
	cfg   *config.CriminalityConfig
	mu    sync.RWMutex
	rng   *rand.Rand
}

func New(s *store.Store, cfg *config.Config) *Service {
	return &Service{
		store: s,
		cfg:   &cfg.Criminality,
		rng:   rand.New(rand.NewSource(rand.Int63())),
	}
}

func (svc *Service) Store() *store.Store { return svc.store }
func (svc *Service) Cfg() *config.CriminalityConfig { return svc.cfg }

// T resolves a criminality translation key for the given language.
func (svc *Service) T(lang, key string, params ...map[string]any) string {
	return i18n.T("criminality."+key, lang, params...)
}

// ErrorMsg returns a localized error string (prefixless).
func (svc *Service) ErrorMsg(lang, key string, params ...map[string]any) string {
	return i18n.T("criminality.error."+key, lang, params...)
}
