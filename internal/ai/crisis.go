package ai

import (
	"context"
	"regexp"
	"strings"

	"github.com/CodeMeAPixel/Mellow/internal/helplines"
)

type CrisisResult struct {
	NeedsSupport bool
	Respond      bool
	Level        string
	Keywords     []string
	Summary      string
}

const CrisisResourceBlock = helplines.DefaultBlock

const criticalLead = "I'm really concerned about what you've shared, and I want you to be safe. " +
	"Please talk to someone who can help right now."

const criticalCrisisMessage = criticalLead + "\n\n" + CrisisResourceBlock

const mediumLead = "I hear how much pain you're in, and I'm glad you said something. " +
	"You don't have to carry this alone."

const mediumCrisisFallback = mediumLead + "\n\n" + CrisisResourceBlock

var selfHarmDirect = []string{
	"kill myself", "end my life", "want to die", "suicide", "suicidal", "self harm",
	"cut myself", "hurt myself", "end it all", "not worth living", "better off dead",
	"going to kill myself", "planning to die", "don't want to be alive", "want to be dead",
	"rather be dead", "take my own life", "ending my life",
}

var violenceDirect = []string{
	"kill someone", "hurt someone else", "attack someone", "planning to hurt", "going to attack",
}

var immediateDanger = []string{
	"have the pills", "wrote my note", "goodbye world", "this is the end",
	"can't go on anymore", "ready to end it", "last message", "won't be here tomorrow",
	"my final goodbye", "no longer able to cope", "too painful to continue",
}

var genericHelp = regexp.MustCompile(`(?i)\b(help with|help finding|help understanding|technical help|math help|school help|help me (?:with|find))\b`)

var crisisPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\b(want to|going to|might|will|plan to).{0,30}(kill myself|end my life|hurt myself|die tonight|end it all)\b`),
	regexp.MustCompile(`(?i)\b(thinking about|thoughts of).{0,20}(suicide|killing myself|ending it all|dying|hurting myself)\b`),
	regexp.MustCompile(`(?i)\b(can't|cannot|don't want to).{0,20}(go on|take it anymore|live like this|be here anymore|continue|keep going)\b`),
	regexp.MustCompile(`(?i)\b(tonight|today|right now|soon).{0,30}(kill myself|end it|hurt myself|die|won't be here)\b`),
	regexp.MustCompile(`(?i)\b(pills|rope|bridge|gun|knife|razor|blade).{0,30}(end it|kill myself|hurt myself|suicide|die)\b`),
}

var methodContent = regexp.MustCompile(`(?i)(how (much|many|to)|dosage|lethal|overdose|\btie a\b|noose|hang (yourself|myself|from)|jump (from|off)|\d+\s*(pills|mg)|(take|swallow|use)\s+(all|the|\d+|those|these)\s*(pills|of|tablets)?|step[ -]?by[ -]?step|instructions (for|to|on))`)

func containsMethodContent(s string) bool { return methodContent.MatchString(s) }

type keywordScreen struct {
	hasKeywords bool
	hasPatterns bool
	keywords    []string
	confidence  string
}

func screenCrisisKeywords(message string) keywordScreen {
	if genericHelp.MatchString(message) {
		return keywordScreen{}
	}
	lower := strings.ToLower(message)
	var found []string
	var sawSelfHarm, sawViolence, sawDanger bool

	for _, k := range selfHarmDirect {
		if strings.Contains(lower, k) {
			found = append(found, k)
			sawSelfHarm = true
		}
	}
	for _, k := range violenceDirect {
		if strings.Contains(lower, k) {
			found = append(found, k)
			sawViolence = true
		}
	}
	for _, k := range immediateDanger {
		if strings.Contains(lower, k) {
			found = append(found, k)
			sawDanger = true
		}
	}

	var patterns bool
	for _, p := range crisisPatterns {
		if p.MatchString(message) {
			patterns = true
			break
		}
	}

	conf := "none"
	switch {
	case patterns:
		conf = "high"
	case sawSelfHarm, sawViolence:
		conf = "medium"
	case sawDanger:
		conf = "low"
	}

	return keywordScreen{
		hasKeywords: len(found) > 0,
		hasPatterns: patterns,
		keywords:    found,
		confidence:  conf,
	}
}

const crisisClassifySystem = "You are a careful triage assistant for a mental health support bot. Classify the user's message for crisis risk. Reply with exactly one line in the form LEVEL: reason, where LEVEL is one of none, low, medium, high, critical. Be conservative: only use high or critical when there is a clear expression of intent, plan, or means to cause serious harm."

var levelRank = map[string]int{"none": 0, "low": 1, "medium": 2, "high": 3, "critical": 4}

func maxLevel(a, b string) string {
	if levelRank[a] >= levelRank[b] {
		return a
	}
	return b
}

func (c *Client) AnalyzeCrisis(ctx context.Context, text string) (CrisisResult, error) {
	screen := screenCrisisKeywords(text)
	res := CrisisResult{Level: "none", Keywords: screen.keywords}

	if !screen.hasKeywords && !screen.hasPatterns {
		return res, nil
	}

	if !c.live {
		if screen.hasPatterns {
			res.Level, res.NeedsSupport, res.Respond = "high", true, true
		} else {
			res.Level, res.NeedsSupport = "medium", true
		}
		return res, nil
	}

	classified, summary := "", ""
	if out, err := c.constrainedOneShot(ctx, crisisClassifySystem, text); err == nil {
		classified, summary = parseCrisisLine(out)
	}
	res.Summary = summary

	res.Level = maxLevel("medium", classified)
	res.NeedsSupport = true
	res.Respond = levelRank[classified] >= levelRank["high"]

	if classified == "" && screen.hasPatterns {
		res.Level, res.Respond = "high", true
	}
	return res, nil
}

func parseCrisisLine(s string) (level, reason string) {
	s = strings.TrimSpace(s)
	if i := strings.IndexAny(s, ":-"); i >= 0 {
		level = strings.ToLower(strings.TrimSpace(s[:i]))
		reason = strings.TrimSpace(s[i+1:])
	} else {
		level = strings.ToLower(strings.Fields(s + " ")[0])
	}
	switch level {
	case "none", "low", "medium", "high", "critical":
		return level, reason
	}
	return "", s
}

const crisisLeadInSystem = "You are Mellow, a mental health companion. The user is in distress. " +
	"Reply with ONE short, warm sentence (under 30 words) that acknowledges their feelings and that they reached out. " +
	"Do NOT give advice. Do NOT ask questions. Do NOT mention any method of self-harm. Do NOT try to talk them out of anything. " +
	"Do NOT roleplay. Just acknowledge, then stop."

func (c *Client) CrisisResponse(ctx context.Context, level, userMessage string) string {
	return c.CrisisResponseFor(ctx, level, userMessage, CrisisResourceBlock)
}

func (c *Client) CrisisResponseFor(ctx context.Context, level, userMessage, block string) string {
	if block == "" {
		block = CrisisResourceBlock
	}
	if levelRank[level] >= levelRank["high"] {
		return criticalLead + "\n\n" + block
	}

	if !c.live {
		return mediumLead + "\n\n" + block
	}
	lead, err := c.constrainedOneShot(ctx, crisisLeadInSystem, userMessage)
	lead = strings.TrimSpace(lead)
	if err != nil || lead == "" || len(lead) > 400 || containsMethodContent(lead) {
		return mediumLead + "\n\n" + block
	}
	return lead + "\n\n" + block
}

const crisisResourcesIntroSystem = "You are Mellow, a mental health companion. Write TWO short, warm sentences telling the user support is available and they deserve it. " +
	"Do NOT list hotlines or numbers yourself. Do NOT give methods or step-by-step advice. Do NOT ask questions. Keep it under 50 words."

func (c *Client) CrisisResourcesText(ctx context.Context, situation string) string {
	return c.CrisisResourcesTextFor(ctx, situation, CrisisResourceBlock)
}

func (c *Client) CrisisResourcesTextFor(ctx context.Context, situation, block string) string {
	if block == "" {
		block = CrisisResourceBlock
	}
	intro := "You reached out, and that matters. Support is available and you deserve it."
	if c.live {
		if out, err := c.constrainedOneShot(ctx, crisisResourcesIntroSystem, situation); err == nil {
			out = strings.TrimSpace(out)
			if out != "" && len(out) <= 400 && !containsMethodContent(out) {
				intro = out
			}
		}
	}
	return intro + "\n\n" + block
}
