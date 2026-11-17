package series

import (
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/calendar"
	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
)

func TestTheSeriesConformsToCalendarOnFullAirDays(t *testing.T) {
	sh := open(t)
	r := newShow(81001, "Lumen", 0, 6)
	for i, aired := range []string{"2026-11-03", "2026-11-03", "2026-10-31", "2026-12-01", "2026-11", ""} {
		r.Episodes[i].AirDate = aired
	}
	tree := sh.add(t, r)
	day := calendar.NewDay(2026, time.November, 1)
	span, err := calendar.NewSpan(day, day.Add(30))
	if err != nil {
		t.Fatal(err)
	}
	on := calendar.NewDay(2026, time.November, 3)
	contenttypetest.Calendar(t, contenttypetest.CalendarCase{Library: sh.lib, Type: sh.s, Span: span, Index: "series_episode_air", Want: []calendar.Dated{
		{Work: tree.ID, Item: tree.Items[0].ID, Kind: airKind.Key, Day: on},
		{Work: tree.ID, Item: tree.Items[1].ID, Kind: airKind.Key, Day: on},
	}})
}
