package setting_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
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

var keepFor = setting.Duration{
	Key: "history.keepFor", Label: "Keep history for",
	Default: 90 * 24 * time.Hour, Min: 30 * 24 * time.Hour, Max: 1095 * 24 * time.Hour, Step: 24 * time.Hour,
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
		{"below a floor", rss, ptr[int64](540), nil, "Use 10 min or longer. Private trackers ban for hammering."},
		{"below a minimum with no floor", lifetime, ptr[int64](540), nil, "Between 10 min and 7 days."},
		{"above the maximum", lifetime, ptr[int64](8 * 86400), nil, "Between 10 min and 7 days."},
		{"off the step", rss, ptr[int64](630), nil, "In whole minutes."},
		{"a range past a thousand days", keepFor, ptr[int64](29 * 86400), nil, "Between 30 days and 1,095 days."},
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
	jitter := setting.Count{Key: "tasks.jitterPercent", Label: "Jitter", Default: 10, Min: 0, Max: 50, Unit: setting.Noun{One: "%", Many: "%"}}
	runs := setting.Count{Key: "tasks.runsKept", Label: "Runs kept", Default: 50, Min: 1, Max: 1000, Unit: setting.Noun{One: "run", Many: "runs"}}
	floored := setting.Count{Key: "tasks.retries", Label: "Retries", Default: 3, Min: 1, Max: 5, Unit: setting.Noun{One: "try", Many: "tries"}, Floor: "One try at least."}
	for _, tc := range []struct {
		name    string
		c       setting.Count
		n       int64
		refusal string
	}{
		{"inside", runs, 1000, ""},
		{"below", runs, 0, "Between 1 and 1000 runs."},
		{"above a percentage", jitter, 51, "Between 0 and 50%."},
		{"below a floor of one", floored, 0, "At least 1 try. One try at least."},
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

var window = setting.Window{Key: "tasks.window", Label: "Maintenance window", Default: setting.Span{Start: 240, End: 360}}

func TestWindowParse(t *testing.T) {
	for _, tc := range []struct {
		name    string
		times   setting.Times
		want    setting.Span
		refusal string
	}{
		{"crossing midnight", setting.Times{Start: "23:30", End: "01:00"}, setting.Span{Start: 1410, End: 60}, ""},
		{"empty", setting.Times{Start: "04:00", End: "04:00"}, setting.Span{}, "The start and end are the same time. Choose a different end. A window can cross midnight, such as 22:00 to 04:00."},
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
			`{"key":"nodes.joinCodeLifetime","label":"Join Code lifetime","source":"database","applies":"live","isDefault":true,"pendingRestart":false,"kind":"duration","value":3600,"running":3600,"default":3600,"min":600,"max":604800,"step":60}`,
		},
		{
			"a duration with a floor, overridden", rss.Field(ptr[int64](1800)),
			`{"key":"tasks.rss.cadence","label":"RSS sync","source":"database","applies":"live","isDefault":false,"pendingRestart":false,"kind":"duration","value":1800,"running":1800,"default":900,"min":600,"max":7200,"step":60,"floor":"Private trackers ban for hammering."}`,
		},
		{
			"a duration that says what a longer time costs", setting.Duration{Key: "history.keepFor", Label: "Keep history for", Default: 24 * time.Hour, Min: 24 * time.Hour, Max: 48 * time.Hour, Step: 24 * time.Hour, Description: "A longer time makes each Backup larger."}.Field(nil),
			`{"key":"history.keepFor","label":"Keep history for","source":"database","applies":"live","isDefault":true,"pendingRestart":false,"kind":"duration","value":86400,"running":86400,"default":86400,"min":86400,"max":172800,"step":86400,"description":"A longer time makes each Backup larger."}`,
		},
		{
			"a count", setting.Count{Key: "tasks.runsKept", Label: "Runs kept", Default: 50, Min: 1, Max: 1000, Unit: setting.Noun{One: "run", Many: "runs"}}.Field(ptr[int64](20)),
			`{"key":"tasks.runsKept","label":"Runs kept","source":"database","applies":"live","isDefault":false,"pendingRestart":false,"kind":"count","value":20,"running":20,"default":50,"min":1,"max":1000,"unit":{"one":"run","many":"runs"}}`,
		},
		{
			"a choice following another setting", setting.Choice[overdue]{Key: "tasks.housekeeping.overdue", Label: "When overdue", Default: "wait", DefaultFrom: "Scheduled tasks", Options: rule.Options}.Field(nil),
			`{"key":"tasks.housekeeping.overdue","label":"When overdue","defaultFrom":"Scheduled tasks","source":"database","applies":"live","isDefault":true,"pendingRestart":false,"kind":"choice","value":"wait","running":"wait","default":"wait","options":[{"value":"catch-up","label":"Catch up"},{"value":"wait","label":"Wait"}]}`,
		},
		{
			"a count with words for zero", setting.Count{Key: "log.max_files", Label: "Files kept", Default: 5, Max: 100, Unit: setting.Noun{One: "file", Many: "files"}, Zero: "No limit"}.Field(ptr[int64](0)),
			`{"key":"log.max_files","label":"Files kept","source":"database","applies":"live","isDefault":false,"pendingRestart":false,"kind":"count","value":0,"running":0,"default":5,"min":0,"max":100,"unit":{"one":"file","many":"files"},"zero":"No limit"}`,
		},
		{
			"a listen address on this machine only", webListen.Field(ptr(netip.MustParseAddrPort("[::1]:9000"))),
			`{"key":"http.listen","label":"Listen on","source":"database","applies":"live","isDefault":false,"pendingRestart":false,"kind":"listen","value":{"host":"local","port":9000,"address":"[::1]:9000"},"running":{"host":"local","port":9000,"address":"[::1]:9000"},"default":{"host":"all","port":8484,"address":"[::]:8484"}}`,
		},
		{
			"a switch", setting.Switch{Key: "log.compress", Label: "Compress rotated files", Default: true}.Field(nil),
			`{"key":"log.compress","label":"Compress rotated files","source":"database","applies":"live","isDefault":true,"pendingRestart":false,"kind":"switch","value":true,"running":true,"default":true}`,
		},
		{
			"a rate with words for zero", downloadLimit.Field(ptr[int64](0)),
			`{"key":"downloading.downloadLimit","label":"Download limit","source":"database","applies":"live","isDefault":false,"pendingRestart":false,"kind":"rate","value":0,"running":0,"default":0,"max":10737418240,"zero":"No limit"}`,
		},
		{
			"an address", listen.Field(nil),
			`{"key":"torrent.listen","label":"Listen address","source":"database","applies":"live","isDefault":true,"pendingRestart":false,"kind":"address","value":"0.0.0.0:6881","running":"0.0.0.0:6881","default":"0.0.0.0:6881"}`,
		},
		{
			"a folder on a Node", downloads.Field(ptr("/srv/torrents"), "0199f0a1-1111-7000-8000-000000000001"),
			`{"key":"torrent.downloads","label":"Downloads folder","source":"database","applies":"live","isDefault":false,"pendingRestart":false,"kind":"folder","value":"/srv/torrents","running":"/srv/torrents","default":"/var/lib/slipstream/downloads","nodeId":"0199f0a1-1111-7000-8000-000000000001","purpose":"downloads"}`,
		},
		{
			"a space following another setting, overridden", warnBelow.Field(&setting.Amount{N: 10, Unit: setting.PercentUnit}),
			`{"key":"health.storageWarnBelow","label":"Warn when free space is below","defaultFrom":"Storage levels","source":"database","applies":"live","isDefault":false,"pendingRestart":false,"kind":"space","value":{"amount":10,"unit":"percent"},"running":{"amount":10,"unit":"percent"},"default":{"amount":107374182400,"unit":"bytes"},"maxBytes":1125899906842624}`,
		},
		{
			"a window", window.Field(&setting.Span{Start: 60, End: 180}, "Europe/London"),
			`{"key":"tasks.window","label":"Maintenance window","source":"database","applies":"live","isDefault":false,"pendingRestart":false,"kind":"window","value":{"start":"01:00","end":"03:00"},"running":{"start":"01:00","end":"03:00"},"default":{"start":"04:00","end":"06:00"},"zone":"Europe/London"}`,
		},
		{
			"a list at its default", setting.List{Key: "k.list", Label: "Extensions", Default: []string{"mkv"}, Min: 1, Item: setting.Extension}.Field(nil),
			`{"key":"k.list","label":"Extensions","source":"database","applies":"live","isDefault":true,"pendingRestart":false,"kind":"list","value":["mkv"],"running":["mkv"],"default":["mkv"],"min":1,"item":"extension"}`,
		},
		{
			"an empty IP address", setting.IP{Key: "reachability.stableIPv6", Label: "Stable IPv6 address", Family: setting.IPv6Only}.Field(nil),
			`{"key":"reachability.stableIPv6","label":"Stable IPv6 address","source":"database","applies":"live","isDefault":true,"pendingRestart":false,"kind":"ip","value":null,"running":null,"default":null,"family":"ipv6"}`,
		},
		{
			"a list of servers", setting.List{Key: "reachability.stunServers", Label: "STUN servers", Item: setting.HostPort}.Field(&[]string{"stun.example.org:3478"}),
			`{"key":"reachability.stunServers","label":"STUN servers","source":"database","applies":"live","isDefault":false,"pendingRestart":false,"kind":"list","value":["stun.example.org:3478"],"running":["stun.example.org:3478"],"default":[],"min":0,"item":"host-port"}`,
		},
		{
			"a size, overridden", smallestVideo.Field(ptr(100 * setting.MB)),
			`{"key":"library.importing.smallestVideo","label":"Smallest video file","source":"database","applies":"live","isDefault":false,"pendingRestart":false,"kind":"size","value":104857600,"running":104857600,"default":52428800,"min":0,"max":10737418240}`,
		},
		{
			"a template at its default", fileTemplate.Field(nil),
			`{"key":"show.naming.file.standard","label":"Standard file","source":"database","applies":"live","isDefault":true,"pendingRestart":false,"kind":"template","value":"{Title} - {Target}","default":"{Title} - {Target}","field":"file.standard","plain":false}`,
		},
		{
			"plain text, overridden", colonText.Field(ptr(" ~ ")),
			`{"key":"show.naming.colonCustom","label":"Custom colon","source":"database","applies":"live","isDefault":false,"pendingRestart":false,"kind":"template","value":" ~ ","default":" - ","field":"colonCustom","plain":true}`,
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

var webListen = setting.Listen{Key: "http.listen", Label: "Listen on", Default: netip.MustParseAddrPort("[::]:8484")}

func TestListenParseAndHost(t *testing.T) {
	for _, tc := range []struct {
		host    string
		port    int64
		want    setting.ListenAt
		refusal string
	}{
		{"::", 1, setting.ListenAt{Host: "all", Port: 1, Address: "[::]:1"}, ""},
		{"::1", 65535, setting.ListenAt{Host: "local", Port: 65535, Address: "[::1]:65535"}, ""},
		{"127.0.0.1", 8484, setting.ListenAt{Host: "other", Port: 8484, Address: "127.0.0.1:8484"}, ""},
		{"0.0.0.0", 8484, setting.ListenAt{Host: "other", Port: 8484, Address: "0.0.0.0:8484"}, ""},
		{"::", 0, setting.ListenAt{}, "Between 1 and 65535."},
		{"::", 65536, setting.ListenAt{}, "Between 1 and 65535."},
	} {
		got, err := webListen.Parse(netip.MustParseAddr(tc.host), tc.port)
		if tc.refusal != "" {
			if s := sentence(err); s != tc.refusal {
				t.Fatalf("Parse(%s, %d) refusal = %q, want %q", tc.host, tc.port, s, tc.refusal)
			}
			continue
		}
		if err != nil || setting.ListenOn(*got.Override()) != tc.want {
			t.Fatalf("Parse(%s, %d) = %v, %v; want %+v", tc.host, tc.port, got.Override(), err, tc.want)
		}
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
		{"rate", setting.Rate{Key: "r", Default: 1000, Max: 1 << 20}.Check(), "setting r: default and range are not whole steps of 1024"},
		{"address", setting.Address{Key: "a", Default: netip.MustParseAddrPort("0.0.0.0:0")}.Check(), "setting a: default 0.0.0.0:0 is not an address and port"},
		{"space", setting.Space{Key: "s", Default: setting.Amount{N: 101, Unit: setting.PercentUnit}}.Check(), "setting s: default {N:101 Unit:percent} is not a space: s: Between 0 and 100%."},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.err == nil || tc.err.Error() != tc.want {
				t.Fatalf("Check = %v, want %s", tc.err, tc.want)
			}
		})
	}
	if err := warnBelow.Check(); err != nil {
		t.Fatalf("a sound space: %v", err)
	}
	if err := lifetime.Check(); err != nil {
		t.Fatalf("a sound declaration: %v", err)
	}
}

func TestParseJSONReadsEachKindsWireValue(t *testing.T) {
	pick := setting.Choice[string]{Key: "k.choice", Default: "a", Options: []setting.Option[string]{{Value: "a", Label: "A"}, {Value: "b", Label: "B"}}}
	var countries []setting.Option[string]
	for _, c := range []string{"AT", "BE", "CH", "DE", "ES", "FR", "GB"} {
		countries = append(countries, setting.Option[string]{Value: c, Label: c})
	}
	country := setting.Choice[string]{Key: "k.country", Label: "Country", Default: "GB", Options: countries}
	runsKept := setting.Count{Key: "k.count", Default: 5, Min: 1, Max: 10}
	follow := setting.Switch{Key: "k.switch", Default: true}
	for _, tc := range []struct {
		name    string
		parse   func(json.RawMessage) (string, error)
		raw     string
		want    string
		refusal string
	}{
		{"a duration in seconds", stored(rss.ParseJSON), `1800`, "1800", ""},
		{"a duration out of range", stored(rss.ParseJSON), `60`, "", "Use 10 min or longer. Private trackers ban for hammering."},
		{"a duration as a word", stored(rss.ParseJSON), `"soon"`, "", "Seconds, as a whole number."},
		{"null resets a duration", stored(rss.ParseJSON), `null`, "reset", ""},
		{"a count", stored(runsKept.ParseJSON), `7`, "7", ""},
		{"a count as a fraction", stored(runsKept.ParseJSON), `7.5`, "", "A whole number."},
		{"a choice", stored(pick.ParseJSON), `"b"`, "b", ""},
		{"a choice outside its options", stored(pick.ParseJSON), `"c"`, "", "Choose A or B."},
		{"a choice outside a long list", stored(country.ParseJSON), `"ZZ"`, "", "ZZ is not one of the 7 choices for Country."},
		{"a switch", stored(follow.ParseJSON), `false`, "false", ""},
		{"a switch as a number", stored(follow.ParseJSON), `0`, "", "On or off: true or false."},
		{"null resets a switch", stored(follow.ParseJSON), ` null `, "reset", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.parse(json.RawMessage(tc.raw))
			if tc.refusal != "" {
				if s := sentence(err); s != tc.refusal {
					t.Fatalf("refusal = %q, want %q", s, tc.refusal)
				}
				return
			}
			if err != nil || got != tc.want {
				t.Fatalf("ParseJSON(%s) = %q, %v, want %q", tc.raw, got, err, tc.want)
			}
		})
	}
}

var downloadLimit = setting.Rate{Key: "downloading.downloadLimit", Label: "Download limit", Max: 10 << 30, Zero: "No limit"}

func TestRateParse(t *testing.T) {
	for _, tc := range []struct {
		name    string
		n       *int64
		want    *int64
		refusal string
	}{
		{"null resets", nil, nil, ""},
		{"zero", ptr[int64](0), ptr[int64](0), ""},
		{"whole KB/s", ptr[int64](1536 << 10), ptr[int64](1536 << 10), ""},
		{"the maximum", ptr[int64](10 << 30), ptr[int64](10 << 30), ""},
		{"above the maximum", ptr[int64](10<<30 + 1024), nil, "Up to 10240 MB/s."},
		{"below zero", ptr[int64](-1024), nil, "Up to 10240 MB/s."},
		{"off a whole KB/s", ptr[int64](1500), nil, "In whole KB/s."},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := downloadLimit.Parse(tc.n)
			if tc.refusal != "" {
				if s := sentence(err); s != tc.refusal {
					t.Fatalf("refusal = %q, want %q", s, tc.refusal)
				}
				return
			}
			if o := got.Override(); err != nil || (o == nil) != (tc.want == nil) || (o != nil && *o != *tc.want) {
				t.Fatalf("Parse = %v, %v; want %v", o, err, tc.want)
			}
		})
	}
}

func TestRateWords(t *testing.T) {
	for n, want := range map[int64]string{0: "0 KB/s", 512 << 10: "512 KB/s", 1536 << 10: "1536 KB/s", 2 << 20: "2 MB/s"} {
		if got := setting.RateWords(n); got != want {
			t.Errorf("RateWords(%d) = %q, want %q", n, got, want)
		}
	}
}

var listen = setting.Address{Key: "torrent.listen", Label: "Listen address", Default: netip.MustParseAddrPort("0.0.0.0:6881")}

func TestAddressParse(t *testing.T) {
	for _, tc := range []struct {
		text    *string
		want    string
		refusal string
	}{
		{nil, "", ""},
		{ptr("192.168.1.20:51413"), "192.168.1.20:51413", ""},
		{ptr("[::]:6881"), "[::]:6881", ""},
		{ptr("seedbox:6881"), "", "An address and port, such as 0.0.0.0:6881."},
		{ptr("0.0.0.0"), "", "An address and port, such as 0.0.0.0:6881."},
		{ptr("0.0.0.0:0"), "", "A port between 1 and 65535."},
	} {
		got, err := listen.Parse(tc.text)
		if tc.refusal != "" {
			if s := sentence(err); s != tc.refusal {
				t.Errorf("Parse(%s) refusal = %q, want %q", *tc.text, s, tc.refusal)
			}
			continue
		}
		o := got.Override()
		if err != nil || (o == nil) != (tc.want == "") || (o != nil && o.String() != tc.want) {
			t.Errorf("Parse = %v, %v; want %q", o, err, tc.want)
		}
	}
	if got := listen.Value(nil); got.String() != "0.0.0.0:6881" {
		t.Errorf("Value(nil) = %s, want the default", got)
	}
}

var downloads = setting.Folder{Key: "torrent.downloads", Label: "Downloads folder", Default: "/var/lib/slipstream/downloads", Purpose: setting.FolderDownloads}

func TestFolderParse(t *testing.T) {
	if got, err := downloads.Parse(ptr(`D:\Torrents`)); err != nil || *got.Override() != `D:\Torrents` {
		t.Errorf("Parse(D:\\Torrents) = %v, %v", got.Override(), err)
	}
	if _, err := downloads.Parse(ptr("  ")); sentence(err) != "Choose a folder." {
		t.Errorf("Parse(blank) = %v, want Choose a folder.", err)
	}
	if got, err := downloads.Parse(nil); err != nil || got.Override() != nil {
		t.Errorf("Parse(nil) = %v, %v, want a reset", got.Override(), err)
	}
}

var warnBelow = setting.Space{
	Key: "health.storageWarnBelow", Label: "Warn when free space is below",
	Default: setting.Amount{N: 100 * setting.GB, Unit: setting.BytesUnit}, DefaultFrom: "Storage levels",
}

func TestSpaceParse(t *testing.T) {
	for _, tc := range []struct {
		name    string
		amount  *setting.Amount
		refusal string
	}{
		{"null resets", nil, ""},
		{"no bytes", &setting.Amount{N: 0, Unit: setting.BytesUnit}, ""},
		{"1 PB", &setting.Amount{N: 1024 * setting.TB, Unit: setting.BytesUnit}, ""},
		{"a byte over 1 PB", &setting.Amount{N: 1024*setting.TB + 1, Unit: setting.BytesUnit}, "Between 0 GB and 1024 TB."},
		{"negative bytes", &setting.Amount{N: -1, Unit: setting.BytesUnit}, "Between 0 GB and 1024 TB."},
		{"no percent", &setting.Amount{N: 0, Unit: setting.PercentUnit}, ""},
		{"all of it", &setting.Amount{N: 100, Unit: setting.PercentUnit}, ""},
		{"over 100%", &setting.Amount{N: 101, Unit: setting.PercentUnit}, "Between 0 and 100%."},
		{"negative percent", &setting.Amount{N: -1, Unit: setting.PercentUnit}, "Between 0 and 100%."},
		{"no unit", &setting.Amount{N: 5}, "Choose bytes or percent."},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := warnBelow.Parse(tc.amount)
			if tc.refusal != "" {
				if s := sentence(err); s != tc.refusal {
					t.Fatalf("refusal = %q, want %q", s, tc.refusal)
				}
				return
			}
			if o := got.Override(); err != nil || (o == nil) != (tc.amount == nil) || (o != nil && *o != *tc.amount) {
				t.Fatalf("Parse = %v, %v; want %v", o, err, tc.amount)
			}
		})
	}
	if got := warnBelow.Value(nil); got != warnBelow.Default {
		t.Fatalf("Value(nil) = %+v, want the default", got)
	}
}

func TestAmountWords(t *testing.T) {
	for _, tc := range []struct {
		amount setting.Amount
		want   string
	}{
		{setting.Amount{N: 0, Unit: setting.BytesUnit}, "0 GB"},
		{setting.Amount{N: 100 * setting.GB, Unit: setting.BytesUnit}, "100 GB"},
		{setting.Amount{N: setting.GB / 2, Unit: setting.BytesUnit}, "0.5 GB"},
		{setting.Amount{N: 40 * setting.MB, Unit: setting.BytesUnit}, "40 MB"},
		{setting.Amount{N: 1023 * setting.GB, Unit: setting.BytesUnit}, "1023 GB"},
		{setting.Amount{N: 1536 * setting.GB, Unit: setting.BytesUnit}, "1.5 TB"},
		{setting.Amount{N: 2 * setting.TB, Unit: setting.BytesUnit}, "2 TB"},
		{setting.Amount{N: 10, Unit: setting.PercentUnit}, "10%"},
	} {
		if got := tc.amount.Words(); got != tc.want {
			t.Errorf("Words(%+v) = %q, want %q", tc.amount, got, tc.want)
		}
	}
}

func stored[T any](parse func(json.RawMessage) (setting.Stored[T], error)) func(json.RawMessage) (string, error) {
	return func(raw json.RawMessage) (string, error) {
		s, err := parse(raw)
		if o := s.Override(); o != nil {
			return fmt.Sprint(*o), err
		}
		return "reset", err
	}
}
