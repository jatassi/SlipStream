package trackers_test

import (
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/health"
	"github.com/jatassi/slipstream/internal/indexers"
)

func (l *library) health() health.Answer {
	l.t.Helper()
	a, err := l.x.HealthSource().Answer(l.t.Context())
	if err != nil {
		l.t.Fatal(err)
	}
	return a
}

func TestHealthWarnsOnceATorrentThatOwesHasNotSeededForTheWarnDelay(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := newLibrary(t)
		l.exec(`UPDATE trackers_settings SET warn_after_seconds = 3600`)
		seedpool := l.tracker("seedpool", indexers.Private)
		start := time.Now()
		for _, name := range []string{"Orchard", "Harbor"} {
			id := l.torrent(name, &seedpool, true, start, 0, 4*gib)
			l.grabOf(seedpool, id, nil, ptr(3*day))
			l.exec(`UPDATE downloadclients_torrent SET wanted = 'paused' WHERE id = ?`, id)
		}
		l.runGoals()
		if a := l.health(); len(a.Items) != 0 || !a.Next.Equal(start.Add(time.Hour)) {
			t.Fatalf("as Orchard and Harbor stop seeding Health holds %+v next at %v, want nothing until an hour on", a.Items, a.Next)
		}

		time.Sleep(time.Hour)
		a := l.health()
		want := "2 torrents on seedpool stopped seeding before seedpool's rule was met. Resume or retry them before seedpool counts a hit and run."
		if len(a.Items) != 1 || a.Items[0].Level != health.Warn || a.Items[0].Text != want {
			t.Fatalf("an hour on Health holds %+v, want one warning: %s", a.Items, want)
		}
		if act := a.Items[0].Action; act.Kind() != health.ActionLink || act.Href() != "/settings/trackers/"+seedpool.String()+"?tab=problems" ||
			act.Label() != "Open seedpool's torrents" {
			t.Errorf("the warning's action links %s %q, want seedpool's Torrents on Problems", act.Href(), act.Label())
		}

		l.offline = true
		if a := l.health(); len(a.Items) != 0 || len(a.Unsure) != 1 {
			t.Errorf("with their Node offline Health holds %+v and is unsure of %+v, want no warning and seedpool unsure", a.Items, a.Unsure)
		}
		l.offline = false

		l.exec(`UPDATE downloadclients_torrent SET wanted = 'started'`)
		l.runGoals()
		if a := l.health(); len(a.Items) != 0 || len(a.Unsure) != 0 || !a.Next.IsZero() {
			t.Errorf("once both seed again Health holds %+v, unsure of %+v, next at %v; want nothing", a.Items, a.Unsure, a.Next)
		}
	})
}

func TestHealthRaisesNothingForATorrentWhoseNodeIsOffline(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := newLibrary(t)
		l.exec(`UPDATE trackers_settings SET warn_after_seconds = 3600`)
		seedpool := l.tracker("seedpool", indexers.Private)
		orchard := l.torrent("Orchard", &seedpool, true, time.Now(), 0, 4*gib)
		l.grabOf(seedpool, orchard, nil, ptr(3*day))
		l.offline = true
		l.runGoals()
		time.Sleep(2 * time.Hour)
		a := l.health()
		if len(a.Items) != 0 || len(a.Unsure) != 1 || a.Unsure[0].Text != "Whether seedpool's torrents on home are seeding is unknown until it's back online." {
			t.Errorf("with Orchard's Node offline Health holds %+v and is unsure of %+v, want no warning and seedpool unsure", a.Items, a.Unsure)
		}
	})
}
