package discord

import (
	"sync/atomic"
	"time"

	"github.com/disgoorg/disgo/discord"
)

const (
	colorPrimary = 0x7289DA
	colorInfo    = 0x00AFF4
	colorSuccess = 0x57F287
	colorWarning = 0xFEE75C
	colorError   = 0xED4245

	footerText = "Mellow - Your AI Mental Health Companion"
	authorName = "Pixelated (CodeMeAPixel)"
	authorURL  = "https://codemeapixel.dev"
)

type brandInfo struct {
	Website   string
	Docs      string
	Support   string
	Invite    string
	Source    string
	AvatarURL string
}

var brandState atomic.Pointer[brandInfo]

func setBrand(b brandInfo) { brandState.Store(&b) }

func brand() brandInfo {
	if b := brandState.Load(); b != nil {
		return *b
	}
	return brandInfo{}
}

func webBase() string     { return brand().Website }
func docsLink() string    { return brand().Docs }
func supportLink() string { return brand().Support }
func inviteLink() string  { return brand().Invite }
func sourceLink() string  { return brand().Source }

func baseEmbed() discord.Embed {
	return discord.NewEmbed().
		WithColor(colorPrimary).
		WithFooter(footerText, brand().AvatarURL).
		WithTimestamp(time.Now())
}

func infoEmbed(title, desc string) discord.Embed {
	return baseEmbed().WithColor(colorInfo).WithTitle(title).WithDescription(desc)
}

func successEmbed(title, desc string) discord.Embed {
	return baseEmbed().WithColor(colorSuccess).WithTitle(title).WithDescription(desc)
}

func errorEmbed(title, desc string) discord.Embed {
	return baseEmbed().WithColor(colorError).WithTitle(title).WithDescription(desc)
}
