package pets

import (
	"testing"

	"guacagamblebot/internal/components"
)

// Discord rejects the whole interaction response (400 Invalid Form Body) when a
// component emoji name holds more than one emoji. A pet type carrying e.g.
// "🐺🐺🐺" therefore made /pet and the sanctuary pickers fail silently for every
// player owning one, with nothing in the logs.
func TestPetTypeEmojisAreSingleEmoji(t *testing.T) {
	for name, pt := range PetTypes {
		if components.SafeEmoji(pt.Emoji) == nil {
			t.Errorf("pet type %q has an emoji Discord will reject: %q", name, pt.Emoji)
		}
	}
}

// SafeEmoji must accept ZWJ sequences, variation selectors and skin tones (all
// one emoji to Discord) while rejecting concatenations of several.
func TestSafeEmojiClassifies(t *testing.T) {
	bad := []string{"", "\U0001F43A\U0001F43A\U0001F43A", "\U0001F40D\u26A1", "\U0001F422\U0001F333"}
	for _, e := range bad {
		if components.SafeEmoji(e) != nil {
			t.Errorf("SafeEmoji(%q) should be rejected", e)
		}
	}
	good := []string{"\U0001F43A", "\U0001F43F\uFE0F", "\U0001F44D\U0001F3FD", "\U0001F469\u200D\U0001F680"}
	for _, e := range good {
		if components.SafeEmoji(e) == nil {
			t.Errorf("SafeEmoji(%q) should be accepted", e)
		}
	}
}
