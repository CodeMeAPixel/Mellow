package ai

import (
	"context"
	"net/url"
	"regexp"
	"strings"
)

type Meme struct {
	Title    string
	TopText  string
	BotText  string
	ImageURL string
}

var memegenTemplates = map[string]string{
	"drake":          "drake",
	"distracted":     "distracted",
	"expandingbrain": "expandingbrain",
	"twobuttons":     "choicememe",
	"changemymind":   "changemymind",
	"thisisfine":     "fine",
	"spiderman":      "spiderman",
}

func (c *Client) GenerateMeme(ctx context.Context, template, topic string) (Meme, error) {
	tmpl := strings.ToLower(strings.TrimSpace(template))
	if _, ok := memegenTemplates[tmpl]; !ok {
		tmpl = "drake"
	}
	user := "Create a wholesome, all-audiences meme.\nTemplate: " + tmpl + "\n"
	if topic != "" {
		user += "Topic: " + topic + "\n"
	}
	user += "Respond with exactly three lines:\nTitle: ...\nTop: ...\nBottom: ...\nKeep the text short and punchy. Never mock mental health."

	out, err := c.oneShot(ctx, "You are Mellow, a mental health companion with a kind sense of humour.", user)
	if err != nil {
		return Meme{}, err
	}
	m := parseMeme(out)
	m.ImageURL = memegenURL(memegenTemplates[tmpl], m.TopText, m.BotText)
	return m, nil
}

var (
	reTitle = regexp.MustCompile(`(?i)title:\s*(.+)`)
	reTop   = regexp.MustCompile(`(?i)top:\s*(.+)`)
	reBot   = regexp.MustCompile(`(?i)bottom:\s*(.+)`)
)

func parseMeme(s string) Meme {
	var m Meme
	if x := reTitle.FindStringSubmatch(s); x != nil {
		m.Title = strings.TrimSpace(x[1])
	}
	if x := reTop.FindStringSubmatch(s); x != nil {
		m.TopText = strings.TrimSpace(x[1])
	}
	if x := reBot.FindStringSubmatch(s); x != nil {
		m.BotText = strings.TrimSpace(x[1])
	}
	if m.TopText == "" && m.BotText == "" {
		lines := strings.Split(strings.TrimSpace(s), "\n")
		if len(lines) >= 2 {
			m.TopText = strings.TrimSpace(lines[0])
			m.BotText = strings.TrimSpace(lines[1])
		}
	}
	return m
}

func memegenURL(template, top, bot string) string {
	slug := func(s string) string {
		if s == "" {
			return "_"
		}
		s = strings.ReplaceAll(s, "_", "__")
		s = strings.ReplaceAll(s, " ", "_")
		return url.PathEscape(s)
	}
	return "https://api.memegen.link/images/" + template + "/" + slug(top) + "/" + slug(bot) + ".png"
}
