package events_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/events"
)

func create(t *testing.T, observe func(events.Record) []events.Record) (log *events.Log, dir string) {
	t.Helper()
	dir = t.TempDir()
	log, err := events.Create(dir, observe)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = log.Close() })
	return log, dir
}

func none(events.Record) []events.Record { return nil }

func appendRec(t *testing.T, log *events.Log, kind, data string) events.Record {
	t.Helper()
	rec, err := log.Append(events.Record{Kind: kind, Node: events.World, Data: json.RawMessage(data)})
	if err != nil {
		t.Fatal(err)
	}
	return rec
}

func read(t *testing.T, dir string) []events.Record {
	t.Helper()
	recs, err := events.Read(dir)
	if err != nil {
		t.Fatal(err)
	}
	return recs
}

func kinds(recs []events.Record) []string {
	out := make([]string, len(recs))
	for i, r := range recs {
		out[i] = fmt.Sprintf("%d %s", r.Seq, r.Kind)
	}
	return out
}

func TestTheLogNumbersEveryRecordOnceUnderConcurrentAppends(t *testing.T) {
	log, dir := create(t, none)
	const writers, each = 8, 50
	var wg sync.WaitGroup
	for w := range writers {
		wg.Go(func() {
			for i := range each {
				if _, err := log.Append(events.Record{Kind: "test.append", Data: json.RawMessage(fmt.Sprintf(`{"writer":%d,"i":%d}`, w, i))}); err != nil {
					t.Error(err)
				}
			}
		})
	}
	wg.Wait()
	recs := read(t, dir)
	if len(recs) != writers*each {
		t.Fatalf("read %d records, want %d", len(recs), writers*each)
	}
	next := make([]int, writers)
	for i, r := range recs {
		var d struct{ Writer, I int }
		if err := json.Unmarshal(r.Data, &d); err != nil {
			t.Fatal(err)
		}
		if r.Seq != uint64(i+1) || d.I != next[d.Writer] {
			t.Fatalf("record %d is seq %d, writer %d's append %d; want seq %d and append %d", i, r.Seq, d.Writer, d.I, i+1, next[d.Writer])
		}
		next[d.Writer]++
	}
}

func TestWhatARecordCausesFollowsIt(t *testing.T) {
	log, dir := create(t, func(r events.Record) []events.Record {
		if r.Kind == "fault.set" {
			return []events.Record{{Kind: "fault.armed"}}
		}
		return nil
	})
	appendRec(t, log, "run.ready", `{}`)
	set, err := log.Apply(func() (events.Record, []events.Record, error) {
		return events.Record{Kind: "fault.set"}, []events.Record{{Kind: "fault.fired"}, {Kind: "fault.fired"}}, nil
	})
	if err != nil || set.Seq != 2 {
		t.Fatalf("Apply = %+v, %v; want seq 2", set, err)
	}
	if _, err := log.Apply(func() (events.Record, []events.Record, error) { return events.Record{}, nil, errors.New("refused") }); err == nil || err.Error() != "refused" {
		t.Fatalf("a refused change = %v, want refused", err)
	}
	appendRec(t, log, "step.start", `{"n":1}`)
	want := []string{"1 run.ready", "2 fault.set", "3 fault.fired", "4 fault.fired", "5 fault.armed", "6 step.start"}
	if got := kinds(read(t, dir)); !slices.Equal(got, want) {
		t.Fatalf("the log holds %q, want %q", got, want)
	}
}

func TestReadSkipsALineStillBeingWritten(t *testing.T) {
	dir := t.TempDir()
	data := `{"seq":1,"at":"2026-10-14T00:00:00Z","kind":"run.launched","node":"world"}` + "\n" + `{"seq":2,"at":"2026-10-14T0`
	if err := os.WriteFile(filepath.Join(dir, events.File), []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := kinds(read(t, dir)); !slices.Equal(got, []string{"1 run.launched"}) {
		t.Fatalf("read %q, want only the whole line", got)
	}
	if err := os.WriteFile(filepath.Join(dir, events.File), []byte("{\"seq\":1}\nnot json\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := events.Read(dir); err == nil || err.Error() != "events.jsonl:2: invalid character 'o' in literal null (expecting 'u')" {
		t.Fatalf("Read of a corrupt line = %v", err)
	}
}

var (
	started = events.Query{Kind: "tracker.*", Match: map[string]string{"event": "started"}}
	mail    = events.Query{Kind: "mail.*", Node: "home", Match: map[string]string{"to": "a@example.org"}, After: 1}
)

func announces(t *testing.T) (log *events.Log, dir string) {
	t.Helper()
	log, dir = create(t, none)
	appendRec(t, log, "tracker.announce", `{"event":"started"}`)
	appendRec(t, log, "tracker.announce", `{"event":"completed"}`)
	return log, dir
}

func TestSelectAndWaitOnALiveLog(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		log, dir := announces(t)
		if got, err := events.Select(dir, started); err != nil || len(got) != 1 || got[0].Seq != 1 {
			t.Fatalf("Select = %+v, %v; want seq 1", got, err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		if got, err := events.Wait(ctx, dir, started); err != nil || got.Seq != 1 {
			t.Fatalf("Wait for a logged record = %+v, %v; want seq 1", got, err)
		}

		go func() {
			time.Sleep(10*time.Second + 10*time.Millisecond)
			appendRec(t, log, "tracker.announce", `{"event":"started"}`)
		}()
		start := time.Now()
		later := started
		later.After = 2
		if got, err := events.Wait(ctx, dir, later); err != nil || got.Seq != 3 || time.Since(start) != 10*time.Second+25*time.Millisecond {
			t.Fatalf("Wait for a later record = %+v, %v after %v; want seq 3 on the tick after it was appended", got, err, time.Since(start))
		}

		short, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		if _, err := events.Wait(short, dir, mail); err == nil || err.Error() != "no record matches kind mail.*, node home, to=a@example.org, after seq 1: context deadline exceeded" {
			t.Fatalf("Wait past its deadline = %v", err)
		}
	})
}

func TestSelectAndWaitOnAStoppedLog(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		log, dir := announces(t)
		appendRec(t, log, events.RunStopped, `{"clean":true}`)
		if err := log.Close(); err != nil {
			t.Fatal(err)
		}
		if _, err := log.Append(events.Record{Kind: "late"}); err == nil {
			t.Fatal("Append after Close succeeded")
		}
		if got, err := events.Select(dir, started); err != nil || len(got) != 1 || got[0].Seq != 1 {
			t.Fatalf("Select = %+v, %v; want seq 1", got, err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		if got, err := events.Wait(ctx, dir, started); err != nil || got.Seq != 1 {
			t.Fatalf("Wait for a logged record = %+v, %v; want seq 1", got, err)
		}
		start := time.Now()
		if _, err := events.Wait(ctx, dir, mail); err == nil || err.Error() != "no record matches kind mail.*, node home, to=a@example.org, after seq 1: the run stopped at seq 3" || time.Since(start) != 0 {
			t.Fatalf("Wait on a stopped run = %v after %v; want the stop at once", err, time.Since(start))
		}
	})
}
