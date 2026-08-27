package components

// The bot's embed palette. Every embed border colour in the bot resolves to one
// of these constants so that the same kind of message looks the same wherever
// it is produced. Before this existed the codebase carried 39 distinct hex
// literals across 500 call sites, including pure case variants of one another
// (0x9b59b6 / 0x9B59B6) and unintentional duplicates of a single meaning
// (0x00FF00 alongside 0x2ECC71 for "this succeeded").
//
// The values are the Flat UI palette, which is what the bot had already
// converged on by hand. They are untyped constants so they drop straight into
// discordgo's MessageEmbed.Color int field and into Embed's colour argument.
//
// Pick by *meaning*, not by appearance: a caller that wants "this went well"
// asks for ColorSuccess, never 0x2ECC71. That indirection is the whole point —
// retheming the bot then means editing this file rather than 500 call sites.
const (
	// ColorSuccess marks an action that completed in the player's favour: a
	// win, a claimed reward, a completed expedition, a heal.
	ColorSuccess = 0x2ECC71
	// ColorDanger marks loss, damage and failure, including user-facing errors.
	ColorDanger = 0xE74C3C
	// ColorWarning marks a risky or degrading state that is not yet a failure:
	// caution prompts, damage-over-time ticks, decay.
	ColorWarning = 0xE67E22
	// ColorInfo is the neutral default for informational embeds that carry no
	// win/loss verdict.
	ColorInfo = 0x3498DB
	// ColorArcane marks the game's mystical surfaces: quests, lore, the Veil,
	// stun effects and epic-tier drops.
	ColorArcane = 0x9B59B6
	// ColorReward marks payouts and accolades: achievements, jackpots,
	// leaderboard placements.
	ColorReward = 0xF1C40F
	// ColorMuted marks inert or dismissed states: a declined duel, a dodged
	// attack, common-tier loot.
	ColorMuted = 0x95A5A6
	// ColorIdle is the resting border for an embed with nothing to report yet,
	// such as a fight journal before the first blow lands.
	ColorIdle = 0x2C3E50
	// ColorBrand is the bot's own identity colour, reserved for surfaces that
	// speak as the bot rather than about the game: /help, onboarding, setup.
	ColorBrand = 0x5865F2
)

// Rarity tier colours. These alias the palette above rather than introducing
// new values, so a legendary drop and a jackpot payout read as the same kind of
// event. Kept as their own names because callers reason in tiers, and because
// the tier-to-colour mapping is a game-design decision that may change
// independently of what "success" looks like.
const (
	ColorCommon    = ColorMuted
	ColorRare      = ColorSuccess
	ColorEpic      = ColorArcane
	ColorLegendary = 0xFFD700
)
