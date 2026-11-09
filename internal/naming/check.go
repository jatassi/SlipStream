package naming

import (
	"errors"
	"fmt"
	"iter"
	"strings"

	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/release"
)

// sampleExt is the extension the check names samples with.
const sampleExt = ".mkv"

// fileSample is one of the core's files a type's samples are crossed
// with. bare marks the one with no release facts, which a Missing sample
// pairs with alone.
type fileSample struct {
	label string
	facts FileFacts
	bare  bool
}

var fileSamples = []fileSample{
	{label: "a Standard file", facts: FileFacts{Target: "1080p"}},
	{label: "a cut", facts: FileFacts{Target: "4K HDR", Cut: "Director's Cut"}},
	{label: "a Kept file", facts: FileFacts{Target: "1080p (extra 1)"}},
	{label: "every release fact", facts: FileFacts{
		Target:  "4K HDR (extra 2)",
		Cut:     "Extended",
		Release: "The.Quiet.Orchard.2021.2160p.UHD.BluRay.REMUX.DV.HDR10.HEVC.TrueHD.Atmos.7.1-FLUX",
		Source:  "the.quiet.orchard.2021.2160p.uhd.bluray.remux.dv.hdr10.hevc.truehd.atmos.7.1-flux.mkv",
		Formats: []string{"IMAX", "HDR10+"},
		Facts: release.Facts{
			Quality:   release.Known(release.Bluray2160p),
			Revision:  release.Revision{Version: 2, Real: 1, Repack: true},
			Group:     release.Known("FLUX"),
			HDR:       release.Known([]release.HDR{"DV", "HDR10"}),
			Video:     release.Known(release.VideoCodec("x265")),
			Audio:     release.Known([]release.AudioCodec{"TrueHD", "DD+"}),
			Channels:  release.Known([]release.Channels{"7.1"}),
			Languages: release.Known([]release.Language{release.Original, "en", "de", "fr"}),
			Coverage:  release.Coverage{Original: release.Known(release.Language("ja"))},
		},
		Media: media.Record{
			Video:     media.Video{BitDepth: 10},
			Subtitles: []media.Subtitle{{Language: "en"}, {Language: "de"}, {Language: "fr"}, {Language: "ja"}},
		},
	}},
	{label: "no release facts", facts: FileFacts{Target: "1080p"}, bare: true},
}

// check judges s as one save would leave it, naming samples within
// budget bytes: every folder renders non-empty for every sample, and with
// renaming on each file variant holds {Target} and an Edition token, its
// anchor is followed by " - " with no Item or file token before it, and
// every crossed sample fits budget and keeps its anchor. newScheme has
// already refused templates that don't parse or bind. With renaming off
// that is all the check judges.
func check(s *Scheme, budget int) []Problem {
	c := problems{}
	for _, sample := range s.decl.Samples {
		c.folders(s, sample)
	}
	if !s.rename {
		return c.list
	}
	for v, variant := range s.decl.Files {
		key := fileKey(variant)
		for _, sentence := range s.holds(v) {
			c.add(key, sentence)
		}
		if p := s.anchorProblem(v); p != "" {
			c.add(key, p)
		}
		if c.has(key) {
			continue
		}
		c.samples(s, v, budget)
	}
	return c.list
}

type problems struct{ list []Problem }

func (c *problems) add(key, sentence string) {
	c.list = append(c.list, Problem{Key: key, Sentence: sentence})
}

func (c *problems) has(key string) bool {
	for _, p := range c.list {
		if p.Key == key {
			return true
		}
	}
	return false
}

// once adds the problem unless key already has one.
func (c *problems) once(key, sentence string) {
	if !c.has(key) {
		c.add(key, sentence)
	}
}

// folders checks that each folder template renders a name for sample.
func (c *problems) folders(s *Scheme, sample Sample) {
	st := s.style()
	empty := func(b bound, sub Subject) bool { return segment(st.render(b, sub, FileFacts{})) == "" }
	sentence := fmt.Sprintf("The folder name is empty for the sample %q. Use a token every Work has a value for, such as its title.", sample.Label)
	if empty(s.folder, Subject{Work: sample.Work}) {
		c.once(folderKey, sentence)
	}
	if len(sample.Units) == 0 {
		return
	}
	first := Subject{Work: sample.Work, Units: sample.Units[:1]}
	for i, b := range s.levels {
		if empty(b, first) {
			c.once(levelKey(i), sentence)
		}
	}
	if s.decl.Aside != "" && empty(s.aside, first) {
		c.once(asideKey, sentence)
	}
}

// holds is what file variant v lacks of {Target} and an Edition token.
func (s *Scheme) holds(v int) []string {
	var target, edition bool
	for _, p := range s.files[v].parts {
		if r, ok := p.(*boundRef); ok {
			target = target || r.entry.key == KeyOf("Target")
			edition = edition || r.entry.key == KeyOf("Edition") || r.entry.key == KeyOf("Edition Tag")
		}
	}
	var out []string
	if !target {
		out = append(out, "The file name has no {Target}. Add {Target} so each file of a Work gets its own name.")
	}
	if !edition {
		out = append(out, "The file name has no Edition token. Add {Edition Tag} or {Edition} so each cut gets its own name.")
	}
	return out
}

// samples names every crossing of variant v, and adds the first that
// doesn't fit budget or loses its anchor.
func (c *problems) samples(s *Scheme, v, budget int) {
	key := fileKey(s.decl.Files[v])
	st := s.style()
	for x := range s.crossings(v) {
		which := fmt.Sprintf("For the sample %q with %s, ", x.sample.Label, x.file.label)
		_, err := s.Name(Placing{Subject: x.sub, LevelFolders: true, Landing: Rendered(), File: x.file.facts, Ext: sampleExt, Budget: budget})
		var long *TooLongError
		if errors.As(err, &long) {
			c.add(key, which+long.Error()+". Shorten the template.")
			return
		}
		name, anchor := st.render(s.files[v], x.sub, x.file.facts), s.anchorText(v, x.sub, x.file.facts)
		if !strings.HasPrefix(name, anchor+dash) {
			c.add(key, which+s.lostAnchor(v, name, anchor))
			return
		}
	}
}

// crossing is a type sample paired with one of the core's file samples.
type crossing struct {
	sample Sample
	sub    Subject
	file   fileSample
}

// crossings are the type's samples that name with variant v, each
// crossed with the core's file samples: a Missing sample with the file
// that has no release facts, every other with the rest. A sample whose
// first Item can't take v names with the first variant, which is judged
// on its own.
func (s *Scheme) crossings(v int) iter.Seq[crossing] {
	return func(yield func(crossing) bool) {
		for _, sample := range s.decl.Samples {
			sub := Subject{Work: sample.Work, Units: make([]Unit, len(sample.Units))}
			for i, u := range sample.Units {
				u.Item.Variant = s.decl.Files[v].Key
				sub.Units[i] = u
			}
			if s.variant(sub) != v {
				continue
			}
			for _, fs := range fileSamples {
				if sample.Missing == fs.bare && !yield(crossing{sample: sample, sub: sub, file: fs}) {
					return
				}
			}
		}
	}
}

func (s *Scheme) lostAnchor(v int, name, anchor string) string {
	if s.decl.Files[v].Anchor.kind == folderAnchor {
		return fmt.Sprintf("the file name %q doesn't start with the folder name %q and ' - '. Write the start of the file template the way the folder template is written, %s.", name, anchor, groupWork)
	}
	return fmt.Sprintf("the file name %q doesn't start with %q and ' - '. Put ' - ' right after it, %s.", name, anchor, groupItem)
}
