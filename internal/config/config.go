package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Token          string
	ClientID       string
	PrivateGuildID string
	OwnerIDs       []string

	DatabaseURL string

	AnthropicAPIKey string

	EncryptionKey  string
	EncryptionSalt []string

	Port         int
	StatusAPIKey string
	StatusAPIURL string
	APIToken     string
	LogChannelID string
	LogLevel     string

	WebsiteURL string
	DocsURL    string
	SupportURL string
	InviteURL  string
	SourceURL  string

	GitHubToken string
	GitHubRepo  string

	OmniplexToken   string
	OmniplexBaseURL string

	SKUMellowPlus string
}

const defaultWebsite = "https://mymellow.xyz"

func Load() (*Config, error) {
	c := &Config{
		Token:           os.Getenv("TOKEN"),
		ClientID:        os.Getenv("CLIENT_ID"),
		PrivateGuildID:  os.Getenv("PRIVATE_GUILD_ID"),
		OwnerIDs:        splitCSV(os.Getenv("OWNER_IDS")),
		DatabaseURL:     os.Getenv("DATABASE_URL"),
		AnthropicAPIKey: os.Getenv("ANTHROPIC_API_KEY"),
		EncryptionKey:   os.Getenv("ENCRYPTION_KEY"),
		EncryptionSalt:  splitSalts(os.Getenv("ENCRYPTION_SALT_LIST")),
		StatusAPIKey:    os.Getenv("MELLOW_STATUS_API_KEY"),
		APIToken:        os.Getenv("API_TOKEN"),
		LogChannelID:    os.Getenv("LOG_CHANNEL_ID"),
		LogLevel:        firstNonEmpty(os.Getenv("LOG_LEVEL"), "info"),
	}

	port := firstNonEmpty(os.Getenv("PORT"), "9420")
	p, err := strconv.Atoi(port)
	if err != nil {
		return nil, fmt.Errorf("invalid PORT %q: %w", port, err)
	}
	c.Port = p

	var missing []string
	if c.Token == "" {
		missing = append(missing, "TOKEN")
	}
	if c.ClientID == "" {
		missing = append(missing, "CLIENT_ID")
	}
	if c.DatabaseURL == "" {
		missing = append(missing, "DATABASE_URL")
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required env vars: %s", strings.Join(missing, ", "))
	}

	if len(c.EncryptionSalt) == 0 {
		c.EncryptionSalt = []string{"mellow-encryption-salt"}
	}

	c.WebsiteURL = strings.TrimRight(firstNonEmpty(os.Getenv("WEBSITE_URL"), defaultWebsite), "/")
	c.DocsURL = firstNonEmpty(os.Getenv("DOCS_URL"), c.WebsiteURL+"/docs")
	c.SupportURL = firstNonEmpty(os.Getenv("SUPPORT_URL"), c.WebsiteURL+"/support")
	c.InviteURL = firstNonEmpty(os.Getenv("INVITE_URL"), c.WebsiteURL+"/invite")
	c.SourceURL = firstNonEmpty(os.Getenv("SOURCE_URL"), "https://github.com/CodeMeAPixel/Mellow")
	c.StatusAPIURL = firstNonEmpty(os.Getenv("STATUS_API_URL"), c.WebsiteURL+"/api/status")
	c.GitHubToken = os.Getenv("GITHUB_TOKEN")
	c.GitHubRepo = firstNonEmpty(os.Getenv("GITHUB_REPO"), repoFromURL(c.SourceURL), "CodeMeAPixel/Mellow")
	if stripped := repoFromURL(c.GitHubRepo); stripped != "" {
		c.GitHubRepo = stripped
	}
	c.OmniplexToken = os.Getenv("OMNIPLEX_TOKEN")
	c.OmniplexBaseURL = os.Getenv("OMNIPLEX_BASE_URL")
	c.SKUMellowPlus = os.Getenv("SKU_MELLOW_PLUS_ID")

	return c, nil
}

func (c *Config) EncryptionEnabled() bool { return c.EncryptionKey != "" }

func splitCSV(raw string) []string { return splitSalts(raw) }

func repoFromURL(u string) string {
	for _, p := range []string{"https://github.com/", "http://github.com/", "github.com/"} {
		if strings.HasPrefix(u, p) {
			return strings.TrimSuffix(strings.TrimPrefix(u, p), ".git")
		}
	}
	return ""
}

func splitSalts(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func firstNonEmpty(vs ...string) string {
	for _, v := range vs {
		if v != "" {
			return v
		}
	}
	return ""
}
