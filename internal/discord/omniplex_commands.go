package discord

import (
	"strings"

	"github.com/CodeMeAPixel/Mellow/internal/omniplex"
	"github.com/disgoorg/disgo/discord"
)

func (b *Bot) OmniplexCommands() []omniplex.Command {
	out := make([]omniplex.Command, 0, len(b.commands))
	for _, c := range b.commands {
		if c.Private || c.Category == "Stub" {
			continue
		}
		out = append(out, omniplex.Command{
			Name:        c.Name,
			Description: c.Description,
			Category:    c.Category,
			Usage:       usageString(c),
		})
	}
	return out
}

func usageString(c *Command) string {
	var subs, args []string
	for _, o := range c.Options {
		switch v := o.(type) {
		case discord.ApplicationCommandOptionSubCommand:
			subs = append(subs, v.Name)
		case discord.ApplicationCommandOptionSubCommandGroup:
			subs = append(subs, v.Name)
		case discord.ApplicationCommandOptionString:
			args = append(args, wrap(v.Name, v.Required))
		case discord.ApplicationCommandOptionInt:
			args = append(args, wrap(v.Name, v.Required))
		case discord.ApplicationCommandOptionBool:
			args = append(args, wrap(v.Name, v.Required))
		case discord.ApplicationCommandOptionUser:
			args = append(args, wrap(v.Name, v.Required))
		case discord.ApplicationCommandOptionChannel:
			args = append(args, wrap(v.Name, v.Required))
		case discord.ApplicationCommandOptionRole:
			args = append(args, wrap(v.Name, v.Required))
		}
	}
	s := "/" + c.Name
	if len(subs) > 0 {
		s += " <" + strings.Join(subs, " | ") + ">"
	}
	if len(args) > 0 {
		s += " " + strings.Join(args, " ")
	}
	return s
}

func wrap(name string, required bool) string {
	if required {
		return "<" + name + ">"
	}
	return "[" + name + "]"
}
