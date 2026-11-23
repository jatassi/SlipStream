package notifications

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/history"
	"github.com/jatassi/slipstream/internal/history/entry"
)

func TestTheOwnRowsRenderTheirWords(t *testing.T) {
	at := func(h, m int) time.Time { return time.Date(2026, 11, 16, h, m, 0, 0, time.UTC) }
	c := entry.Context{Now: at(14, 40), Zone: time.UTC, Sender: "Den"}
	disk := history.Item{Key: "storage.d", Level: "warn", Subject: "Disk D: is almost full", Text: "12 GB are free.", Since: at(14, 2), Cleared: at(14, 31)}
	node := history.Item{Key: "nodes.attic", Level: "fail", Subject: "Attic is offline", Text: "Attic hasn't been seen since 13:00.", Since: at(13, 5)}
	render := func(row entry.Entry, f any) entry.Message {
		t.Helper()
		b, err := json.Marshal(f)
		if err != nil {
			t.Fatal(err)
		}
		m, err := row.RenderStored(b, c)
		if err != nil {
			t.Fatal(err)
		}
		return entry.Message{Title: m.Title, Lines: m.Lines, Link: m.Link, Tone: m.Tone}
	}
	for _, tc := range []struct {
		name  string
		got   entry.Message
		link  string
		tone  entry.Tone
		names []string
	}{
		{"one new item", render(history.NewItemsRow, history.HealthFacts{Items: []history.Item{disk}}), "/health#health-storage-d", entry.Warn,
			[]string{disk.Subject, disk.Text, "14:02"}},
		{"two new items", render(history.NewItemsRow, history.HealthFacts{Items: []history.Item{disk, node}}), "", entry.Fail,
			[]string{disk.Subject, node.Subject}},
		{"one cleared item", render(history.ClearedRow, history.HealthFacts{Items: []history.Item{disk}}), "", entry.Good,
			[]string{disk.Subject, "14:02", "14:31"}},
		{"test", render(testRow, testFacts{}), "", entry.Info, []string{"Den", "14:40"}},
	} {
		read := strings.Join(append([]string{tc.got.Title}, tc.got.Lines...), "\n")
		if tc.got.Title == "" || tc.got.Link.Path != tc.link || tc.got.Tone != tc.tone {
			t.Errorf("%s: %+v, want a title, link %q and tone %v", tc.name, tc.got, tc.link, tc.tone)
		}
		for _, n := range tc.names {
			if !strings.Contains(read, n) {
				t.Errorf("%s reads %q, want it to name %q", tc.name, read, n)
			}
		}
	}
}
