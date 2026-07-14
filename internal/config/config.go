package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	DiscordToken string
	TZ           string
	Environment  string
	DBPath       string
	GuildID      int64
	Prefix       string

	StartingBalance int
	DailyAmount     int
	ChannelID       int64
	TestChannelID   int64
	PetChannelID    int64
	BaseJackpot     int
}

func Load() *Config {
	_ = godotenv.Load()
	return &Config{
		DiscordToken:    os.Getenv("DISCORD_TOKEN"),
		TZ:              os.Getenv("TZ"),
		Environment:     os.Getenv("ENVIRONMENT"),
		DBPath:          getEnv("DB_PATH", "./data/guacabot_go.db"),
		GuildID:         getInt64("GUILD_ID", 0),
		Prefix:          getEnv("PREFIX", "!"),
		StartingBalance: getInt("STARTING_BALANCE", 100),
		DailyAmount:     getInt("DAILY_AMOUNT", 50),
		ChannelID:       getInt64("CHANNEL_ID", 1465882503045841156),
		TestChannelID:   getInt64("TEST_CHANNEL_ID", 1470452977587458334),
		PetChannelID:    getInt64("PET_CHANNEL_ID", 1478386999584096327),
		BaseJackpot:     getInt("BASE_JACKPOT", 500),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func getInt64(key string, fallback int64) int64 {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			return n
		}
	}
	return fallback
}
