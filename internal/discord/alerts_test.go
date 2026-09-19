package discord

import (
	"strings"
	"testing"
)

func TestAlertRecipients(t *testing.T) {
	main := "111"
	extra := []string{"222", "333", "111", "444", "555"}

	if got := alertRecipients(&main, extra, false); len(got) != 1 || got[0] != "111" {
		t.Errorf("free servers should only use the main channel, got %v", got)
	}
	got := alertRecipients(&main, extra, true)
	want := []string{"111", "222", "333", "444"}
	if len(got) != len(want) {
		t.Fatalf("got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("got %v want %v", got, want)
		}
	}
	if got := alertRecipients(nil, nil, true); len(got) != 0 {
		t.Errorf("no channels configured should send nowhere, got %v", got)
	}
	empty := ""
	if got := alertRecipients(&empty, nil, false); len(got) != 0 {
		t.Errorf("an empty channel id is not a channel, got %v", got)
	}
}

func TestAlertTextNeverRepeatsMessageContent(t *testing.T) {
	text := alertText(42, "high", "1", "2", "3")
	for _, want := range []string{"<@42>", "(high)", "channels/1/2/3", "not repeated"} {
		if !strings.Contains(text, want) {
			t.Errorf("alert missing %q:\n%s", want, text)
		}
	}
	if strings.Contains(alertText(42, "high", "1", "", ""), "Jump to") {
		t.Error("no link expected without a message")
	}
}
