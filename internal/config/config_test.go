package config

import "testing"

func TestWebsiteDefaultsAndDerivation(t *testing.T) {
	t.Setenv("TOKEN", "x")
	t.Setenv("CLIENT_ID", "x")
	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("WEBSITE_URL", "https://mymellow.xyz/")

	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.WebsiteURL != "https://mymellow.xyz" {
		t.Errorf("website = %q", c.WebsiteURL)
	}
	if c.DocsURL != "https://docs.mymellow.xyz" {
		t.Errorf("docs = %q", c.DocsURL)
	}
	if c.SupportURL != "https://discord.gg/cYauqJfnNK" {
		t.Errorf("support = %q", c.SupportURL)
	}
	if c.InviteURL != "https://mymellow.xyz/invite" {
		t.Errorf("invite = %q", c.InviteURL)
	}
}

func TestWebsiteExplicitOverrides(t *testing.T) {
	t.Setenv("TOKEN", "x")
	t.Setenv("CLIENT_ID", "x")
	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("WEBSITE_URL", "https://example.test")
	t.Setenv("DOCS_URL", "https://docs.example.test")

	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.DocsURL != "https://docs.example.test" {
		t.Errorf("docs override = %q", c.DocsURL)
	}
	if c.SupportURL != "https://discord.gg/cYauqJfnNK" {
		t.Errorf("support = %q", c.SupportURL)
	}
}
