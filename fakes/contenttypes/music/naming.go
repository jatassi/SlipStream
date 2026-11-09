package music

import (
	"context"

	"github.com/jatassi/slipstream/fakes/contenttypes/music/internal/db"
	"github.com/jatassi/slipstream/internal/naming"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

var declaration = naming.Declaration{
	Tokens: []naming.Token{
		{Name: "Artist Name", Kind: naming.KindTitle, Scope: naming.WorkScope, Example: "The Lanterns", About: "The artist's name."},
		{Name: "Disc", Kind: naming.KindNumber, Scope: naming.ItemScope, Seq: naming.SeqOuter, Example: "1", About: "The disc the track is on."},
		{Name: "Track", Kind: naming.KindNumber, Scope: naming.ItemScope, Seq: naming.SeqInner, Example: "2", About: "The track's number on its disc."},
		{Name: "Track Title", Kind: naming.KindTitle, Scope: naming.ItemScope, Example: "Paper Boats", About: "The track's title."},
	},
	Folder: "{Artist Name}",
	Files: []naming.Variant{
		{Key: "standard", Label: "Track file", File: "{Artist Name} - {Disc}-{Track:00} - {Track Title} {Edition} {Target}", Anchor: naming.BlockAnchor()},
	},
	Samples: []naming.Sample{
		{Label: "A track", Work: lanterns, Units: []naming.Unit{onDisc(1, 2, "Paper Boats")}},
		{Label: "Two tracks", Work: lanterns, Units: []naming.Unit{onDisc(1, 1, "Harbour"), onDisc(1, 2, "Paper Boats")}},
		{Label: "Nothing known", Missing: true, Work: lanterns, Units: []naming.Unit{{Item: naming.ItemValues{Set: naming.Set{"Disc": naming.Number(1), "Track": naming.Number(1)}}}}},
	},
}

var lanterns = naming.Set{"Artist Name": naming.Text("The Lanterns")}

func onDisc(disc, number int64, title string) naming.Unit {
	return naming.Unit{Item: naming.ItemValues{Set: naming.Set{"Disc": naming.Number(disc), "Track": naming.Number(number), "Track Title": naming.Text(title)}}}
}

func (*Music) Naming() naming.Declaration { return declaration }

// NamingValues reads every artist of ws, its tracks and their Items.
func (*Music) NamingValues(ctx context.Context, r storage.Reader[file], ws works.Among[works.WorkID]) (map[works.WorkID]naming.Values, error) {
	q := db.New(r)
	trees, err := works.OutlinesIn(ctx, r, ws)
	if err != nil {
		return nil, err
	}
	tracks, err := works.Read(ws, func() ([]db.MusicTrack, error) { return q.ListEveryTrack(ctx) }, func(ids []works.WorkID) ([]db.MusicTrack, error) {
		return q.ListTracksOf(ctx, ids)
	})
	if err != nil {
		return nil, err
	}
	out := make(map[works.WorkID]naming.Values, len(trees))
	labels := map[works.ItemID]string{}
	for _, t := range trees {
		out[t.ID] = naming.Values{Work: naming.Set{"Artist Name": naming.Text(t.Label)}, Items: map[works.ItemID]naming.ItemValues{}}
		for _, it := range t.Items {
			labels[it.ID] = it.Label
		}
	}
	for _, t := range tracks {
		out[t.WorkID].Items[t.ItemID] = naming.ItemValues{Set: naming.Set{
			"Disc": naming.Number(t.Disc), "Track": naming.Number(t.Number), "Track Title": naming.Text(labels[t.ItemID]),
		}}
	}
	return out, nil
}
