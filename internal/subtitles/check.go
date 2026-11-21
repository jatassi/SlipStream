package subtitles

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/language"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/subtitles/cues"
	"github.com/jatassi/slipstream/internal/subtitles/internal/db"
	"github.com/jatassi/slipstream/internal/subtitles/timing"
	"github.com/jatassi/slipstream/internal/subtitling"
	"github.com/jatassi/slipstream/internal/targets"
)

// Checked is bytes the Controller vouches for, with the Sidecar they make.
// Only check, parseUpload and found make one, so the Keeper never writes
// bytes a holder sent that the Controller didn't check.
type Checked struct {
	sidecar Sidecar
	bytes   []byte
}

func (c Checked) Sidecar() Sidecar { return c.sidecar }

// asked is what a holder was asked and what a pass becomes. sidecar is the
// Sidecar a pass makes, its want, source and tail set; check fills in its
// bytes and timing. offer is the candidate a Fetch asked for, nil for a
// Sync, and with it group and hi decide whether the Sidecar is final.
type asked struct {
	sidecar Sidecar
	offer   *subtitling.Offer
	group   string
	hi      targets.HI
	// bound is the latest a retimed cue may end.
	bound time.Duration
}

// boundOf is the latest a cue of a file may end: its duration plus
// margin, or its map's length when the container states none.
func boundOf(duration time.Duration, frames int, margin time.Duration) time.Duration {
	if duration == 0 {
		duration = time.Duration(frames) * timing.Frame
	}
	return duration + margin
}

// Refused is a holder's answer the check turned away, with the rule it
// broke. It is timing.ErrForged.
type Refused struct{ Rule string }

func (r *Refused) Error() string {
	return "the Subtitles Node's response fails the Controller's check: " + r.Rule
}

func (r *Refused) Is(target error) bool { return target == timing.ErrForged }

func refused(rule string, args ...any) error { return &Refused{Rule: fmt.Sprintf(rule, args...)} }

// check is the one check of a holder's passing answer. The answer must be
// the candidate asked, both byte sets must fit limit and be UTF-8 in the
// format the answer declares, with the same cues and text, and every
// retimed cue must end by a.bound. The claimed segments must refit from
// the two sets of cue times, and the timing match recomputed on the
// file's own map must pass g. The Sidecar's numbers are the Controller's,
// and whether it is final is the Controller's own reading of the
// candidate's release. Every refusal is timing.ErrForged.
func check(a asked, ref timing.Reference, ans subtitling.Fetched, g timing.Gate, k timing.Constants, limit int) (Checked, error) {
	if a.offer != nil && ans.Key != a.offer.Key {
		return Checked{}, refused("it's candidate %q, not the %q asked for", ans.Key, a.offer.Key)
	}
	lang := a.sidecar.Tail.Language
	original, err := parsed("original", ans.Original, ans.Format, lang, limit)
	if err != nil {
		return Checked{}, err
	}
	retimed, err := parsed("retimed subtitle", ans.Retimed, ans.Format, lang, limit)
	if err != nil {
		return Checked{}, err
	}
	if !cues.SameText(original, retimed) {
		return Checked{}, refused("its text differs from the original's")
	}
	for _, t := range retimed.Times() {
		if t.End > a.bound {
			return Checked{}, refused("a cue ends at %v, past %v", t.End, a.bound)
		}
	}
	firsts := make([]int, len(ans.Claimed.Segments))
	for i, s := range ans.Claimed.Segments {
		firsts[i] = s.First
	}
	fit, err := timing.Judge(ref, subtitling.Spans(original), subtitling.Spans(retimed), firsts, ans.Claimed.Prominence, g, k, original.Unit())
	if err != nil {
		return Checked{}, refused("its cue times don't follow the segments it claims")
	}
	if fit.Verdict != timing.Passed {
		return Checked{}, refused("its timing match on this file's map is %d%%", percent(fit.Match))
	}
	sc := a.sidecar
	if o := a.offer; o != nil {
		sc.Source, sc.Tail.HI = FromProvider, o.HI
		sc.Offer = &Offer{Provider: o.Provider, Key: o.Key, Release: first(o.Releases), AI: o.AI}
		sc.Final = finalOf(*o, a.group, a.hi)
	}
	sc.Tail.Ext = string(retimed.Ext())
	if sc.TailText == "" {
		sc.TailText = sc.Tail.String()
	}
	sc.Fit = &Fit{Offset: fit.Offset().Round(time.Millisecond), Ratio: fit.Ratio(), Segments: len(fit.Segments), Match: percent(fit.Match)}
	sc.TimingDue = false
	return checkedOf(sc, retimed), nil
}

// parsed is b read as format, which it must be as sent: UTF-8 with no
// byte order mark, in SRT, ASS or SSA.
func parsed(which string, b []byte, format cues.Format, lang language.Code, limit int) (cues.Doc, error) {
	if len(b) > limit {
		return cues.Doc{}, refused("the %s is %d bytes, over the %d byte cap", which, len(b), limit)
	}
	d, err := cues.Parse(b, lang, 0, limit)
	if err != nil {
		return cues.Doc{}, refused("the %s doesn't parse: %v", which, err)
	}
	if d.From() != format || d.Format() != format {
		return cues.Doc{}, refused("the %s is %v, not the %v it declares", which, d.From(), format)
	}
	if !bytes.Equal(d.Bytes(), b) {
		return cues.Doc{}, refused("the %s isn't UTF-8 without a byte order mark", which)
	}
	return d, nil
}

func checkedOf(sc Sidecar, d cues.Doc) Checked {
	b := d.Bytes()
	sc.SHA256, sc.Size, sc.Cues = sha256.Sum256(b), int64(len(b)), d.Len()
	return Checked{sidecar: sc, bytes: b}
}

// finalOf is whether a candidate ends searching for its want: a person
// made it for this file's Release group, and it isn't hearing impaired
// against the want's preference.
func finalOf(o subtitling.Offer, group string, hi targets.HI) bool {
	if o.AI || group == "" || o.HI && hi == targets.Avoid {
		return false
	}
	for _, name := range o.Releases {
		if g, ok := release.Title(name, release.TitleOnly{}, nil).Facts.Group.Get(); ok && strings.EqualFold(g, group) {
			return true
		}
	}
	return false
}

func first(s []string) string {
	if len(s) == 0 {
		return ""
	}
	return s[0]
}

func percent(f float64) int { return int(math.Round(max(f, 0) * 100)) }

// parseUpload vouches for an Admin's bytes kept untimed: parsed,
// converted to SRT when they are in another text format, and UTF-8.
func parseUpload(b []byte, w Want, limit int) (Checked, error) {
	d, err := cues.Parse(b, w.Language, 0, limit)
	if err != nil {
		return Checked{}, err
	}
	t := Tail{Language: w.Language, Forced: w.Forced, Ext: string(d.Ext())}
	return checkedOf(Sidecar{Tail: t, TailText: t.String(), Source: FromUpload, Final: true}, d), nil
}

// found vouches for bytes the Keeper read back for sc, which must be the
// bytes sc's row names.
func found(sc Sidecar, b []byte) (Checked, error) {
	if sha256.Sum256(b) != sc.SHA256 {
		return Checked{}, fmt.Errorf("the bytes read for Sidecar %s aren't the ones its row names", sc.ID)
	}
	return Checked{sidecar: sc, bytes: b}, nil
}

var framerates = map[string]timing.Framerates{"none": timing.NoRatios, "standard": timing.StandardRatios, "range": timing.RangeRatios}

// defaultMaxOffset is subtitles.sync.maxOffset's default.
const defaultMaxOffset = 30 * time.Second

// gate is the Admin's Timing sync settings, in r.
func (s *Subtitles) gate(ctx context.Context, r storage.Reader[storage.Library]) (timing.Gate, error) {
	row, err := db.New(r).GetGate(ctx)
	if err != nil {
		return timing.Gate{}, err
	}
	g := timing.Gate{Match: float64(timing.DefaultMatchPercent) / 100, MaxOffset: defaultMaxOffset, Framerates: timing.StandardRatios}
	if row.SyncMatch != nil {
		g.Match = float64(*row.SyncMatch) / 100
	}
	if row.SyncMaxOffsetSeconds != nil {
		g.MaxOffset = time.Duration(*row.SyncMaxOffsetSeconds) * time.Second
	}
	if row.SyncFramerates != nil {
		g.Framerates = framerates[*row.SyncFramerates]
	}
	return g, nil
}
