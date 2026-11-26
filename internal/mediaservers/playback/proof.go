package playback

import (
	"slices"

	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/release"
)

// need is whether a play left a fact as the file holds it.
type need func(Play) bool

var (
	needDirectPlay    need = func(p Play) bool { return p.Decision == link.DecisionDirectPlay }
	needVideo         need = func(p Play) bool { return p.Streams.Video == link.StreamDirect }
	needAudio         need = func(p Play) bool { return p.Streams.Audio == link.StreamDirect }
	needSubtitles     need = func(p Play) bool { return p.Streams.Subtitles == link.SubtitlesDirect }
	needVideoAndAudio need = func(p Play) bool { return needVideo(p) && needAudio(p) }
)

// unchanged is which streams must be direct for a play to have received a
// fact as the file holds it.
var unchanged = map[FactKind]need{
	FactContainer:  needDirectPlay,
	FactResolution: needVideo, FactVideoCodec: needVideo, FactVideoProfile: needVideo,
	FactBitDepth: needVideo, FactHDR: needVideo, FactDolbyVision: needVideo,
	FactAudioCodec: needAudio, FactChannels: needAudio,
	FactSubtitleCodec: needSubtitles, FactSubtitleImage: needSubtitles,
	FactBitrate: needVideoAndAudio,
}

// evidence is whether p proves anything: a snapshot, and seen on more than
// one look.
func evidence(p Play) bool { return p.Snapshot != nil && p.LastSeen.After(p.Started) }

// Proofs are the values one player received unchanged, each with the
// newest play that proves it.
type Proofs struct {
	byKind map[FactKind]map[string]proof
}

type proof struct {
	rank int64
	play Play
}

// prove is what plays, one player's oldest first, prove.
func prove(plays []Play) Proofs {
	pr := Proofs{byKind: map[FactKind]map[string]proof{}}
	for _, p := range plays {
		if !evidence(p) {
			continue
		}
		for _, f := range p.Snapshot.Facts.Tracked() {
			if !unchanged[f.Kind](p) {
				continue
			}
			if pr.byKind[f.Kind] == nil {
				pr.byKind[f.Kind] = map[string]proof{}
			}
			pr.byKind[f.Kind][f.Text] = proof{rank: f.Rank, play: p}
		}
	}
	return pr
}

// proven is the newest play proving f: the value itself, or for an
// ordered kind any higher one.
func (pr Proofs) proven(f Fact) (Play, bool) {
	if !ordered(f.Kind) {
		p, ok := pr.byKind[f.Kind][f.Text]
		return p.play, ok
	}
	var newest Play
	found := false
	for text, p := range pr.byKind[f.Kind] {
		if text != f.Text && p.rank < f.Rank {
			continue
		}
		if !found || newer(p.play, newest) {
			newest, found = p.play, true
		}
	}
	return newest, found
}

func newer(a, b Play) bool {
	return a.Started.After(b.Started) || a.Started.Equal(b.Started) && a.ID.String() > b.ID.String()
}

// ordered is whether a value of k is proven by a higher one too.
func ordered(k FactKind) bool {
	for _, d := range dimensions {
		if d.Fact == k && d.Lower {
			return true
		}
	}
	return false
}

// provenFile is whether f's value on d is proven, so it can't have been
// the block.
func (pr Proofs) provenFile(d Dimension, f Facts) bool {
	k := dimensions[order(d)].Fact
	tracked := f.Tracked()
	i := slices.IndexFunc(tracked, func(t Fact) bool { return t.Kind == k })
	if k == "" || i < 0 {
		return false
	}
	_, ok := pr.proven(tracked[i])
	return ok
}

// unprovenHDR is the file's HDR formats other than SDR that this player
// hasn't proven, most demanding first. A Dolby Vision profile 8.1 file is
// DV and HDR10, so with HDR10 proven it is DV.
func (pr Proofs) unprovenHDR(v Video) []release.HDR {
	return slices.DeleteFunc(hdrOf(v), func(h release.HDR) bool {
		_, ok := pr.proven(Fact{Kind: FactHDR, Text: string(h)})
		return ok
	})
}

// blockedHDR is the file's value on hdr: its unproven HDR formats, or
// every format other than SDR when the player proved them all and a
// reason names hdr anyway.
func (pr Proofs) blockedHDR(v Video) []release.HDR {
	if hdr := pr.unprovenHDR(v); len(hdr) > 0 {
		return hdr
	}
	return hdrOf(v)
}

// hdrOf is v's HDR formats other than SDR, most demanding first.
func hdrOf(v Video) []release.HDR {
	var out []release.HDR
	for _, h := range release.HDRs() {
		if h != "SDR" && slices.Contains(v.HDR, h) {
			out = append(out, h)
		}
	}
	return out
}

// evidenceOf is the newest play proving each of fs.
func (pr Proofs) evidenceOf(in Input, fs []Fact) []Evidence {
	out := make([]Evidence, 0, len(fs))
	for _, f := range fs {
		p, _ := pr.proven(f)
		w := in.Works[p.Snapshot.Work]
		out = append(out, Evidence{
			FactValue: FactValue{Fact: f.Kind, Value: f.Text}, PlayID: p.ID.String(), WorkID: p.Snapshot.Work.String(),
			Title: w.Label, ContentType: string(w.ContentType), StartedAt: p.Started,
		})
	}
	return out
}
