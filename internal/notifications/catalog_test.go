package notifications

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/notifications/notice"
)

func TestTheOwnRowsRenderTheirWords(t *testing.T) {
	at := func(h, m int) time.Time { return time.Date(2026, 11, 16, h, m, 0, 0, time.UTC) }
	c := notice.Context{Now: at(14, 40), Zone: time.UTC, Sender: "Den"}
	disk := item{Category: "storage", Key: "d", Level: "warn", Subject: "Disk D: is almost full", Text: "12 GB are free.", Since: at(14, 2), Cleared: at(14, 31)}
	node := item{Category: "nodes", Key: "attic", Level: "fail", Subject: "Attic is offline", Text: "Attic hasn't been seen since 13:00.", Since: at(13, 5)}
	render := func(row notice.Entry, f any) notice.Message {
		t.Helper()
		b, err := json.Marshal(f)
		if err != nil {
			t.Fatal(err)
		}
		m, err := row.RenderStored(b, c)
		if err != nil {
			t.Fatal(err)
		}
		return notice.Message{Title: m.Title, Lines: m.Lines, Link: m.Link, Tone: m.Tone}
	}
	for _, tc := range []struct {
		name string
		got  notice.Message
		want notice.Message
	}{
		{"one new item", render(newItemsRow, healthFacts{Items: []item{disk}}), notice.Message{
			Title: "Disk D: is almost full", Lines: []string{"12 GB are free.", "It has been on Health since 14:02."},
			Link: notice.Link{Path: "/health#health-storage-d"}, Tone: notice.Warn,
		}},
		{"two new items", render(newItemsRow, healthFacts{Items: []item{disk, node}}), notice.Message{
			Title: "2 new Health items", Lines: []string{"Disk D: is almost full", "Attic is offline"}, Tone: notice.Fail,
		}},
		{"one cleared item", render(clearedRow, healthFacts{Items: []item{disk}}), notice.Message{
			Title: "Cleared: Disk D: is almost full", Lines: []string{"It was on Health from 14:02 to 14:31."}, Tone: notice.Good,
		}},
		{"test", render(testRow, testFacts{}), notice.Message{
			Title: "Test from SlipStream", Lines: []string{"This connection works.", "Den sent this message at 14:40."},
		}},
	} {
		if !reflect.DeepEqual(tc.got, tc.want) {
			t.Errorf("%s: %+v, want %+v", tc.name, tc.got, tc.want)
		}
	}
}
