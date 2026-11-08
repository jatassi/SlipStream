package rutracker

import (
	"testing"

	"github.com/jatassi/slipstream/internal/newznab"
)

func TestTitleParser(t *testing.T) {
	movie := []newznab.Category{newznab.MoviesHD}
	tv := []newznab.Category{newznab.TVHD}
	for _, c := range []struct {
		name                     string
		in                       string
		cats                     []newznab.Category
		strip, first, all, addRu bool
		want                     string
	}{
		{name: "strip", in: "Тихий сад / The Quiet Orchard (Иван Петров / Ivan Petrov) [2021, США, драма, WEB-DL 1080p] Dub", cats: movie, strip: true,
			want: "The Quiet Orchard [2021, WEB-DL 1080p] Dub"},
		{name: "first tags", in: "[Тест] (2021) The Quiet Orchard [WEB-DL 1080p]", cats: movie, first: true,
			want: "(2021) The Quiet Orchard [WEB-DL 1080p] [Тест]"},
		{name: "all tags", in: "The Quiet Orchard [2021] (Director's Cut) [WEB-DL 1080p]", cats: movie, all: true,
			want: "The Quiet Orchard (Director's Cut) [2021] [WEB-DL 1080p]"},
		{name: "nested tags", in: "The Quiet Orchard [2021, [WEB-DL] 1080p] (Cut (Extended))", cats: movie, all: true,
			want: "The Quiet Orchard (Cut (Extended)) [2021, [WEB-DL] 1080p]"},
		// The C#'s "\bRUS\b" is a backspace on each side, so RUS is added again.
		{name: "rus", in: "The Quiet Orchard RUS [2021]", cats: movie, addRu: true,
			want: "The Quiet Orchard RUS [2021] RUS"},
		{name: "tv anime", in: "Огни гавани ТВ-2 [1+2 из 12] [2024]", cats: tv,
			want: "Огни гавани S2 E1+2 of 12 [2024]"},
		{name: "cyrillic x", in: "Огни гавани - Х (Сезон 1) Серии: 4 из 10", cats: tv,
			want: "Огни гавани - XX (S1E4 of 10"},
		{name: "music untouched", in: "Salt Meadow (Иван) [2020, FLAC]", cats: []newznab.Category{newznab.AudioLossless},
			want: "Salt Meadow (Иван) [2020, FLAC]"},
	} {
		if got := titleParser.Parse(c.in, c.cats, c.strip, c.first, c.all, c.addRu); got != c.want {
			t.Errorf("%s: Parse(%q) = %q, want %q", c.name, c.in, got, c.want)
		}
	}
}
