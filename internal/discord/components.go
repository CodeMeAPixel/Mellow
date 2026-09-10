package discord

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/CodeMeAPixel/Mellow/internal/ai"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

type wordGame struct {
	userID     int64
	kind       string
	difficulty string
	answer     string
	createdAt  time.Time
}

const gameTTL = time.Hour

func (b *Bot) putGame(id string, g *wordGame) {
	b.gamesMu.Lock()
	b.games[id] = g
	b.gamesMu.Unlock()
}

func (b *Bot) getGame(id string) (*wordGame, bool) {
	b.gamesMu.Lock()
	defer b.gamesMu.Unlock()
	g, ok := b.games[id]
	if ok && time.Since(g.createdAt) > gameTTL {
		delete(b.games, id)
		return nil, false
	}
	return g, ok
}

func (b *Bot) dropGame(id string) {
	b.gamesMu.Lock()
	delete(b.games, id)
	b.gamesMu.Unlock()
}

func (b *Bot) SweepGames(ctx context.Context) {
	t := time.NewTicker(15 * time.Minute)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			b.gamesMu.Lock()
			for id, g := range b.games {
				if time.Since(g.createdAt) > gameTTL {
					delete(b.games, id)
				}
			}
			b.gamesMu.Unlock()
		}
	}
}

func (b *Bot) onComponent(e *events.ComponentInteractionCreate) {
	id := e.ButtonInteractionData().CustomID()
	parts := strings.Split(id, ":")
	if len(parts) < 2 || parts[0] != "wg" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	userID := int64(e.User().ID)

	switch parts[1] {
	case "ans":
		gameID := strings.Join(parts[2:], ":")
		g, ok := b.getGame(gameID)
		if !ok {
			_ = e.CreateMessage(ephemeral("That game has expired."))
			return
		}
		if g.userID != userID {
			_ = e.CreateMessage(ephemeral("You can only answer your own game."))
			return
		}
		modal := discord.NewModalCreate("wg:sub:"+gameID, "Your answer").
			AddLabel("Type your answer", discord.NewParagraphTextInput("answer").WithRequired(true).WithMaxLength(500))
		_ = e.Modal(modal)
	case "new":
		if len(parts) < 4 {
			return
		}
		_ = e.DeferCreateMessage(false)
		b.startWordGame(ctx, wgReplyFollowup(e), userID, parts[2], parts[3])
	}
}

func (b *Bot) onModal(e *events.ModalSubmitInteractionCreate) {
	id := e.Data.CustomID
	parts := strings.Split(id, ":")
	if len(parts) < 3 || parts[0] != "wg" || parts[1] != "sub" {
		return
	}
	gameID := strings.Join(parts[2:], ":")
	g, ok := b.getGame(gameID)
	if !ok {
		_ = e.CreateMessage(ephemeral("That game has expired."))
		return
	}
	answer := e.Data.Text("answer")
	_ = e.DeferCreateMessage(false)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	correct := ai.FuzzyMatch(g.answer, answer)
	feedback, err := b.ai.EvaluateWordGame(ctx, g.kind, g.answer, answer)
	if err != nil || feedback == "" {
		if correct {
			feedback = "Correct! Nicely done."
		} else {
			feedback = "Not quite, but a good try. Have another go."
		}
	}
	b.dropGame(gameID)

	title := "Keep thinking"
	color := colorWarning
	if correct {
		title = "Great job"
		color = colorSuccess
	}
	emb := baseEmbed().WithColor(color).WithTitle(title).WithDescription(feedback)
	again := discord.NewPrimaryButton("Play again", "wg:new:"+g.kind+":"+g.difficulty)
	_, _ = e.Client().Rest.CreateFollowupMessage(e.ApplicationID(), e.Token(), discord.MessageCreate{
		Embeds:     []discord.Embed{emb},
		Components: []discord.LayoutComponent{discord.NewActionRow(again)},
	})
	_ = b.store.RecordCopingToolUsage(ctx, int64(e.User().ID), "wordgame")
}

type followupFunc func(discord.MessageCreate)

func wgReplyFollowup(e *events.ComponentInteractionCreate) followupFunc {
	return func(m discord.MessageCreate) {
		_, _ = e.Client().Rest.CreateFollowupMessage(e.ApplicationID(), e.Token(), m)
	}
}

func (b *Bot) startWordGame(ctx context.Context, send followupFunc, userID int64, kind, difficulty string) {
	text, answer, err := b.ai.GenerateWordGame(ctx, kind, difficulty)
	if err != nil || text == "" {
		send(discord.MessageCreate{Content: "I could not start a word game right now. Please try again later."})
		return
	}
	gameID := strconv.FormatInt(time.Now().UnixNano(), 36) + "-" + strconv.FormatInt(userID%100000, 36)
	b.putGame(gameID, &wordGame{userID: userID, kind: kind, difficulty: difficulty, answer: answer, createdAt: time.Now()})

	emb := infoEmbed(wordGameTitle(kind), text).
		AddField("Type", strings.ToUpper(kind), true).
		AddField("Difficulty", strings.ToUpper(difficulty), true)
	btn := discord.NewPrimaryButton("Submit your answer", "wg:ans:"+gameID)
	send(discord.MessageCreate{
		Embeds:     []discord.Embed{emb},
		Components: []discord.LayoutComponent{discord.NewActionRow(btn)},
	})
	_ = b.store.RecordCopingToolUsage(ctx, userID, "wordgame")
}

func wordGameTitle(kind string) string {
	switch kind {
	case "rhyme":
		return "Rhyme Time"
	case "puzzle":
		return "Word Puzzle"
	case "positive":
		return "Positive Words"
	default:
		return "Word Association"
	}
}

func ephemeral(s string) discord.MessageCreate {
	return discord.MessageCreate{Content: s, Flags: discord.MessageFlagEphemeral}
}
