package indexers

import (
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/searching"
)

type named struct{ roster }

func (named) Name(n role.NodeID) string {
	if n == seed {
		return "seedbox"
	}
	return "den"
}

func TestStatusWords(t *testing.T) {
	x := New(Config{Roster: named{roster{holders: []role.NodeID{home, seed}}}, Zone: time.UTC})
	at := func(h, m int) time.Time { return time.Date(2026, 3, 1, h, m, 0, 0, time.UTC) }
	stop := func(c Cause, text string) View {
		return View{
			Status: Status{State: Stopped, Cause: c, Text: text, Node: seed, Success: t0},
			Waits:  &Outcome{Kind: IsStopped, Cause: c, Text: text},
		}
	}
	for _, tc := range []struct {
		name          string
		v             View
		sentence, fix string
	}{
		{"working", View{Status: Status{State: OK, Success: t0}}, "TorrentLeech is working.", ""},
		{"never searched", View{Status: Status{State: OK}}, "SlipStream hasn't searched TorrentLeech yet.", ""},
		{"off", View{Status: Status{State: Stopped}}, "TorrentLeech is off.", ""},
		{
			"failing past its wait", View{Status: Status{State: Failing, Success: t0}},
			"The last request to TorrentLeech failed. The next search tries it again.", "",
		},
		{
			"failing", View{Waits: &Outcome{Kind: IsFailing, Until: at(14, 20)}},
			"TorrentLeech isn't responding. SlipStream retries at 14:20.", "",
		},
		{"limited", View{Waits: &Outcome{Kind: IsLimited, Until: at(14, 5)}}, "TorrentLeech is limiting requests until 14:05.", ""},
		{
			"at its search limit", View{Waits: &Outcome{Kind: LimitReached, Until: at(14, 5)}},
			"TorrentLeech reached its search limit. SlipStream searches it again at 14:05.", "",
		},
		{
			"password", stop(searching.CredentialsRejected, ""),
			"Stopped because TorrentLeech rejected the username or password.", "Update the username or password with Edit. Saving tests them.",
		},
		{
			"captcha", stop(searching.CaptchaNeeded, ""),
			"Stopped because TorrentLeech needs a captcha to sign in again.", "Test the Indexer and type the code.",
		},
		{
			"challenge with no solver", stop(searching.ChallengeUnsolved, ""),
			"Stopped because TorrentLeech is behind a browser challenge and seedbox has no Challenge solver.",
			"Set a Challenge solver on seedbox, or paste a browser cookie with Edit.",
		},
		{
			"challenge the solver couldn't clear",
			View{Status: Status{State: Stopped, Node: seed, SolverTried: true}, Waits: &Outcome{Kind: IsStopped, Cause: searching.ChallengeUnsolved}},
			"Stopped because seedbox's Challenge solver couldn't clear TorrentLeech's challenge.", "Paste a browser cookie with Edit.",
		},
		{
			"cookie", stop(searching.CookieExpired, ""),
			"Stopped because TorrentLeech no longer accepts the browser cookie.", "Paste a new browser cookie with Edit.",
		},
		{
			"definition", stop(searching.DefinitionUnloadable, ""),
			"Stopped because TorrentLeech's definition doesn't load.", "Replace the custom definition, or wait for a catalog fix.",
		},
		{
			"account", stop(searching.AccountRefused, "Account suspended"),
			"Stopped because TorrentLeech refused the API key with 'Account suspended'.", "Check the account, then test the Indexer.",
		},
		{
			"request", stop(searching.RequestRejected, "Incorrect parameter"),
			"Stopped because TorrentLeech refused a search with 'Incorrect parameter'.", "Test the Indexer after the feed is fixed.",
		},
		{
			"waiting inside the grace", View{SignsIn: true, Waits: &Outcome{Kind: Waiting, Node: seed, Since: at(8, 0)}},
			"TorrentLeech is waiting for seedbox, which has been offline since 08:00.", "",
		},
		{
			"waiting on a lapsed home", View{SignsIn: true, Waits: &Outcome{Kind: Waiting, Node: seed, Since: at(8, 0), Lapsed: true}},
			"TorrentLeech is waiting for seedbox, which has been offline since 08:00.", "Allow failover to search from another Node.",
		},
		{
			"waiting for a home that can't take work", View{Waits: &Outcome{Kind: Waiting, Node: seed}},
			"TorrentLeech is waiting for seedbox.", "",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := tc.v
			v.Name, v.On = "TorrentLeech", tc.name != "off"
			sentence, fix := x.words(v)
			if sentence != tc.sentence || fix != tc.fix {
				t.Errorf("words = %q, %q, want %q, %q", sentence, fix, tc.sentence, tc.fix)
			}
		})
	}
}
