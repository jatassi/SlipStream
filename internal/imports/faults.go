package imports

import (
	"context"
	"slices"

	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/storage"
)

// faultCauses is the Blocklist cause of each release fault.
var faultCauses = map[FailureCode]grabs.BlockCause{NoVideoFile: grabs.CauseNoVideo, UnreadableFile: grabs.CauseUnreadable}

// settingCodes are the left-out reasons only an Admin's setting gives, each
// with its setting.
var settingCodes = map[ReasonCode]string{TooSmall: SmallestVideo.Key, NotAVideoType: VideoExtensions.Key, RuntimeSample: SampleShorter.Key}

// faults applies the release faults once nothing of im is left to decide
// or place and nothing was placed. When every file failed unreadable, im
// fails unreadable. When a file was left out by a setting alone, those
// files wait settings, never blocklisted. Else every video was left out by
// the name and folder rules, and im fails no_video_file.
func faults(im *Import, rules Rules) {
	if im.Failure != nil || slices.ContainsFunc(im.Files, func(f File) bool { return f.Outcome != LeftOut && f.Outcome != FailedOut }) {
		return
	}
	if len(im.Files) > 0 && !slices.ContainsFunc(im.Files, func(f File) bool { return f.Outcome != FailedOut || f.Reason.Code != Unreadable }) {
		im.Failure = new(UnreadableFile)
		return
	}
	waits := false
	for i, f := range im.Files {
		if f.Outcome != LeftOut {
			continue
		}
		key, ok := settingCodes[f.Reason.Code]
		if !ok {
			continue
		}
		r := Reason{Code: Settings, Setting: key}
		switch key {
		case SmallestVideo.Key:
			r.Value = rules.Smallest
		case SampleShorter.Key:
			r.Value = int64(rules.Sample)
		}
		im.Files[i].Outcome, im.Files[i].Reason, waits = Waiting, &r, true
	}
	if !waits {
		im.Failure = new(NoVideoFile)
	}
}

// block puts imp's Grab's Release on the Blocklist for each of causes, in
// tx.
func (im *Imports) block(ctx context.Context, tx *storage.Tx[file], g grabs.GrabID, causes []grabs.BlockCause, a *after) error {
	for _, c := range causes {
		added, err := im.cfg.Grabs.Block(ctx, tx, g, c, nil)
		if err != nil {
			return err
		}
		a.blocked = a.blocked || added
	}
	return nil
}

// misdescribed reports whether the probe of every probed video of imp
// contradicts its Release's title: the resolution the title names, or
// the HDR it names. A title that names neither contradicts nothing.
func (im *Imports) misdescribed(ctx context.Context, r storage.Reader[file], imp Import) (bool, error) {
	cuts, err := im.cfg.Targets.CutsIn(ctx, r)
	if err != nil {
		return false, err
	}
	probed := 0
	for _, f := range imp.Files {
		if f.Probe == nil {
			continue
		}
		probed++
		if !contradicts(release.Title(f.ReleaseName, im.cfg.Facets[imp.ContentType], cuts.For(imp.ContentType)).Facts, f.Probe.Probed()) {
			return false, nil
		}
	}
	return probed > 0, nil
}

// contradicts reports whether p measured another resolution than title
// names, or other HDR than the HDR it names.
func contradicts(title release.Facts, p release.Probed) bool {
	if named, ok := title.Resolution.Get(); ok {
		if got, ok := p.Resolution.Get(); ok && got != named {
			return true
		}
	}
	named, ok := title.HDR.Get()
	got, measured := p.HDR.Get()
	return ok && len(named) > 0 && measured && !slices.Equal(named, got)
}
