package cues

import (
	"fmt"
	"testing"
)

func TestPick(t *testing.T) {
	en := lang(t, "en")
	var pack []Entry
	for _, e := range []int{6, 7, 1, 2, 3, 4, 5, 8, 9, 10} {
		pack = append(pack, Entry{Name: fmt.Sprintf("Breaking Bad  S01E%02d  Title %d.srt", e, e), Bytes: srt(e)})
	}
	pack = append(pack, Entry{Name: "Breaking Bad  S02E01  Seven Thirty-Seven.srt", Bytes: srt(50)})
	for e := 1; e <= 10; e++ {
		got, ok := Pick(pack, 1, []int{e}, en)
		if want := fmt.Sprintf("Breaking Bad  S01E%02d  Title %d.srt", e, e); !ok || got.Name != want {
			t.Errorf("episode %d picked %q, %v; want %q", e, got.Name, ok, want)
		}
	}
	if got, ok := Pick(pack, 1, []int{11}, en); ok {
		t.Errorf("episode 11 picked %q; want none", got.Name)
	}

	subdl := []Entry{
		{Name: "00 Schitt's Creek SEASON 1 subdl.com.srt", Bytes: srt(1)},
		{Name: "01 Pilot.en.srt", Bytes: srt(300)},
		{Name: "02 Our Cup Runneth Over.en.srt", Bytes: srt(310)},
	}
	if got, ok := Pick(subdl, 1, []int{2}, en); !ok || got.Name != "02 Our Cup Runneth Over.en.srt" {
		t.Errorf("a SubDL pack's episode 2 picked %q, %v", got.Name, ok)
	}

	double := []Entry{
		{Name: "Show.S03E01E02.1080p.WEB.srt", Bytes: srt(800)},
		{Name: "Show.S03E03-04.srt", Bytes: srt(800)},
		{Name: "Show.S03E05-E07.srt", Bytes: srt(800)},
		{Name: `Season 3\Show.S03E08 - 720p.srt`, Bytes: srt(400)},
		{Name: "Show.3x09.srt", Bytes: srt(400)},
	}
	for _, c := range []struct {
		episodes []int
		want     string
	}{
		{[]int{1, 2}, "Show.S03E01E02.1080p.WEB.srt"},
		{[]int{3, 4}, "Show.S03E03-04.srt"},
		{[]int{5, 6, 7}, "Show.S03E05-E07.srt"},
		{[]int{8}, `Season 3\Show.S03E08 - 720p.srt`},
		{[]int{9}, "Show.3x09.srt"},
	} {
		if got, ok := Pick(double, 3, c.episodes, en); !ok || got.Name != c.want {
			t.Errorf("episodes %v picked %q, %v; want %q", c.episodes, got.Name, ok, c.want)
		}
	}
	if got, ok := Pick(double, 3, []int{1}, en); ok {
		t.Errorf("episode 1 alone picked the double episode %q", got.Name)
	}

	single := []Entry{
		{Name: "Heat (1995) DVD.US.Retail.srt", Bytes: srt(900)},
		{Name: "Heat (1995).fr.srt", Bytes: srt(950)},
		{Name: "Heat (1995).eng.srt", Bytes: srt(880)},
	}
	if got, ok := Pick(single, 0, nil, en); !ok || got.Name != "Heat (1995).eng.srt" {
		t.Errorf("a one-title archive picked %q; want the member tagged English", got.Name)
	}
	if got, ok := Pick(single[:2], 0, nil, lang(t, "de")); !ok || got.Name != "Heat (1995).fr.srt" {
		t.Errorf("with no tag for the language, picked %q; want the larger", got.Name)
	}
	one := []Entry{{Name: "Breaking.Bad.0tv.en.srt", Bytes: srt(5)}}
	if got, ok := Pick(one, 1, []int{1}, en); !ok || got.Name != one[0].Name {
		t.Errorf("an archive whose names give no numbering picked %q, %v", got.Name, ok)
	}
	if _, ok := Pick(nil, 0, nil, en); ok {
		t.Error("an empty archive picked a member")
	}
}
