package notifications

import (
	"encoding/json"
	"reflect"
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
		name string
		got  entry.Message
		want entry.Message
	}{
		{"one new item", render(history.NewItemsRow, history.HealthFacts{Items: []history.Item{disk}}), entry.Message{
			Title: "New Health item for Disk D: is almost full", Lines: []string{"12 GB are free.", "It has been on Health since 14:02."},
			Link: entry.Link{Path: "/health#health-storage-d"}, Tone: entry.Warn,
		}},
		{"two new items", render(history.NewItemsRow, history.HealthFacts{Items: []history.Item{disk, node}}), entry.Message{
			Title: "2 new Health items", Lines: []string{"Disk D: is almost full", "Attic is offline"}, Tone: entry.Fail,
		}},
		{"one cleared item", render(history.ClearedRow, history.HealthFacts{Items: []history.Item{disk}}), entry.Message{
			Title: "Health item for Disk D: is almost full cleared", Lines: []string{"It was on Health from 14:02 to 14:31."}, Tone: entry.Good,
		}},
		{"test", render(testRow, testFacts{}), entry.Message{
			Title: "Test from SlipStream", Lines: []string{"This connection works.", "Den sent this message at 14:40."},
		}},
	} {
		if !reflect.DeepEqual(tc.got, tc.want) {
			t.Errorf("%s: %+v, want %+v", tc.name, tc.got, tc.want)
		}
	}
}
