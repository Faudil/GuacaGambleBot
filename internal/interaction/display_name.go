package interaction

import (
	"strconv"

	"github.com/bwmarrin/discordgo"
)

// DisplayName returns the guild display name of a user. The interaction member
// is used directly when it matches the requested user, otherwise the member is
// fetched from the guild. Falls back to a mention when the member cannot be
// resolved.
func DisplayName(s *discordgo.Session, guildID string, member *discordgo.Member, userID int64) string {
	if member != nil && member.User != nil && userID == ToInt64(member.User.ID) {
		if member.Nick != "" {
			return member.Nick
		}
		return member.User.Username
	}
	mem, err := s.GuildMember(guildID, strconv.FormatInt(userID, 10))
	if err != nil || mem == nil {
		return "<@" + strconv.FormatInt(userID, 10) + ">"
	}
	if mem.Nick != "" {
		return mem.Nick
	}
	if mem.User != nil {
		return mem.User.Username
	}
	return "<@" + strconv.FormatInt(userID, 10) + ">"
}
