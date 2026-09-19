package discord

import (
	"testing"

	"github.com/CodeMeAPixel/Mellow/internal/db/gen"
)

func TestJournalTagsAndFilter(t *testing.T) {
	entries := []gen.JournalEntry{
		{ID: 1, Content: "Rough day at work #Work #stress"},
		{ID: 2, Content: "Long walk by the river #outside"},
		{ID: 3, Content: "Talked to mum, felt lighter #family #work"},
		{ID: 4, Content: "no tags here, just work stuff"},
	}

	if got := journalTags("a #Work b #work #x_1"); len(got) != 2 || got[0] != "work" || got[1] != "x_1" {
		t.Errorf("tags = %v", got)
	}

	ids := func(es []gen.JournalEntry) (out []int32) {
		for _, e := range es {
			out = append(out, e.ID)
		}
		return
	}
	cases := []struct {
		search, tag string
		want        []int32
	}{
		{"", "work", []int32{1, 3}},
		{"", "#WORK", []int32{1, 3}},
		{"river", "", []int32{2}},
		{"work", "", []int32{1, 3, 4}},
		{"work", "family", []int32{3}},
		{"", "missing", nil},
	}
	for _, tc := range cases {
		got := ids(filterJournal(entries, tc.search, tc.tag))
		if len(got) != len(tc.want) {
			t.Errorf("search=%q tag=%q got %v want %v", tc.search, tc.tag, got, tc.want)
			continue
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Errorf("search=%q tag=%q got %v want %v", tc.search, tc.tag, got, tc.want)
			}
		}
	}
}
