package helplines

import (
	"sort"
	"strings"
)

const footer = "You can also reach out to someone you trust, or go to your nearest emergency room. " +
	"I'm an AI and not a substitute for a person or professional, but you deserve real support and it is available."

const international = "- Anywhere: find a local helpline at **https://findahelpline.com** or **https://www.iasp.info/resources/Crisis_Centres/**\n\n"

const DefaultBlock = "**If you are in immediate danger, call your local emergency number now.**\n\n" +
	"- US: call or text **988** (Suicide & Crisis Lifeline), available 24/7\n" +
	"- US: text **HOME** to **741741** (Crisis Text Line)\n" +
	"- Anywhere: find a local helpline at **https://findahelpline.com** or **https://www.iasp.info/resources/Crisis_Centres/**\n\n" +
	footer

type Country struct {
	Code      string   `json:"code"`
	Name      string   `json:"name"`
	Emergency string   `json:"-"`
	Lines     []string `json:"-"`
}

var countries = []Country{
	{"AU", "Australia", "000", []string{
		"call **13 11 14** (Lifeline), available 24/7",
		"text **0477 13 11 14** (Lifeline)",
		"call **1300 659 467** (Suicide Call Back Service)",
	}},
	{"BR", "Brazil", "192 (SAMU) or 190", []string{
		"call **188** (CVV), available 24/7",
	}},
	{"CA", "Canada", "911", []string{
		"call or text **988** (Suicide Crisis Helpline), available 24/7",
		"young people: call **1-800-668-6868** or text **686868** (Kids Help Phone)",
	}},
	{"FR", "France", "112 (or 15)", []string{
		"call **3114** (national suicide prevention line), available 24/7",
	}},
	{"DE", "Germany", "112", []string{
		"call **0800 111 0 111** or **0800 111 0 222** (Telefonseelsorge), free, available 24/7",
	}},
	{"IN", "India", "112", []string{
		"call **14416** or **1-800-891-4416** (Tele-MANAS)",
	}},
	{"IE", "Ireland", "112 or 999", []string{
		"call **116 123** (Samaritans), free, available 24/7",
		"text **HELLO** to **50808**",
	}},
	{"NL", "Netherlands", "112", []string{
		"call **113** or **0800-0113** (113 Suicide Prevention), available 24/7",
	}},
	{"NZ", "New Zealand", "111", []string{
		"call or text **1737** (Need to talk?), free, available 24/7",
		"call **0800 543 354** (Lifeline)",
	}},
	{"ES", "Spain", "112", []string{
		"call **024** (suicide prevention line), available 24/7",
	}},
	{"GB", "United Kingdom", "999 (or 112)", []string{
		"call **116 123** (Samaritans), free, available 24/7",
		"text **SHOUT** to **85258** (Shout Crisis Text Line)",
	}},
	{"US", "United States", "911", []string{
		"call or text **988** (Suicide & Crisis Lifeline), available 24/7",
		"text **HOME** to **741741** (Crisis Text Line)",
	}},
}

func Countries() []Country {
	out := make([]Country, len(countries))
	copy(out, countries)
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func Lookup(code string) (Country, bool) {
	code = strings.ToUpper(strings.TrimSpace(code))
	for _, c := range countries {
		if c.Code == code {
			return c, true
		}
	}
	return Country{}, false
}

func Block(code string) string {
	c, ok := Lookup(code)
	if !ok {
		return DefaultBlock
	}
	var b strings.Builder
	b.WriteString("**If you are in immediate danger, call " + c.Emergency + " now.**\n\n")
	for _, l := range c.Lines {
		b.WriteString("- " + c.Name + ": " + l + "\n")
	}
	b.WriteString(international)
	b.WriteString(footer)
	return b.String()
}
