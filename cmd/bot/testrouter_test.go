package main

import (
	"testing"

	"github.com/bwmarrin/discordgo"

	"guacagamblebot/internal/config"
	"guacagamblebot/internal/interaction"
	"guacagamblebot/internal/store"
	"guacagamblebot/internal/testutil"
	"guacagamblebot/internal/universe/hoakhaven"
	"guacagamblebot/internal/universe/scifi"
	"guacagamblebot/internal/universe/scorch"
)

// buildTestRouter wires every cog through the same cogGroups table main uses,
// so these tests inspect exactly the command set the running bot registers.
func buildTestRouter(t *testing.T) *interaction.Router {
	t.Helper()
	hoakhaven.Register()
	scifi.Register()
	scorch.Register()
	cfg := &config.Config{Prefix: "!"}
	str := store.New(testutil.NewDB(t), cfg)
	router := interaction.NewRouter(&interaction.Bot{Session: &discordgo.Session{}, Prefix: "!"}, str)
	for _, group := range cogGroups() {
		router.Categorize(group.category, func() {
			for _, register := range group.cogs {
				register(router, str, cfg)
			}
		})
	}
	return router
}
