package discord

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/disgoorg/disgo/discord"
)

type guidedStep struct {
	text string
	hold time.Duration
}

var guidedExercises = []struct{ value, label string }{
	{"breathing", "Box breathing (about 1 minute)"},
	{"grounding", "5-4-3-2-1 grounding (about 2 minutes)"},
	{"winddown", "Wind-down for the evening (about 3 minutes)"},
}

var guidedActive sync.Map

const guidedFooter = "\n\nIf things feel like too much, `/crisis resources` is always there."

func guidedTitle(exercise string) string {
	switch exercise {
	case "breathing":
		return "Box breathing"
	case "grounding":
		return "5-4-3-2-1 grounding"
	case "winddown":
		return "Evening wind-down"
	}
	return ""
}

func guidedSteps(exercise string) []guidedStep {
	switch exercise {
	case "breathing":
		steps := []guidedStep{{"Get comfortable and let your shoulders drop. We will breathe together in a steady 4-4-4-4 pattern. It starts in a moment.", 6 * time.Second}}
		const rounds = 4
		for r := 1; r <= rounds; r++ {
			tag := fmt.Sprintf("\n\nRound %d of %d", r, rounds)
			steps = append(steps,
				guidedStep{"**Breathe in** slowly through your nose, for 4." + tag, 4 * time.Second},
				guidedStep{"**Hold** gently, for 4." + tag, 4 * time.Second},
				guidedStep{"**Breathe out** slowly through your mouth, for 4." + tag, 4 * time.Second},
				guidedStep{"**Rest**, for 4." + tag, 4 * time.Second},
			)
		}
		return append(steps, guidedStep{"Well done. Notice how your body feels now, and take as long as you need." + guidedFooter, 0})

	case "grounding":
		const hold = 25 * time.Second
		return []guidedStep{
			{"This helps bring you back to the present. Take your time with each step.", 8 * time.Second},
			{"Look around and name **5** things you can see.", hold},
			{"Notice **4** things you can feel or touch, like your feet on the floor or the fabric of your clothes.", hold},
			{"Listen for **3** things you can hear, near or far.", hold},
			{"Find **2** things you can smell, or two smells you like to remember.", hold},
			{"Name **1** thing you can taste, or one thing you are grateful for right now.", hold},
			{"Take one slow breath. You are here, and you are okay in this moment." + guidedFooter, 0},
		}

	case "winddown":
		const hold = 30 * time.Second
		return []guidedStep{
			{"Let us slow the day down together. There is nothing to do except follow along.", 8 * time.Second},
			{"Unclench your jaw and let your shoulders drop away from your ears.", hold},
			{"Take three slow breaths, making each exhale a little longer than the inhale.", hold},
			{"Think of one thing from today that went okay, however small.", hold},
			{"Choose one thing that can wait until tomorrow, and let yourself put it down for tonight.", hold},
			{"Notice the bed or chair holding your weight. You do not have to hold yourself up right now.", hold},
			{"You did enough today. Rest well." + guidedFooter, 0},
		}
	}
	return nil
}

func guidedCommand() *Command {
	choices := make([]discord.ApplicationCommandOptionChoiceString, 0, len(guidedExercises))
	for _, e := range guidedExercises {
		choices = append(choices, discord.ApplicationCommandOptionChoiceString{Name: e.label, Value: e.value})
	}
	return &Command{
		Name: "guided", Description: "A guided breathing, grounding, or wind-down exercise, live in this chat.",
		Category: "Coping", Cooldown: 10 * time.Second,
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionString{Name: "exercise", Description: "Which exercise", Required: true, Choices: choices},
		},
		Run: runGuided,
	}
}

func guidedEmbed(title, text string) discord.Embed { return infoEmbed(title, text) }

func runGuided(ctx context.Context, c *Ctx) error {
	exercise := c.String("exercise")
	steps := guidedSteps(exercise)
	if steps == nil {
		return c.ReplyEphemeral("Unknown exercise.")
	}
	if _, running := guidedActive.LoadOrStore(c.UserID, true); running {
		return c.ReplyEphemeral("You already have a guided exercise running. Let it finish, then start another.")
	}

	_ = c.Defer(true)
	client, appID, token := c.Event.Client(), c.Event.ApplicationID(), c.Event.Token()
	title := guidedTitle(exercise)

	msg, err := client.Rest.CreateFollowupMessage(appID, token, discord.MessageCreate{
		Embeds: []discord.Embed{guidedEmbed(title, steps[0].text)},
		Flags:  discord.MessageFlagEphemeral,
	})
	if err != nil {
		guidedActive.Delete(c.UserID)
		return err
	}
	_ = c.Store.RecordCopingToolUsageIn(ctx, c.UserID, "guided_"+exercise, c.Bot.activityGuild(c.GuildID))

	userID := c.UserID
	go func() {
		defer guidedActive.Delete(userID)
		timeout := time.NewTimer(10 * time.Minute)
		defer timeout.Stop()
		for i := 1; i < len(steps); i++ {
			select {
			case <-time.After(steps[i-1].hold):
			case <-timeout.C:
				return
			}
			embeds := []discord.Embed{guidedEmbed(title, steps[i].text)}
			if _, err := client.Rest.UpdateFollowupMessage(appID, token, msg.ID, discord.MessageUpdate{Embeds: &embeds}); err != nil {
				return
			}
		}
	}()
	return nil
}
