package pets

import (
	"math/rand"

	"guacagamblebot/internal/model"
)

type InteractionChoice struct {
	ID         string
	Label      string
	Emoji      string
	BondReward int
	XPReward   int
	ItemReward string // optional, rarely
}

type InteractionDef struct {
	ID          string
	Triggers    []string // "hunt", "battle", "feed", "expedition", "idle"
	Chance      float64  // base trigger probability
	CooldownM   int      // cooldown in minutes (default 180 = 3h)
	GetIntro    func(*model.UserPet) string
	Choices     []InteractionChoice
	Personality string // "" means any, or specific trait
}

type InteractionResult struct {
	ID         string
	Intro      string
	Choices    []InteractionChoice
}

type ChoiceResult struct {
	BondReward int
	XPReward   int
	ItemReward string
	Detail     string
}

var interactionPool = []InteractionDef{
	{
		ID: "play_time", Triggers: []string{"feed", "idle"}, Chance: 0.15, CooldownM: 180,
		GetIntro: func(p *model.UserPet) string {
			return personalityIntro(p.Personality, map[string]string{
				"playful":     "**" + p.Nickname + "** drops a stick at your feet and wags excitedly, eyes sparkling with anticipation!",
				"grumpy":      "**" + p.Nickname + "** sighs dramatically and nudges a toy toward you with a reluctant paw.",
				"timid":       "**" + p.Nickname + "** places a small leaf at your feet, then hides behind a bush, peeking out nervously.",
				"curious":     "**" + p.Nickname + "** tilts its head and presents a strange object it found, waiting for your reaction.",
				"sleepy":      "**" + p.Nickname + "** half-opens one eye, weakly wagging a toy while clearly half-asleep.",
				"gentle":      "**" + p.Nickname + "** gently places its favorite toy in your lap and looks at you with trusting eyes.",
				"fierce":      "**" + p.Nickname + "** drops a chewed-up stick and growls playfully, challenging you to a game!",
				"mischievous": "**" + p.Nickname + "** darts around you in circles, then pretends to hide something behind its back.",
				"loyal":       "**" + p.Nickname + "** brings you its most prized possession — a shiny rock it's been saving.",
				"brave":       "**" + p.Nickname + "** stands tall and presents a stick like a warrior offering a sacred sword!",
			}, "**"+p.Nickname+"** drops a stick at your feet and looks up at you expectantly!")
		},
		Choices: []InteractionChoice{
			{ID: "fetch", Label: "Play fetch", Emoji: "🎾", BondReward: 3, XPReward: 10},
			{ID: "tug", Label: "Tug of war", Emoji: "🪢", BondReward: 4, XPReward: 15, ItemReward: "pebble"},
			{ID: "ignore", Label: "Pet it gently", Emoji: "🤲", BondReward: 2},
		},
	},
	{
		ID: "snack_time", Triggers: []string{"feed", "hunt"}, Chance: 0.12, CooldownM: 180,
		GetIntro: func(p *model.UserPet) string {
			return personalityIntro(p.Personality, map[string]string{
				"playful":     "**" + p.Nickname + "** tsks and paws at its empty food bowl, giving you the most dramatic hungry eyes you've ever seen!",
				"grumpy":      "**" + p.Nickname + "** glares at its food bowl, then at you, clearly unimpressed with today's menu.",
				"curious":     "**" + p.Nickname + "** sniffs the air and follows an invisible scent trail, eventually ending up at its food bowl.",
				"timid":       "**" + p.Nickname + "** approaches its bowl hesitantly, looking back at you for reassurance before eating.",
				"sleepy":      "**" + p.Nickname + "** sleepwalks to the food bowl, eats while napping, then wanders back.",
				"gentle":      "**" + p.Nickname + "** waits politely by its bowl, never rushing, always grateful.",
				"mischievous": "**" + p.Nickname + "** tips its bowl over, then acts surprised, as if it magically emptied itself.",
				"loyal":       "**" + p.Nickname + "** saves half its food and pushes the bowl toward you, offering to share.",
				"fierce":      "**" + p.Nickname + "** attacks its food bowl with relentless fury, winning the battle byte by byte.",
				"brave":       "**" + p.Nickname + "** bravely faces the empty bowl, determined to wait for its next meal without complaint.",
			}, "**"+p.Nickname+"** looks at you with big, hopeful eyes near the food bowl.")
		},
		Choices: []InteractionChoice{
			{ID: "feed_treat", Label: "Give a treat", Emoji: "🍖", BondReward: 3, XPReward: 5},
			{ID: "share_meal", Label: "Share your meal", Emoji: "🍲", BondReward: 5, XPReward: 10},
			{ID: "cook", Label: "Cook something special", Emoji: "🍳", BondReward: 4, XPReward: 20, ItemReward: "tomato"},
		},
	},
	{
		ID: "explore_together", Triggers: []string{"expedition", "idle"}, Chance: 0.12, CooldownM: 240,
		GetIntro: func(p *model.UserPet) string {
			return personalityIntro(p.Personality, map[string]string{
				"curious":     "**" + p.Nickname + "** stares intently at a distant hill, then looks at you, then back at the hill. The message is clear.",
				"playful":     "**" + p.Nickname + "** dashes toward the horizon, stops, looks back, dashes further — a clear invitation!",
				"timid":       "**" + p.Nickname + "** peeks around a corner, ears perked, then hides behind your legs.",
				"brave":       "**" + p.Nickname + "** puffs out its chest and gestures toward the unknown with unmatched courage.",
				"grumpy":      "**" + p.Nickname + "** mutters something under its breath but starts walking, checking you're following.",
				"sleepy":      "**" + p.Nickname + "** yawns, stretches, and points a paw at a nearby cave before yawning again.",
				"loyal":       "**" + p.Nickname + "** stays by your side but keeps glancing at a path it wants to explore together.",
				"fierce":      "**" + p.Nickname + "** snarls at the wilderness, ready to conquer whatever lies beyond.",
				"gentle":      "**" + p.Nickname + "** gently nudges you toward a flower-filled meadow it discovered.",
				"mischievous": "**" + p.Nickname + "** pretends to run away, then circles back behind you and scares you playfully.",
			}, "**"+p.Nickname+"** looks at you with an adventurous glint in its eyes.")
		},
		Choices: []InteractionChoice{
			{ID: "explore", Label: "Explore together", Emoji: "🧭", BondReward: 4, XPReward: 25},
			{ID: "follow", Label: "Let it lead the way", Emoji: "🐾", BondReward: 3, XPReward: 15, ItemReward: "coal"},
			{ID: "rest", Label: "Take a break", Emoji: "😴", BondReward: 2},
		},
	},
	{
		ID: "grooming", Triggers: []string{"feed", "idle"}, Chance: 0.10, CooldownM: 240,
		GetIntro: func(p *model.UserPet) string {
			return personalityIntro(p.Personality, map[string]string{
				"gentle":      "**" + p.Nickname + "** sits down and looks at you, then at its fur, then back at you with hopeful eyes.",
				"grumpy":      "**" + p.Nickname + "** has leaves stuck in its fur. It pretends not to care, but keeps glancing at them.",
				"playful":     "**" + p.Nickname + "** rolls in the mud, then proudly presents its filthy self to you.",
				"timid":       "**" + p.Nickname + "** approaches with a tangled coat, looking embarrassed.",
				"curious":     "**" + p.Nickname + "** examines its own reflection in a puddle and tries to fix its messy fur.",
				"sleepy":      "**" + p.Nickname + "** has bedhead and doesn't seem to notice. Or care.",
				"brave":       "**" + p.Nickname + "** returns from battle covered in scars and dirt, standing proudly — but could use a clean.",
				"fierce":      "**" + p.Nickname + "** growls at a brush, then reluctantly allows a single stroke.",
				"loyal":       "**" + p.Nickname + "** brings you a brush and sits perfectly still, trusting you completely.",
				"mischievous": "**" + p.Nickname + "** shakes vigorously right next to you, spraying water everywhere, then grins.",
			}, "**"+p.Nickname+"** seems to be asking for some grooming attention.")
		},
		Choices: []InteractionChoice{
			{ID: "brush", Label: "Brush its fur", Emoji: "🪥", BondReward: 4, XPReward: 5},
			{ID: "bath", Label: "Give a bath", Emoji: "🛁", BondReward: 5, XPReward: 10, ItemReward: "sardine"},
			{ID: "massage", Label: "Give a massage", Emoji: "💆", BondReward: 3},
		},
	},
	{
		ID: "training", Triggers: []string{"battle", "hunt"}, Chance: 0.10, CooldownM: 240,
		GetIntro: func(p *model.UserPet) string {
			return personalityIntro(p.Personality, map[string]string{
				"brave":       "**" + p.Nickname + "** strikes a fighting pose and challenges you to a sparring match!",
				"fierce":      "**" + p.Nickname + "** is panting heavily, but its eyes burn with the desire to keep training.",
				"playful":     "**" + p.Nickname + "** turns training into a game, bounce-bounce-bouncing around you.",
				"grumpy":      "**" + p.Nickname + "** grumbles but gets into position. Fine. ONE more round.",
				"timid":       "**" + p.Nickname + "** is shaking but stands its ground, determined to improve.",
				"curious":     "**" + p.Nickname + "** watches your movements carefully, trying to learn new techniques.",
				"loyal":       "**" + p.Nickname + "** mimics your every move perfectly, wanting to make you proud.",
				"gentle":      "**" + p.Nickname + "** spars gently, pulling its punches to avoid hurting you.",
				"sleepy":      "**" + p.Nickname + "** falls asleep mid-training session, snoring cutely.",
				"mischievous": "**" + p.Nickname + "** keeps trying to trip you during training, giggling each time.",
			}, "**"+p.Nickname+"** wants to train and is looking at you expectantly.")
		},
		Choices: []InteractionChoice{
			{ID: "spar", Label: "Spar with it", Emoji: "⚔️", BondReward: 4, XPReward: 30},
			{ID: "teach", Label: "Teach a new move", Emoji: "📖", BondReward: 3, XPReward: 20},
			{ID: "praise", Label: "Praise its effort", Emoji: "👏", BondReward: 5},
		},
	},
	{
		ID: "rescue", Triggers: []string{"battle", "hunt"}, Chance: 0.05, CooldownM: 480,
		GetIntro: func(p *model.UserPet) string {
			return personalityIntro(p.Personality, map[string]string{
				"brave":       "**" + p.Nickname + "** runs to you, then turns and growls at something in the bushes. Something is out there.",
				"fierce":      "**" + p.Nickname + "** bares its teeth at a shady figure lurking nearby. It won't back down.",
				"timid":       "**" + p.Nickname + "** tremblingly stands between you and a strange creature, protecting you despite its fear.",
				"loyal":       "**" + p.Nickname + "** blocks your path, refusing to let you proceed. Danger ahead.",
				"curious":     "**" + p.Nickname + "** discovered a trapped animal and is trying to free it, looking at you for help.",
				"gentle":      "**" + p.Nickname + "** found a wounded little creature and gently nudges it toward you for help.",
				"playful":     "**" + p.Nickname + "** thinks the dangerous situation is a game. You need to handle this carefully.",
				"grumpy":      "**" + p.Nickname + "** sighs and deals with the threat efficiently, then gives you a 'you owe me' look.",
				"sleepy":      "**" + p.Nickname + "** yawns, lazily scares off the threat with a single look, then goes back to napping.",
				"mischievous": "**" + p.Nickname + "** pranks the threat so thoroughly it runs away confused and embarrassed.",
			}, "**"+p.Nickname+"** senses danger and takes a protective stance.")
		},
		Choices: []InteractionChoice{
			{ID: "stand_together", Label: "Stand together", Emoji: "🤝", BondReward: 8, XPReward: 40, ItemReward: "rough_diamond"},
			{ID: "investigate", Label: "Investigate together", Emoji: "🔦", BondReward: 6, XPReward: 30},
			{ID: "retreat", Label: "Retreat to safety", Emoji: "🏃", BondReward: 4, XPReward: 10},
		},
	},
}

// MaybeTriggerInteraction checks cooldown and probability and returns an interaction if triggered.
func MaybeTriggerInteraction(pet *model.UserPet, trigger string) *InteractionResult {
	// Filter by trigger
	var candidates []InteractionDef
	for _, in := range interactionPool {
		for _, t := range in.Triggers {
			if t == trigger && (in.Personality == "" || in.Personality == pet.Personality) {
				candidates = append(candidates, in)
				break
			}
		}
	}
	if len(candidates) == 0 {
		return nil
	}
	// Pick random candidate
	chosen := candidates[rand.Intn(len(candidates))]
	if rand.Float64() > chosen.Chance {
		return nil
	}
	return &InteractionResult{
		ID:      chosen.ID,
		Intro:   chosen.GetIntro(pet),
		Choices: chosen.Choices,
	}
}

// ResolveInteraction returns the result for a given choice.
func ResolveInteraction(pet *model.UserPet, choiceID string) *ChoiceResult {
	// Find the choice across all interactions
	for _, in := range interactionPool {
		for _, c := range in.Choices {
			if c.ID == choiceID {
				detail := choiceDetail(pet.Personality, choiceID, c)
				return &ChoiceResult{
					BondReward: c.BondReward,
					XPReward:   c.XPReward,
					ItemReward: c.ItemReward,
					Detail:     detail,
				}
			}
		}
	}
	return nil
}

func choiceDetail(personality, choiceID string, choice InteractionChoice) string {
	baseStr := "💕 " + choiceDetailText(choiceID, choice)
	return personalityIntro(personality, map[string]string{
		"playful":     "🎉 " + baseStr,
		"grumpy":      "😤 " + baseStr + " (But secretly, it loved every second.)",
		"curious":     "🔍 " + baseStr + " It files away every detail for later.",
		"timid":       "😰 " + baseStr + " It's getting braver thanks to you!",
		"gentle":      "🤲 " + baseStr + " The gentle moment warms your heart.",
		"sleepy":      "💤 " + baseStr + " It nods off contentedly afterward.",
		"fierce":      "🔥 " + baseStr + " Everything is a competition, and it won.",
		"brave":       "⚔️ " + baseStr + " Another adventure conquered together!",
		"mischievous": "😈 " + baseStr + " It tried to cheat. You pretend not to notice.",
		"loyal":       "❤️ " + baseStr + " This moment means the world to your loyal friend.",
	}, baseStr)
}

func choiceDetailText(choiceID string, _ InteractionChoice) string {
	switch choiceID {
	case "fetch":
		return "You throw the stick. **" + "Your pet" + "** zooms after it, returning with tail wagging furiously!"
	case "tug":
		return "You engage in an epic tug of war. **" + "Your pet" + "** pulls with all its might, and you let it win."
	case "ignore":
		return "You gently pet **" + "Your pet" + "**, and it leans into your hand, perfectly content."
	case "feed_treat":
		return "You give **" + "Your pet" + "** a special treat. It savors every bite, looking at you with pure gratitude."
	case "share_meal":
		return "You share your meal with **" + "Your pet" + "**. It wags happily — sharing is caring!"
	case "cook":
		return "You whip up something delicious. **" + "Your pet" + "** watches every step, drooling slightly."
	case "explore":
		return "You explore the unknown with **" + "Your pet" + "** by your side. Nothing can stop this team!"
	case "follow":
		return "You let **" + "Your pet" + "** take the lead. It puffs with pride and guides you somewhere new."
	case "brush":
		return "You brush **" + "Your pet" + "**'s coat until it shines. It practically purrs with happiness."
	case "bath":
		return "Bath time! **" + "Your pet" + "** splashes everywhere but emerges clean and fluffy."
	case "massage":
		return "You massage **" + "Your pet" + "**'s tired muscles. It melts into a puddle of relaxation."
	case "spar":
		return "You spar with **" + "Your pet" + "**. It's getting faster, stronger — your training is paying off!"
	case "teach":
		return "You teach **" + "Your pet" + "** a new trick. It concentrates hard and eventually nails it!"
	case "praise":
		return "You heap praise on **" + "Your pet" + "**. It puffs up with pride, happier than any treat could make it."
	case "stand_together":
		return "You stand shoulder to shoulder with **" + "Your pet" + "** against the unknown. United, you're unstoppable."
	case "investigate":
		return "You and **" + "Your pet" + "** carefully investigate. It turns out to be a false alarm, but you're glad you checked."
	case "retreat":
		return "You wisely retreat with **" + "Your pet" + "**. Safety first — there will be other battles."
	default:
		return "You spend quality time with **" + "Your pet" + "**. The bond grows stronger."
	}
}

func personalityIntro(personality string, texts map[string]string, fallback string) string {
	if t, ok := texts[personality]; ok {
		return t
	}
	return fallback
}
