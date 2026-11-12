package imports

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/imports/internal/db"
	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
)

var (
	// StopShare is the guard: a comparison that would find more than this
	// share of a root's files gone marks none. 100 never stops on the
	// share.
	StopShare = setting.Count{
		Key: "library.scan.stopShare", Label: "Stop a scan when this share of a root's files is gone",
		Default: 25, Min: 1, Max: 100, Unit: setting.Noun{One: "%", Many: "%"},
	}
	// Probes is how many files a Node probes at once, for a scan and for
	// an Import alike.
	Probes = setting.Count{
		Key: "library.scan.probes", Label: "Files each Node reads media info from at once",
		Default: 2, Min: 1, Max: 8, Unit: setting.Noun{One: "file", Many: "files"},
	}
	VideoExtensions = setting.List{
		Key: "library.importing.videoExtensions", Label: "Video file extensions",
		Default: media.DefaultExtensions, Min: 1, Item: setting.Extension,
	}
	SmallestVideo = setting.Size{
		Key: "library.importing.smallestVideo", Label: "Smallest video file",
		Default: 50 * setting.MB, Min: 0, Max: 10 * setting.GB,
	}
	SampleShorter = setting.Count{
		Key: "library.importing.sampleShorter", Label: "Sample when shorter than",
		Default: 25, Min: 0, Max: 90, Unit: setting.Noun{One: "% of its runtime", Many: "% of its runtime"}, Zero: "Off",
	}
)

// Settings is Library scan's settings an Admin sets in Settings > Library
// scan: the stop share and how many files a Node probes at once.
func (im *Imports) Settings() setting.Panel { return panel{im} }

type panel struct{ im *Imports }

func (p panel) Fields(ctx context.Context) ([]setting.Field, error) {
	s, err := db.New(p.im.cfg.Library).GetSettings(ctx)
	if err != nil {
		return nil, err
	}
	return []setting.Field{StopShare.Field(s.StopShare), Probes.Field(s.Probes)}, nil
}

// Apply stores the patch, all or none.
func (p panel) Apply(ctx context.Context, patch setting.Patch) error {
	for key := range patch {
		if key != StopShare.Key && key != Probes.Key {
			return &setting.UnknownKeyError{Key: key}
		}
	}
	return p.im.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		q := db.New(tx)
		s, err := q.GetSettings(ctx)
		if err != nil {
			return err
		}
		for _, c := range []struct {
			d  setting.Count
			at **int64
		}{{StopShare, &s.StopShare}, {Probes, &s.Probes}} {
			raw, ok := patch[c.d.Key]
			if !ok {
				continue
			}
			v, err := c.d.ParseJSON(raw)
			if err != nil {
				return err
			}
			*c.at = v.Override()
		}
		return q.SetSettings(ctx, db.SetSettingsParams{StopShare: s.StopShare, Probes: s.Probes})
	})
}

// Rules is what an Import reads of the Importing settings: the video
// extensions, the smallest video file in bytes, and the runtime share in
// percent under which a video is a sample, 0 for off.
type Rules struct {
	Videos   media.Videos
	Smallest int64
	Sample   int
}

// RulesIn reads the Importing settings in r.
func RulesIn(ctx context.Context, r storage.Reader[file]) (Rules, error) {
	s, err := db.New(r).GetSettings(ctx)
	if err != nil {
		return Rules{}, err
	}
	exts, err := extensionsOf(s.VideoExtensions)
	if err != nil {
		return Rules{}, err
	}
	return Rules{
		Videos:   media.NewVideos(VideoExtensions.Value(exts)...),
		Smallest: SmallestVideo.Value(s.SmallestVideo),
		Sample:   int(SampleShorter.Value(s.SampleShorter)),
	}, nil
}

// Videos is the Admin's video extensions, which every walk, move and
// survey sends its Node.
func Videos(ctx context.Context, r storage.Reader[file]) (media.Videos, error) {
	rules, err := RulesIn(ctx, r)
	return rules.Videos, err
}

func extensionsOf(stored *string) (*[]string, error) {
	exts, err := fromJSON[[]string](stored)
	if err != nil {
		return nil, fmt.Errorf("the video file extensions: %w", err)
	}
	return exts, nil
}

// Importing is the settings an Admin sets in Settings > Importing: the
// video extensions, the smallest video file and the runtime sample rule.
func (im *Imports) Importing() setting.Panel { return importingPanel{im} }

type importingPanel struct{ im *Imports }

func (p importingPanel) Fields(ctx context.Context) ([]setting.Field, error) {
	s, err := db.New(p.im.cfg.Library).GetSettings(ctx)
	if err != nil {
		return nil, err
	}
	exts, err := extensionsOf(s.VideoExtensions)
	if err != nil {
		return nil, err
	}
	return []setting.Field{VideoExtensions.Field(exts), SmallestVideo.Field(s.SmallestVideo), SampleShorter.Field(s.SampleShorter)}, nil
}

// Apply stores the patch, all or none.
func (p importingPanel) Apply(ctx context.Context, patch setting.Patch) error {
	for key := range patch {
		if key != VideoExtensions.Key && key != SmallestVideo.Key && key != SampleShorter.Key {
			return &setting.UnknownKeyError{Key: key}
		}
	}
	return p.im.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		q := db.New(tx)
		s, err := q.GetSettings(ctx)
		if err != nil {
			return err
		}
		if err := patchExtensions(patch, &s.VideoExtensions); err != nil {
			return err
		}
		for _, c := range []struct {
			key   string
			parse func(json.RawMessage) (setting.Stored[int64], error)
			at    **int64
		}{{SmallestVideo.Key, SmallestVideo.ParseJSON, &s.SmallestVideo}, {SampleShorter.Key, SampleShorter.ParseJSON, &s.SampleShorter}} {
			raw, ok := patch[c.key]
			if !ok {
				continue
			}
			v, err := c.parse(raw)
			if err != nil {
				return err
			}
			*c.at = v.Override()
		}
		return q.SetImportingSettings(ctx, db.SetImportingSettingsParams{
			VideoExtensions: s.VideoExtensions, SmallestVideo: s.SmallestVideo, SampleShorter: s.SampleShorter,
		})
	})
}

func patchExtensions(patch setting.Patch, at **string) error {
	raw, ok := patch[VideoExtensions.Key]
	if !ok {
		return nil
	}
	v, err := VideoExtensions.ParseJSON(raw)
	if err != nil {
		return err
	}
	*at, err = jsonOf(v.Override())
	return err
}

// ExtensionImpact counts the library files and the unfinished Imports'
// files that count as video now and wouldn't with exts.
func (im *Imports) ExtensionImpact(ctx context.Context, _ identity.Admin, exts []string) (library, pending int64, err error) {
	next := media.NewVideos(exts...)
	err = im.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		now, err := Videos(ctx, r)
		if err != nil {
			return err
		}
		stops := func(path string) bool { return now.Video(path) && !next.Video(path) }
		if library, err = files.CountPaths(ctx, r, stops); err != nil {
			return err
		}
		paths, err := db.New(r).ListUnfinishedPaths(ctx)
		for _, p := range paths {
			if stops(p) {
				pending++
			}
		}
		return err
	})
	return library, pending, err
}
