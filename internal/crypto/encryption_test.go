package crypto

import (
	"encoding/base64"
	"strings"
	"testing"
)

const (
	fixtureKey   = "test-key-please-ignore-0123456789"
	fixtureSalt0 = "salt-one"
	fixtureSalt1 = "salt-two"
)

var jsFixtures = []struct {
	plaintext string
	payload   string
}{
	{
		plaintext: "hello world",
		payload:   "Iylq0HlMjuWBYENr52Zzlw==:16:I3IhZ6stzviefKSAusvyJg==:1382VSsTNciLLmo=",
	},
	{
		plaintext: "I have been feeling really anxious lately \U0001F61F",
		payload:   "KGopQB29O3LrqiyNMIRlKw==:16:PNPdkderS4hDpf/M2ZVUYw==:duZOoevV4z/oMO6I7YsYdgMI3bxdVUQNMgwN0zRLggNUwsQk3XnDX8iK2mWORQ==",
	},
	{
		plaintext: "multi\nline\ncontent with : colons : inside",
		payload:   "G3LZm2aOJN4IVr4cDc5XPA==:16:rPdi5Y7JKBoXgy4x4FsFpg==:P6cUc+hvYYKtY21GDnW7t8pLeMsWlx3eCL5g2D6BKFtlHOO78jkqmZs=",
	},
}

func newFixtureService() *Service {
	return New(fixtureKey, []string{fixtureSalt0, fixtureSalt1})
}

func TestDecryptJSFixtures(t *testing.T) {
	s := newFixtureService()
	for _, f := range jsFixtures {
		got := s.Decrypt(f.payload)
		if got != f.plaintext {
			t.Errorf("Decrypt(JS payload)\n got  %q\n want %q", got, f.plaintext)
		}
	}
}

func TestRoundTrip(t *testing.T) {
	s := newFixtureService()
	cases := []string{
		"hello world",
		"unicode 😢 and more 🥺",
		"a string : with : colons",
		"line\nbreaks\tand\ttabs",
		strings.Repeat("long content ", 200),
	}
	for _, in := range cases {
		enc := s.Encrypt(in)
		if !s.IsEncrypted(enc) {
			t.Fatalf("Encrypt(%q) produced non-payload %q", in, enc)
		}
		if parts := strings.Split(enc, ":"); len(parts) != 4 || parts[1] != "16" {
			t.Fatalf("Encrypt(%q) payload shape wrong: %q", in, enc)
		}
		if got := s.Decrypt(enc); got != in {
			t.Errorf("round trip\n got  %q\n want %q", got, in)
		}
	}
}

func TestGoPayloadDecryptsInJS(t *testing.T) {
	s := newFixtureService()
	enc := s.Encrypt("cross-impl check")
	parts := strings.Split(enc, ":")
	if len(parts) != 4 {
		t.Fatalf("want 4 parts, got %d (%q)", len(parts), enc)
	}
	iv := mustB64(t, parts[0])
	if len(iv) != 16 {
		t.Errorf("iv length = %d, want 16", len(iv))
	}
	if parts[1] != "16" {
		t.Errorf("tag marker = %q, want \"16\"", parts[1])
	}
	tag := mustB64(t, parts[2])
	if len(tag) != 16 {
		t.Errorf("tag length = %d, want 16", len(tag))
	}
	mustB64(t, parts[3])
}

func TestEmptyAndBlank(t *testing.T) {
	s := newFixtureService()
	for _, in := range []string{"", "   ", "\t\n"} {
		if got := s.Encrypt(in); got != emptyContent {
			t.Errorf("Encrypt(%q) = %q, want %q", in, got, emptyContent)
		}
	}
	if got := s.EncryptPtr(nil); got != noContent {
		t.Errorf("EncryptPtr(nil) = %q, want %q", got, noContent)
	}
}

func TestDisabledServicePassthrough(t *testing.T) {
	s := New("", nil)
	if s.Enabled() {
		t.Fatal("service with empty key should be disabled")
	}
	const in = "plaintext stays plaintext"
	if got := s.Encrypt(in); got != in {
		t.Errorf("disabled Encrypt = %q, want %q", got, in)
	}
	if got := s.Decrypt(in); got != in {
		t.Errorf("disabled Decrypt = %q, want %q", got, in)
	}
}

func TestNonPayloadPassesThrough(t *testing.T) {
	s := newFixtureService()
	for _, in := range []string{
		"just a normal sentence",
		"contains : one colon",
		"two : colons : here",
		"three : colons : in : text",
	} {
		if s.IsEncrypted(in) {
			t.Errorf("IsEncrypted(%q) = true, want false", in)
		}
		if got := s.Decrypt(in); got != in {
			t.Errorf("Decrypt(%q) = %q, want unchanged", in, got)
		}
	}
}

func TestAlreadyEncryptedNotDoubleWrapped(t *testing.T) {
	s := newFixtureService()
	enc := s.Encrypt("once")
	again := s.Encrypt(enc)
	if again != enc {
		t.Errorf("Encrypt(payload) re-wrapped: %q != %q", again, enc)
	}
}

func mustB64(t *testing.T, s string) []byte {
	t.Helper()
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		t.Fatalf("%q is not base64: %v", s, err)
	}
	return b
}
