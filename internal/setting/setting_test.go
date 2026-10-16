package setting_test

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/setting"
)

var rss = setting.Duration{
	Key: "tasks.rss.cadence", Label: "RSS sync",
	Default: 15 * time.Minute, Min: 10 * time.Minute, Max: 2 * time.Hour, Step: time.Minute,
	Floor: "Private trackers ban for hammering.",
}

var lifetime = setting.Duration{
	Key: "nodes.joinCodeLifetime", Label: "Join Code lifetime",
	Default: time.Hour, Min: 10 * time.Minute, Max: 7 * 24 * time.Hour, Step: time.Minute,
}

func ptr[T any](v T) *T { return &v }

func sentence(err error) string {
	var r *setting.RangeError
	if !errors.As(err, &r) {
		return "not a RangeError: " + err.Error()
	}
	return r.Sentence
}

func TestDurationParse(t *testing.T) {
	for _, tc := range []struct {
		name    string
		d       setting.Duration
		seconds *int64
		want    *int64
		refusal string
	}{
		{"null resets", rss, nil, nil, ""},
		{"inside the range", rss, ptr[int64](1800), ptr[int64](1800), ""},
		{"the minimum", rss, ptr[int64](600), ptr[int64](600), ""},
		{"below a floor", rss, ptr[int64](540), nil, "No faster than 10 min. Private trackers ban for hammering."},
		{"below a minimum with no floor", lifetime, ptr[int64](540), nil, "Between 10 min and 7 days."},
		{"above the maximum", lifetime, ptr[int64](8 * 86400), nil, "Between 10 min and 7 days."},
		{"off the step", rss, ptr[int64](630), nil, "In whole minutes."},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.d.Parse(tc.seconds)
			if tc.refusal != "" {
				if s := sentence(err); s != tc.refusal {
					t.Fatalf("refusal = %q, want %q", s, tc.refusal)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if o := got.Override(); (o == nil) != (tc.want == nil) || (o != nil && *o != *tc.want) {
				t.Fatalf("override = %v, want %v", o, tc.want)
			}
		})
	}
}

func TestCountParse(t *testing.T) {
	jitter := setting.Count{Key: "tasks.jitterPercent", Label: "Jitter", Default: 10, Min: 0, Max: 50, Unit: "%"}
	runs := setting.Count{Key: "tasks.runsKept", Label: "Runs kept", Default: 50, Min: 1, Max: 1000, Unit: "runs"}
	for _, tc := range []struct {
		name    string
		c       setting.Count
		n       int64
		refusal string
	}{
		{"inside", runs, 1000, ""},
		{"below", runs, 0, "Between 1 and 1000 runs."},
		{"above a percentage", jitter, 51, "Between 0 and 50%."},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.c.Parse(&tc.n)
			if tc.refusal != "" {
				if s := sentence(err); s != tc.refusal {
					t.Fatalf("refusal = %q, want %q", s, tc.refusal)
				}
				return
			}
			if err != nil || *got.Override() != tc.n {
				t.Fatalf("Parse(%d) = %v, %v", tc.n, got.Override(), err)
			}
		})
	}
}

type overdue string

var rule = setting.Choice[overdue]{
	Key: "tasks.overdue", Label: "Overdue tasks", Default: "catch-up",
	Options: []setting.Option[overdue]{{Value: "catch-up", Label: "Catch up"}, {Value: "wait", Label: "Wait"}},
}

func TestChoiceRefusesAValueOutsideItsOptions(t *testing.T) {
	if _, err := rule.Parse(ptr("later")); sentence(err) != "Choose Catch up or Wait." {
		t.Fatalf("Parse(later) = %v", err)
	}
	got, err := rule.Parse(ptr("wait"))
	if err != nil || *got.Override() != "wait" {
		t.Fatalf("Parse(wait) = %v, %v", got.Override(), err)
	}
}

var window = setting.Window{Key: "tasks.window", Label: "Maintenance window", Default: setting.Span{Start: 240, End: 360}}

func TestWindowParse(t *testing.T) {
	for _, tc := range []struct {
		name    string
		times   setting.Times
		want    setting.Span
		refusal string
	}{
		{"crossing midnight", setting.Times{Start: "23:30", End: "01:00"}, setting.Span{Start: 1410, End: 60}, ""},
		{"empty", setting.Times{Start: "04:00", End: "04:00"}, setting.Span{}, "The window needs an end after its start."},
		{"not a time", setting.Times{Start: "24:00", End: "01:00"}, setting.Span{}, "Times are 00:00 to 23:59."},
		{"no leading zero", setting.Times{Start: "4:00", End: "06:00"}, setting.Span{}, "Times are 00:00 to 23:59."},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := window.Parse(&tc.times)
			if tc.refusal != "" {
				if s := sentence(err); s != tc.refusal {
					t.Fatalf("refusal = %q, want %q", s, tc.refusal)
				}
				return
			}
			if err != nil || *got.Override() != tc.want {
				t.Fatalf("Parse = %v, %v", got.Override(), err)
			}
		})
	}
}

func TestFieldReportsIsDefault(t *testing.T) {
	for _, tc := range []struct {
		name  string
		field any
		want  string
	}{
		{
			"a duration at its default", lifetime.Field(nil),
			`{"key":"nodes.joinCodeLifetime","label":"Join Code lifetime","source":"database","applies":"live","isDefault":true,"kind":"duration","value":3600,"default":3600,"min":600,"max":604800,"step":60}`,
		},
		{
			"a duration with a floor, overridden", rss.Field(ptr[int64](1800)),
			`{"key":"tasks.rss.cadence","label":"RSS sync","source":"database","applies":"live","isDefault":false,"kind":"duration","value":1800,"default":900,"min":600,"max":7200,"step":60,"floor":"Private trackers ban for hammering."}`,
		},
		{
			"a count", setting.Count{Key: "tasks.runsKept", Label: "Runs kept", Default: 50, Min: 1, Max: 1000, Unit: "runs"}.Field(ptr[int64](20)),
			`{"key":"tasks.runsKept","label":"Runs kept","source":"database","applies":"live","isDefault":false,"kind":"count","value":20,"default":50,"min":1,"max":1000,"unit":"runs"}`,
		},
		{
			"a choice following another setting", setting.Choice[overdue]{Key: "tasks.housekeeping.overdue", Label: "When overdue", Default: "wait", DefaultFrom: "Scheduled tasks", Options: rule.Options}.Field(nil),
			`{"key":"tasks.housekeeping.overdue","label":"When overdue","defaultFrom":"Scheduled tasks","source":"database","applies":"live","isDefault":true,"kind":"choice","value":"wait","default":"wait","options":[{"value":"catch-up","label":"Catch up"},{"value":"wait","label":"Wait"}]}`,
		},
		{
			"a window", window.Field(&setting.Span{Start: 60, End: 180}, "Europe/London"),
			`{"key":"tasks.window","label":"Maintenance window","source":"database","applies":"live","isDefault":false,"kind":"window","value":{"start":"01:00","end":"03:00"},"default":{"start":"04:00","end":"06:00"},"zone":"Europe/London"}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b, err := json.Marshal(tc.field)
			if err != nil {
				t.Fatal(err)
			}
			if string(b) != tc.want {
				t.Fatalf("Field =\n%s\nwant\n%s", b, tc.want)
			}
		})
	}
}

func TestCheckRefusesADefaultOutsideItsRange(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		want string
	}{
		{"duration", setting.Duration{Key: "d", Default: time.Minute, Min: 10 * time.Minute, Max: time.Hour, Step: time.Minute}.Check(), "setting d: default 1m0s is outside 10m0s to 1h0m0s"},
		{"step", setting.Duration{Key: "d", Default: time.Hour, Min: time.Hour, Max: time.Hour, Step: 2 * time.Hour}.Check(), "setting d: step 2h0m0s is not a second, minute, hour or day"},
		{"count", setting.Count{Key: "c", Default: 0, Min: 1, Max: 3}.Check(), "setting c: default 0 is outside 1 to 3"},
		{"choice", setting.Choice[overdue]{Key: "o", Default: "later", Options: rule.Options}.Check(), `setting o: default "later" is not an option`},
		{"window", setting.Window{Key: "w", Default: setting.Span{Start: 60, End: 60}}.Check(), "setting w: default {Start:60 End:60} is not a non-empty daily span"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.err == nil || tc.err.Error() != tc.want {
				t.Fatalf("Check = %v, want %s", tc.err, tc.want)
			}
		})
	}
	if err := lifetime.Check(); err != nil {
		t.Fatalf("a sound declaration: %v", err)
	}
}
