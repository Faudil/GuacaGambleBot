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
	LogLevel     string
	LogFile      string
	LogFormat    string
	LogAddSource bool
	Universe     string

	StartingBalance int
	DailyAmount     int
	ChannelID       int64
	TestChannelID   int64
	PetChannelID    int64
	BaseJackpot     int

	HuntMaxPerDay       int
	HuntCooldownSeconds int

	NPCChatCooldownHours int

	Criminality CriminalityConfig
}

type CriminalityConfig struct {
	StealMaxGoldPercent    float64
	StealMaxPerDay         int
	StealCooldownHours     int
	BurgleCooldownDays     int
	HuntCooldownHours      int
	NotorietyDecayDaily    int
	NotorietyHuntThreshold int
	MinLevelToTarget       int
	BountyHunterLicense    int
	CleanSlateGoldPrice    int
	PacifistGoldPrice      int
}

func Load() *Config {
	_ = godotenv.Load()
	return &Config{
		DiscordToken:         os.Getenv("DISCORD_TOKEN"),
		TZ:                   os.Getenv("TZ"),
		Environment:          os.Getenv("ENVIRONMENT"),
		DBPath:               getEnv("DB_PATH", "./data/guacabot_go.db"),
		GuildID:              getInt64("GUILD_ID", 0),
		Prefix:               getEnv("PREFIX", "!"),
		LogLevel:             getEnv("LOG_LEVEL", "info"),
		LogFile:              os.Getenv("LOG_FILE"),
		LogFormat:            getEnv("LOG_FORMAT", "text"),
		LogAddSource:         os.Getenv("LOG_ADD_SOURCE") == "true",
		Universe:             getEnv("UNIVERSE", "hoakhaven"),
		StartingBalance:      getInt("STARTING_BALANCE", 100),
		DailyAmount:          getInt("DAILY_AMOUNT", 50),
		ChannelID:            getInt64("CHANNEL_ID", 0),
		TestChannelID:        getInt64("TEST_CHANNEL_ID", 0),
		PetChannelID:         getInt64("PET_CHANNEL_ID", 0),
		BaseJackpot:          getInt("BASE_JACKPOT", 500),
		HuntMaxPerDay:        getInt("HUNT_MAX_PER_DAY", 10),
		HuntCooldownSeconds:  getInt("HUNT_COOLDOWN_SECONDS", 10),
		NPCChatCooldownHours: getInt("NPC_CHAT_COOLDOWN_HOURS", 6),
		Criminality: CriminalityConfig{
			StealMaxGoldPercent:    0.05,
			StealMaxPerDay:         3,
			StealCooldownHours:     24,
			BurgleCooldownDays:     7,
			HuntCooldownHours:      8,
			NotorietyDecayDaily:    5,
			NotorietyHuntThreshold: 20,
			MinLevelToTarget:       10,
			BountyHunterLicense:    500,
			CleanSlateGoldPrice:    50000,
			PacifistGoldPrice:      10000,
		},
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
