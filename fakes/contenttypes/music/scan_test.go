package music_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/jatassi/slipstream/internal/scanning"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

func TestPlaceReadsTrackNumbersOnTheAlbumTheFolderNames(t *testing.T) {
	m, lib, tree := added(t, catalog(t), lanterns)
	var place scanning.Placer
	if err := lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		place, err = m.Place(ctx, r, tree.ID)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	labels := map[works.ItemID]string{}
	for _, it := range tree.Items {
		labels[it.ID] = it.Label
	}
	tests := []struct {
		rel  string
		want any
	}{
		{"Harbor Lights/The Lanterns - 1-02 - Paper Boats.mkv", "Paper Boats"},
		{"Night Ferry/02 Signal Fires.mkv", "Signal Fires"},
		{"The Lanterns - 1-03 - Low Bridge.mkv", "Low Bridge"},
		{"The Lanterns - 1-02 - Paper Boats.mkv", scanning.Reason{Code: scanning.NoRule}},
		{"Harbor Lights/The Lanterns - 1-09.mkv", scanning.Reason{Code: scanning.NoSuchItem, Names: "1-09"}},
		{"Live Bootleg.mkv", scanning.Reason{Code: scanning.NoNumber}},
	}
	for _, tt := range tests {
		p := place(tt.rel)
		var got any = p.Reason()
		if items, ok := p.Items(); ok {
			got = labels[items[0]]
		}
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("Place(%q) = %v, want %v", tt.rel, got, tt.want)
		}
	}
}

func TestHintReadsAnArtistFolder(t *testing.T) {
	if got, want := library().Hint("The Lanterns"), (scanning.Hint{Title: "The Lanterns"}); !reflect.DeepEqual(got, want) {
		t.Fatalf("Hint = %+v, want %+v", got, want)
	}
}
