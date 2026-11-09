package files_test

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/jatassi/slipstream/internal/media"
	"math/rand/v2"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"strings"
	"sync"
	"testing"

	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/formats"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// The spec's size for a scored format edit's preview: 20,000 movie Works
// and 200,000 episode files, here 1,000 series of 10 seasons of 20.
const (
	singleWorks    = 20_000
	seasonedWorks  = 1_000
	seasonsEach    = 10
	episodesEach   = 20
	trashRadarrDir = "../formats/testdata/trash/radarr/cf"
)

// pastedFresh is 30 formats TRaSH's HD Bluray + WEB profile scores,
// beyond the two the library's Target scores already.
var pastedFresh = []string{
	"br-disk", "lq", "lq-release-title", "x265-hd", "extras", "av1", "repack-proper", "repack2", "repack3",
	"hd-bluray-tier-01", "hd-bluray-tier-02", "hd-bluray-tier-03", "web-tier-02", "web-tier-03", "remaster",
	"4k-remaster", "criterion-collection", "imax", "imax-enhanced", "amzn", "atvp", "dsnp", "hmax", "nf",
	"pcok", "scene", "bad-dual-groups", "no-rlsgroup", "obfuscated", "retags",
}

// trash is TRaSH's Radarr format of name, with a new id, and its default
// score.
func trash(b *testing.B, name string) (f formats.Format, score int) {
	b.Helper()
	text, err := os.ReadFile(filepath.Join(trashRadarrDir, name+".json"))
	if err != nil {
		b.Fatal(err)
	}
	fs, err := formats.Paste(text)
	if err != nil {
		b.Fatal(err)
	}
	var doc struct {
		Scores map[string]int `json:"trash_scores"`
	}
	if err := json.Unmarshal(text, &doc); err != nil {
		b.Fatal(err)
	}
	fs[0].ID = targets.NewFormatID()
	return fs[0], doc.Scores["default"]
}

// withPattern is f with condition i's pattern set to pattern.
func withPattern(f formats.Format, i int, pattern string) formats.Format {
	value, err := json.Marshal(map[string]string{"value": pattern})
	if err != nil {
		panic(err)
	}
	f.Conditions = append([]formats.Condition(nil), f.Conditions...)
	f.Conditions[i].Fields = value
	return f
}

// stockroom is a library at the spec's size whose Profile holds the Target
// HD, which scores TRaSH's 3D and WEB Tier 01, and Any, which scores
// none, with 30 more TRaSH formats pasted and unscored.
type stockroom struct {
	*library
	svc          *files.Files
	hd, any      targets.Target
	hdSpec       targets.Spec
	profile      targets.ProfileID
	threeD, tier formats.Format
	fresh        map[formats.ID]int
	count        int
}

var hdTiers = [][]release.Quality{
	{release.Bluray2160p, release.WEBDL2160p, release.WEBRip2160p},
	{release.Bluray1080p},
	{release.WEBDL1080p, release.WEBRip1080p},
	{release.Bluray720p, release.WEBDL720p, release.WEBRip720p},
}

func stockUp(b *testing.B) *stockroom {
	b.Helper()
	l := open(b)
	s := &stockroom{library: l, svc: l.files(files.NoGrabs{}), fresh: map[formats.ID]int{}}
	threeD, threeDScore := trash(b, "3d")
	tier, tierScore := trash(b, "web-tier-01")
	s.threeD, s.tier = threeD, tier
	all := []formats.Format{threeD, tier}
	for _, name := range pastedFresh {
		f, score := trash(b, name)
		all = append(all, f)
		s.fresh[f.ID] = score
	}
	if _, err := s.svc.Change(b.Context(), admin, targets.AddFormats{Dialect: formats.Radarr, Add: all}, files.Mode{Unwanted: files.Keep}); err != nil {
		b.Fatal(err)
	}
	s.hd, s.any = l.target(b, "HD", hdTiers...), l.target(b, "Any", hdTiers...)
	s.profile = l.profile(b, "Library", s.hd, s.any)
	r := rand.New(rand.NewPCG(20, 200)) //nolint:gosec // seeded on purpose so runs compare
	var shelf []stock
	for m := range singleWorks {
		shelf = append(shelf, stock{key: works.Key(fmt.Sprintf("m%d", m)), title: fmt.Sprintf("Movie.Number.%d.%d", m, 1950+m%70), items: 1})
	}
	for sw := range seasonedWorks {
		shelf = append(shelf, stock{key: works.Key(fmt.Sprintf("s%d", sw)), title: fmt.Sprintf("Series.Number.%d", sw), items: seasonsEach * episodesEach})
	}
	for i := range shelf {
		shelf[i].tails = make([]tail, shelf[i].items)
		for j := range shelf[i].tails {
			if j%episodesEach == 0 {
				shelf[i].tails[j] = newTail(r)
			} else {
				shelf[i].tails[j] = shelf[i].tails[j-1]
			}
		}
	}
	reads := s.readAll(shelf)
	for _, st := range shelf {
		s.stock(b, st, reads)
	}
	s.count = len(shelf)
	s.hdSpec = targets.Spec{
		Name: "HD", Tiers: hdTiers, Cutoff: release.Bluray1080p,
		Scores: targets.Scores{Formats: map[formats.ID]int{threeD.ID: threeDScore, tier.ID: tierScore}, Increment: 1},
	}
	if _, err := s.svc.Change(b.Context(), admin, targets.UpdateTarget{ID: s.hd.ID, Spec: s.hdSpec}, files.Mode{Unwanted: files.Keep}); err != nil {
		b.Fatal(err)
	}
	return s
}

// stock is one Work to add: its title and one release tail per Item. A
// season's episodes share one, as a season pack's do.
type stock struct {
	key   works.Key
	title string
	items int
	tails []tail
}

// tail is a release name after its title, and the quality it reads as.
type tail struct {
	text    string
	quality release.Quality
}

func (st stock) name(i int) string {
	if st.items == 1 {
		return st.title + "." + st.tails[i].text
	}
	return fmt.Sprintf("%s.S%02dE%02d.%s", st.title, i/episodesEach+1, i%episodesEach+1, st.tails[i].text)
}

var (
	releaseGroups = []string{
		"FLUX", "NTb", "CMRG", "SiC", "TEPES", "HONE", "KiNGS", "playWEB", "SMURF", "KOGi", "EDITH", "GNOME",
		"DON", "CtrlHD", "ZQ", "decibeL", "BMF", "EbP", "FraMeSToR", "SPARKS", "GECKOS", "AMIABLE", "ROVERS",
		"YIFY", "YTS.MX", "RARBG", "EVO", "MeGusta", "ELiTE", "NOGRP", "TOMMY", "BLOOM", "HiQVE", "DiN",
	}
	services = []string{"AMZN", "NF", "DSNP", "ATVP", "HMAX", "PCOK", "HULU", "iT"}
	audios   = []string{"DDP5.1", "DD5.1", "AAC2.0", "DTS-HD.MA.5.1", "TrueHD.7.1.Atmos", "DDP5.1.Atmos", "DTS"}
	extras   = []struct {
		word   string
		chance float64
	}{{"REPACK", 0.03}, {"PROPER", 0.02}, {"IMAX", 0.02}, {"REMASTERED", 0.02}, {"Hybrid", 0.03}, {"3D", 0.005}}
)

func pick[T any](r *rand.Rand, from []T) T { return from[r.IntN(len(from))] }

func newTail(r *rand.Rand) tail {
	res, source := "1080p", "WEB-DL"
	switch x := r.Float64(); {
	case x < 0.2:
		res = "2160p"
	case x < 0.3:
		res = "720p"
	}
	switch x := r.Float64(); {
	case x < 0.4:
		source = "BluRay"
	case x < 0.55:
		source = "WEBRip"
	}
	var words []string
	for _, e := range extras {
		if r.Float64() < e.chance {
			words = append(words, e.word)
		}
	}
	words = append(words, res)
	if source != "BluRay" {
		words = append(words, pick(r, services))
	}
	words = append(words, source)
	if res == "2160p" && r.Float64() < 0.6 {
		words = append(words, pick(r, []string{"HDR", "DV.HDR", "HDR10Plus", "DV"}))
	}
	codec := pick(r, []string{"x264", "H.264"})
	if res == "2160p" || r.Float64() < 0.2 {
		codec = pick(r, []string{"x265", "H.265", "HEVC"})
	}
	words = append(words, pick(r, audios), codec)
	return tail{text: strings.Join(words, ".") + "-" + pick(r, releaseGroups), quality: qualityOf(res, source)}
}

func qualityOf(res, source string) release.Quality {
	return map[string]release.Quality{
		"2160pBluRay": release.Bluray2160p, "2160pWEB-DL": release.WEBDL2160p, "2160pWEBRip": release.WEBRip2160p,
		"1080pBluRay": release.Bluray1080p, "1080pWEB-DL": release.WEBDL1080p, "1080pWEBRip": release.WEBRip1080p,
		"720pBluRay": release.Bluray720p, "720pWEB-DL": release.WEBDL720p, "720pWEBRip": release.WEBRip720p,
	}[res+source]
}

// readAll reads every Work's release names across the cores, one read
// per tail and kind of Work.
func (s *stockroom) readAll(shelf []stock) map[string]release.Read {
	unique := map[string]bool{}
	for _, st := range shelf {
		for i := range st.tails {
			unique[st.sample(i)] = true
		}
	}
	names := make(chan string)
	reads := map[string]release.Read{}
	var mu sync.Mutex
	var wg sync.WaitGroup
	for range runtime.GOMAXPROCS(0) {
		wg.Go(func() {
			for name := range names {
				r := release.Title(name, s.fx, nil)
				mu.Lock()
				reads[name] = r
				mu.Unlock()
			}
		})
	}
	for name := range unique {
		names <- name
	}
	close(names)
	wg.Wait()
	return reads
}

// sample is the name tail i of st is read by: a movie's or an episode's
// of a fixed title.
func (st stock) sample(i int) string {
	if st.items == 1 {
		return "Movie.Name.2010." + st.tails[i].text
	}
	return "Series.Name.S01E01." + st.tails[i].text
}

// stock adds st's Work under the Library Profile and a file for each of
// its Items filling both Targets.
func (s *stockroom) stock(b *testing.B, st stock, reads map[string]release.Read) {
	b.Helper()
	fetched := works.Fetched{Label: st.title}
	if st.items == 1 {
		fetched.Children = []works.Entry{{Key: "b1", Label: "Film", Children: []works.Entry{{Key: "c1", Label: "Feature"}}}}
	} else {
		for season := 1; season <= seasonsEach; season++ {
			level := works.Entry{Key: works.Key(fmt.Sprintf("b%d", season)), Label: fmt.Sprintf("Season %d", season)}
			for e := 1; e <= episodesEach; e++ {
				level.Children = append(level.Children, works.Entry{Key: works.Key(fmt.Sprintf("c%d-%d", season, e)), Label: fmt.Sprintf("S%02dE%02d", season, e)})
			}
			fetched.Children = append(fetched.Children, level)
		}
	}
	s.fx.Put(st.key, contenttypetest.Record{Tree: fetched})
	tree, _, err := s.works.Add(b.Context(), admin, contenttypetest.ID, st.key, works.Adding{
		Monitor: true,
		Also: func(ctx context.Context, tx *storage.Tx[storage.Library], w works.Summary) error {
			if err := targets.Assign(ctx, tx, w, &s.profile); err != nil {
				return err
			}
			return roots.Assign(ctx, tx, w, nil)
		},
	})
	if err != nil {
		b.Fatal(err)
	}
	err = s.lib.Write(b.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		for i, it := range tree.Items {
			name := st.name(i)
			_, err := files.Add(ctx, tx, files.File{
				WorkID: tree.ID, ItemID: it.ID, Cut: release.Standard, RootID: s.root.ID, Path: st.title + "/" + name + ".mkv",
				Size: 4 << 30, Media: media.Record{Video: media.Video{Width: 1920, Height: 1080}}, Quality: st.tails[i].quality,
				ReleaseName: name, Title: reads[st.sample(i)],
			}, []targets.TargetID{s.hd.ID, s.any.ID})
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		b.Fatal(err)
	}
}

// preview runs c's preview once per loop and reports the Works its
// Impact counts, failing when it counts fewer than least. Its CPU
// profile samples carry the label case, the benchmark's name, so
// -tagfocus splits the parallel flip pass by case too.
func (s *stockroom) preview(b *testing.B, c targets.Change, least int) {
	b.Helper()
	var impact files.Impact
	pprof.Do(b.Context(), pprof.Labels("case", b.Name()), func(ctx context.Context) {
		for b.Loop() {
			var err error
			impact, err = s.svc.Change(ctx, admin, c, files.Mode{Unwanted: files.Delete, Preview: true})
			if err != nil {
				b.Fatal(err)
			}
		}
	})
	if impact.Works < least {
		b.Fatalf("the preview's Impact = %+v, want at least %d Works", impact, least)
	}
	b.ReportMetric(float64(impact.Works), "works")
}

// BenchmarkPreviewingAScoredFormatEdit measures, at the spec's size of
// 20,000 movie Works and 200,000 episode files, previewing an edit to a
// scored format of 3 patterns (TRaSH's 3D) and of 22 (WEB Tier 01), a
// Target edit scoring 30 freshly pasted TRaSH formats, and, as the
// baseline, reordering the Profile every Work wants from.
func BenchmarkPreviewingAScoredFormatEdit(b *testing.B) {
	s := stockUp(b)
	b.Run("ThreePatterns", func(b *testing.B) {
		s.preview(b, targets.UpdateFormat{ID: s.threeD.ID, Format: withPattern(s.threeD, 2, `\b(BD3D|IMAX)\b`)}, 1)
	})
	b.Run("TwentyTwoPatterns", func(b *testing.B) {
		s.preview(b, targets.UpdateFormat{ID: s.tier.ID, Format: withPattern(s.tier, 8, `^(FLUX|EDITH)$`)}, 1)
	})
	b.Run("ThirtyPastedFormatsScored", func(b *testing.B) {
		spec := s.hdSpec
		spec.Scores.Formats = map[formats.ID]int{}
		for id, n := range s.hdSpec.Scores.Formats {
			spec.Scores.Formats[id] = n
		}
		for id, n := range s.fresh {
			spec.Scores.Formats[id] = n
		}
		s.preview(b, targets.UpdateTarget{ID: s.hd.ID, Spec: spec}, 1)
	})
	b.Run("ProfileReorderBaseline", func(b *testing.B) {
		s.preview(b, targets.UpdateProfile{ID: s.profile, Spec: targets.ProfileSpec{
			Name: "Library", ContentType: contenttypetest.ID, Entries: entries(s.any, s.hd),
		}}, s.count)
	})
}
