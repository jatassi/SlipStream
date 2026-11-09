package music

import (
	"context"
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/jatassi/slipstream/fakes/contenttypes/music/internal/db"
	"github.com/jatassi/slipstream/internal/scanning"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// Hint reads an artist folder's name.
func (*Music) Hint(name string) scanning.Hint { return scanning.ReadName(name) }

// Claims answers none: an artist has no year and no id a name carries.
func (*Music) Claims(context.Context, storage.Reader[file], works.Among[works.WorkID]) (map[works.WorkID]scanning.Claim, error) {
	return nil, nil
}

func (*Music) YearOf(works.Candidate) int { return 0 }

var (
	// discTrack is a disc and track number as the file template writes
	// them: "The Lanterns - 1-02 - Paper Boats".
	discTrack = regexp.MustCompile(`(?:^|\s)(\d{1,2})-(\d{1,3})(?:\s|$)`)
	// leading is a track number that starts a name, on disc 1: "02 Paper
	// Boats".
	leading = regexp.MustCompile(`^(\d{1,3})\b`)
)

type track struct {
	album        string
	disc, number int64
	item         works.ItemID
}

// Place reads a file's disc and track number, else its leading track
// number on disc 1, on the album its folder names, or on any album when
// the folder names none. A number on several albums places nothing.
func (*Music) Place(ctx context.Context, r storage.Reader[file], work works.WorkID) (scanning.Placer, error) {
	levels, err := works.LevelsOf(ctx, r, work)
	if err != nil {
		return nil, err
	}
	albums := make(map[works.LevelID]string, len(levels))
	named := make(map[string]bool, len(levels))
	for _, l := range levels {
		albums[l.ID] = scanning.Fold(l.Label)
		named[albums[l.ID]] = true
	}
	items, err := works.ItemsOf(ctx, r, works.Only([]works.WorkID{work}))
	if err != nil {
		return nil, err
	}
	on := make(map[works.ItemID]works.LevelID, len(items[work]))
	for _, it := range items[work] {
		on[it.ID] = it.LevelID
	}
	rows, err := db.New(r).ListTracks(ctx, work)
	if err != nil {
		return nil, err
	}
	tracks := make([]track, len(rows))
	for i, t := range rows {
		tracks[i] = track{albums[on[t.ItemID]], t.Disc, t.Number, t.ItemID}
	}
	return func(rel string) scanning.Placement {
		base := path.Base(rel)
		disc, number, names, ok := trackOf(strings.TrimSuffix(base, path.Ext(base)))
		if !ok {
			return scanning.Unplaced(scanning.Reason{Code: scanning.NoNumber})
		}
		album := scanning.Fold(path.Base(path.Dir(rel)))
		var hits []works.ItemID
		for _, t := range tracks {
			if t.disc == disc && t.number == number && (!named[album] || t.album == album) {
				hits = append(hits, t.item)
			}
		}
		switch len(hits) {
		case 0:
			return scanning.Unplaced(scanning.Reason{Code: scanning.NoSuchItem, Names: names})
		case 1:
			return scanning.Placed(hits[0])
		}
		return scanning.Unplaced(scanning.Reason{Code: scanning.NoRule})
	}, nil
}

func trackOf(name string) (disc, number int64, names string, ok bool) {
	if m := discTrack.FindStringSubmatch(name); m != nil {
		disc, _ = strconv.ParseInt(m[1], 10, 64)
		number, _ = strconv.ParseInt(m[2], 10, 64)
		return disc, number, m[1] + "-" + m[2], true
	}
	if m := leading.FindStringSubmatch(name); m != nil {
		number, _ = strconv.ParseInt(m[1], 10, 64)
		return 1, number, m[1], true
	}
	return 0, 0, "", false
}
