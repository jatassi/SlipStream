package node

import (
	"context"
	"maps"
	"slices"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/mediaservers"
	"github.com/jatassi/slipstream/internal/mediaservers/playback"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/subtitles"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// playbackReads are the library reads of the Playback report, over the
// Targets once they open.
type playbackReads struct{ targets **targets.Targets }

func (p playbackReads) played(ctx context.Context, r storage.Reader[storage.Library], id files.ID) (mediaservers.PlayedFile, error) {
	f, err := files.PlayedOf(ctx, r, id)
	if err != nil {
		return mediaservers.PlayedFile{}, err
	}
	orig, err := p.originals(ctx, r, []works.WorkID{f.WorkID})
	if err != nil {
		return mediaservers.PlayedFile{}, err
	}
	all, err := targets.All(ctx, r)
	if err != nil {
		return mediaservers.PlayedFile{}, err
	}
	return mediaservers.PlayedFile{Work: f.WorkID, Item: f.Items[0], Label: files.LabelOf(f), Met: met(f, orig[f.WorkID], all), Media: f.Media}, nil
}

func (p playbackReads) siblings(ctx context.Context, r storage.Reader[storage.Library], items []works.ItemID) (playback.Siblings, error) {
	out := playback.Siblings{Files: map[works.ItemID][]playback.Sibling{}, Goals: map[works.WorkID]playback.Goal{}}
	fs, err := files.OfItems(ctx, r, items)
	if err != nil || len(fs) == 0 {
		return out, err
	}
	ids := make([]files.ID, len(fs))
	seen := map[works.WorkID]bool{}
	var ws []works.WorkID
	for i, f := range fs {
		ids[i] = f.ID
		if !seen[f.WorkID] {
			seen[f.WorkID] = true
			ws = append(ws, f.WorkID)
		}
	}
	sidecars, err := subtitles.SidecarsOf(ctx, r, ids)
	if err != nil {
		return out, err
	}
	goals, err := targets.GoalsOf(ctx, r, works.Only(ws))
	if err != nil {
		return out, err
	}
	originals, err := p.originals(ctx, r, ws)
	if err != nil {
		return out, err
	}
	all, err := targets.All(ctx, r)
	if err != nil {
		return out, err
	}
	for w, g := range goals {
		out.Goals[w] = goalOf(g)
	}
	asked := make(map[works.ItemID]bool, len(items))
	for _, it := range items {
		asked[it] = true
	}
	for _, f := range fs {
		sib := playback.Sibling{File: f.ID, Work: f.WorkID, Label: files.LabelOf(f), Kept: f.Kept(), Met: met(f, originals[f.WorkID], all), Media: f.Media}
		for _, s := range sidecars[f.ID] {
			sib.Sidecars = append(sib.Sidecars, playback.Sidecar{Language: string(s.Tail.Language.Base()), Codec: media.SubtitleCodecOfExt(s.Tail.Ext)})
		}
		for _, it := range f.Items {
			if asked[it] {
				out.Files[it] = append(out.Files[it], sib)
			}
		}
	}
	return out, nil
}

// originals is the original language of each of ws its Content Type
// knows, asking each type once.
func (p playbackReads) originals(ctx context.Context, r storage.Reader[storage.Library], ws []works.WorkID) (map[works.WorkID]release.Fact[release.Language], error) {
	summaries, err := works.SummariesOf(ctx, r, ws)
	if err != nil {
		return nil, err
	}
	byType := map[works.TypeID][]works.WorkID{}
	for _, w := range ws {
		typ := summaries[w].ContentType
		byType[typ] = append(byType[typ], w)
	}
	out := map[works.WorkID]release.Fact[release.Language]{}
	for _, typ := range slices.Sorted(maps.Keys(byType)) {
		of, err := (*p.targets).OriginalsIn(ctx, r, typ, works.Only(byType[typ]))
		if err != nil {
			return nil, err
		}
		maps.Copy(out, of)
	}
	return out, nil
}

func goalOf(g targets.Goals) playback.Goal {
	goal := playback.Goal{Profile: playback.TargetRef{ID: g.Profile.ID.String(), Name: g.Profile.Name}}
	for _, t := range g.Targets {
		goal.Targets = append(goal.Targets, t.ID)
	}
	return goal
}

// met is every Target of all that f meets, judged under its Work's
// original language as files' own scoring judges it.
func met(f files.File, original release.Fact[release.Language], all []targets.Target) []playback.TargetRef {
	facts := f.Facts()
	facts.Coverage.Original = original
	var out []playback.TargetRef
	for _, t := range all {
		if targets.Judge(t, targets.Scored{Facts: facts}, nil).Meets() {
			out = append(out, playback.TargetRef{ID: t.ID.String(), Name: t.Name})
		}
	}
	return out
}
