package indexers_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/magnet"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/searching"
)

const link = "magnet:?xt=urn:btih:0123456789abcdef0123456789abcdef01234567&dn=Night+Harbor&tr=https%3A%2F%2Ftracker.example.test%2Fannounce"

func TestFetchReadsAMagnetLinkWithoutANode(t *testing.T) {
	calls := 0
	x := indexers.New(indexers.Config{Searching: func(role.NodeID) searching.Remote {
		calls++
		return nil
	}})
	got, err := x.Fetch(t.Context(), indexers.Ticket{}, searching.LinkOf(link))
	want := indexers.Magnet{Link: magnet.Link{
		InfoHash: must(bencode.ParseInfoHash("0123456789abcdef0123456789abcdef01234567")),
		Name:     "Night Harbor", Trackers: []string{"https://tracker.example.test/announce"},
	}}
	if err != nil || !reflect.DeepEqual(got, want) || calls != 0 {
		t.Fatalf("Fetch of a magnet link = %+v, %v after %d Node calls; want %+v after none", got, err, calls, want)
	}
	if _, err := x.Fetch(t.Context(), indexers.Ticket{}, searching.LinkOf("magnet:?xt=urn:btmh:1220aa")); !errors.Is(err, bencode.ErrV2Only) {
		t.Errorf("Fetch of a btmh-only magnet link = %v, want ErrV2Only", err)
	}
	if _, err := x.Fetch(t.Context(), indexers.Ticket{}, searching.LinkOf("magnet:?dn=a")); !errors.As(err, new(*magnet.InvalidError)) {
		t.Errorf("Fetch of a magnet link with no info hash = %v, want *magnet.InvalidError", err)
	}
}

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}
