package mediaserverlink_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/mediaserverlink"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	msv1 "github.com/jatassi/slipstream/internal/pb/slipstream/mediaserver/v1"
	"github.com/jatassi/slipstream/internal/storage"
)

var at = time.Date(2026, 11, 14, 14, 2, 3, 456_000_000, time.UTC)

func path(t *testing.T, s string) link.ServerPath {
	t.Helper()
	p, err := link.ParsePath(s)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func server() link.ServerRef {
	return link.ServerRef{ID: link.ServerID{UUID: storage.NewUUID()}, Kind: "jellyfin", Identity: "f4c1", SettingsVersion: 3}
}

func change() link.ChangeID { return link.ChangeID{UUID: storage.NewUUID()} }

func snapshot() *link.Snapshot {
	return &link.Snapshot{
		Item: link.ItemKey{Item: "a1", Top: "s1"},
		Users: []link.UserData{{User: "Sam", Played: true, PlayCount: 2, Position: 754 * time.Second,
			Favorite: true, LastPlayed: at, Rating: 8.5}},
		Collections: []string{"Noir"},
	}
}

func works(t *testing.T) []link.Work {
	return []link.Work{
		link.Refresh{Changes: []link.JobChange{
			{ID: change(), Kind: link.ChangeAdded, Library: "1", Path: path(t, `D:\Movies\Heat (1995)\Heat (1995).mkv`)},
			{ID: change(), Kind: link.ChangeSidecarRemoved, Library: "1", Path: path(t, "/media/movies/Ran/Ran.en.srt")},
		}},
		link.Carry{Changes: []link.JobChange{{ID: change(), Kind: link.ChangeMoved, Library: "2",
			Path: path(t, "/media/shows/Show/S01E01.mkv"), From: path(t, "/media/shows/Show/Show - 1x01.mkv"), Snapshot: snapshot()}}},
		link.Fresh{Library: "3", Folder: path(t, `\\nas\media\music`), Requested: at, Started: at.Add(time.Second)},
		link.Rescan{Libraries: []string{"1", "2"}, Changes: []link.ChangeID{change()}},
		link.Lookup{Paths: []link.ServerPath{path(t, "/media/movies/Heat.mkv")}},
		link.Index{Library: "1", Cursor: "500"},
		link.Libraries{},
	}
}

func TestAJobCrossesTheWireUnchanged(t *testing.T) {
	s := server()
	for _, w := range works(t) {
		j := link.Job{Server: s, Work: w, Step: link.StepOldRemoved, At: at}
		got, err := mediaserverlink.JobOf(s, mediaserverlink.JobOnWire(j))
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, j) {
			t.Errorf("%T crossed as %+v, want %+v", w, got, j)
		}
	}
}

func TestAnAnswerCrossesTheWireUnchanged(t *testing.T) {
	s := server()
	a := link.Advanced{
		Job:  link.Job{Server: s, Work: works(t)[1], Step: link.StepRestored, At: at},
		Done: true,
		Wait: at.Add(time.Minute),
		Outcomes: []link.Outcome{
			{Change: change(), Result: link.ResultAccepted, Lost: []link.Loss{link.LostPlayCount, link.LostLastPlayed}},
			{Change: change(), Result: link.ResultConverted},
		},
		Learned:   []link.IndexEntry{{Path: path(t, `D:\Movies\Heat.mkv`), Library: "1", Item: link.ItemKey{Item: "7", Top: "7"}, Gone: true}},
		Mapping:   &link.MappingFact{Started: at, Scanned: true},
		Libraries: []link.Library{{Key: "1", Name: "Movies", Folders: []link.ServerPath{path(t, `D:\Movies`), path(t, `E:\Movies`)}}},
	}
	got, err := mediaserverlink.AdvancedOf(s, mediaserverlink.AdvancedOnWire(a))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, a) {
		t.Errorf("answer crossed as %+v, want %+v", got, a)
	}
}

func TestACheckCrossesTheWireUnchanged(t *testing.T) {
	c := link.Checked{
		Identity:     link.Identity{ID: "f4c1", Name: "Den Jellyfin", Version: link.MustVersion("12.2.0")},
		MonitorDelay: 60 * time.Second,
		Libraries:    []link.Library{{Key: "a", Name: "Shows", Folders: []link.ServerPath{path(t, "/media/shows")}}},
	}
	m, err := mediaserverlink.CheckedOnWire(c, nil)
	if err != nil {
		t.Fatal(err)
	}
	got, err := mediaserverlink.CheckedOf(m)
	if err != nil || !reflect.DeepEqual(got, c) {
		t.Errorf("check crossed as %+v, %v, want %+v", got, err, c)
	}
	refused := &link.ServerError{Problem: link.Problem{Reason: link.ReasonRefused, Status: 401}}
	m, err = mediaserverlink.CheckedOnWire(link.Checked{}, refused)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := mediaserverlink.CheckedOf(m); !problemIs(err, refused) {
		t.Errorf("a refused check crossed as %v, want %v", err, refused)
	}
}

func TestACredentialCrossesTheWireUnchanged(t *testing.T) {
	c := link.Credential{Token: "token", APIKey: "key"}
	if got := mediaserverlink.CredentialOf(mediaserverlink.CredentialOnWire(c)); got != c {
		t.Errorf("credential crossed changed")
	}
}

// node plays the reaching Node's side of Advance over conn, as the holder's
// handler does.
type node struct {
	server link.ServerRef
	conn   func(link.Job) (link.Advanced, error)
}

func (n node) Advance(_ context.Context, req *msv1.AdvanceRequest) (*msv1.AdvanceResponse, error) {
	j, err := mediaserverlink.JobOf(n.server, req.GetJob())
	if err != nil {
		return nil, err
	}
	a, err := n.conn(j)
	if err != nil {
		p, err := mediaserverlink.ProblemOnWire(err)
		return &msv1.AdvanceResponse{Problem: p}, err
	}
	return mediaserverlink.AdvancedOnWire(a), nil
}

func TestRemoteAnswersAsTheReachingNodesConn(t *testing.T) {
	s := server()
	j := link.Job{Server: s, Work: works(t)[0], Step: link.StepIdle, At: at}
	want := link.Advanced{Job: link.Job{Server: s, Work: j.Work, Step: link.StepReported, At: at}, Wait: at.Add(time.Minute)}
	var got link.Job
	port := mediaserverlink.Remote(node{s, func(j link.Job) (link.Advanced, error) { got = j; return want, nil }})
	a, err := port.Advance(t.Context(), j)
	if err != nil || !reflect.DeepEqual(a, want) || !reflect.DeepEqual(got, j) {
		t.Fatalf("Remote ran %+v and answered %+v, %v; want %+v answered %+v", got, a, err, j, want)
	}
}

func TestRemoteKeepsAServersFailureApartFromTheLinks(t *testing.T) {
	s := server()
	failed := &link.ServerError{Problem: link.Problem{Reason: link.ReasonOtherServer, Found: "9a0b"}}
	_, err := mediaserverlink.Remote(node{s, func(link.Job) (link.Advanced, error) { return link.Advanced{}, failed }}).
		Advance(t.Context(), link.Job{Server: s, Work: link.Libraries{}, Step: link.StepNone})
	if !problemIs(err, failed) {
		t.Errorf("a server's failure crossed as %v, want %v", err, failed)
	}
	down := context.DeadlineExceeded
	_, err = mediaserverlink.Remote(node{s, func(link.Job) (link.Advanced, error) { return link.Advanced{}, down }}).
		Advance(t.Context(), link.Job{Server: s, Work: link.Libraries{}, Step: link.StepNone})
	if _, ok := link.ServerProblem(err); ok || !errors.Is(err, down) {
		t.Errorf("a link failure crossed as %v, want the deadline", err)
	}
}

func problemIs(err error, want *link.ServerError) bool {
	p, ok := link.ServerProblem(err)
	return ok && p == want.Problem
}
