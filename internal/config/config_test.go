package config

import "testing"

func TestWebsiteDefaultsAndDerivation(t *testing.T) {
	t.Setenv("TOKEN", "x")
	t.Setenv("CLIENT_ID", "x")
	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("WEBSITE_URL", "https://mellow.codemeapixel.dev/")

	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.WebsiteURL != "https://mellow.codemeapixel.dev" {
		t.Errorf("website = %q", c.WebsiteURL)
	}
	if c.DocsURL != "https://mellow.codemeapixel.dev/docs" {
		t.Errorf("docs = %q", c.DocsURL)
	}
	if c.SupportURL != "https://mellow.codemeapixel.dev/support" {
		t.Errorf("support = %q", c.SupportURL)
	}
	if c.InviteURL != "https://mellow.codemeapixel.dev/invite" {
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
	if c.SupportURL != "https://example.test/support" {
		t.Errorf("support = %q", c.SupportURL)
	}
}
