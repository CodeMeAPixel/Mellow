package discord

import (
	"context"
	"time"

	"github.com/CodeMeAPixel/Mellow/internal/ai"
	"github.com/CodeMeAPixel/Mellow/internal/db"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/omit"
)

type Command struct {
	Name          string
	Description   string
	Category      string
	Cooldown      time.Duration
	GuildOnly     bool
	Private       bool
	OwnerOnly     bool
	RequiredRoles []string
	RequiredPerms []discord.Permissions
	Options       []discord.ApplicationCommandOption
	Run           func(ctx context.Context, c *Ctx) error
}

func (c Command) create() discord.ApplicationCommandCreate {
	sc := discord.SlashCommandCreate{
		Name:        c.Name,
		Description: c.Description,
		Options:     c.Options,
	}
	if len(c.RequiredPerms) > 0 {
		var p discord.Permissions
		for _, rp := range c.RequiredPerms {
			p = p.Add(rp)
		}
		sc.DefaultMemberPermissions = omit.NewPtr(p)
	}
	return sc
}

type Ctx struct {
	Event    *events.ApplicationCommandInteractionCreate
	Data     discord.SlashCommandInteractionData
	Store    *db.Store
	AI       *ai.Client
	Bot      *Bot
	UserID   int64
	GuildID  *int64
	deferred bool
}

func (c *Ctx) Sub() string {
	if c.Data.SubCommandName != nil {
		return *c.Data.SubCommandName
	}
	return ""
}

func (c *Ctx) String(name string) string   { v, _ := c.Data.OptString(name); return v }
func (c *Ctx) Int(name string) (int, bool) { return c.Data.OptInt(name) }
func (c *Ctx) Bool(name string) bool       { v, _ := c.Data.OptBool(name); return v }

func (c *Ctx) Defer(ephemeral bool) error {
	c.deferred = true
	return c.Event.DeferCreateMessage(ephemeral)
}

func (c *Ctx) Reply(embeds ...discord.Embed) error {
	if c.deferred {
		return c.followup(discord.MessageCreate{Embeds: embeds})
	}
	return c.Event.CreateMessage(discord.MessageCreate{Embeds: embeds})
}

func (c *Ctx) ReplyText(s string) error {
	if c.deferred {
		return c.followup(discord.MessageCreate{Content: s})
	}
	return c.Event.CreateMessage(discord.MessageCreate{Content: s})
}

func (c *Ctx) ReplyEphemeral(s string) error {
	if c.deferred {
		return c.followup(discord.MessageCreate{Content: s, Flags: discord.MessageFlagEphemeral})
	}
	return c.Event.CreateMessage(discord.MessageCreate{Content: s, Flags: discord.MessageFlagEphemeral})
}

func (c *Ctx) followup(m discord.MessageCreate) error {
	_, err := c.Event.Client().Rest.CreateFollowupMessage(
		c.Event.ApplicationID(), c.Event.Token(), m,
	)
	return err
}
