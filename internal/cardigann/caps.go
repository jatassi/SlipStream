package cardigann

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/jatassi/slipstream/internal/newznab"
)

// modeParams are the parameters Prowlarr's enums accept per mode
// (IndexerCapabilities.cs:8-61).
var modeParams = map[string][]string{
	"tv-search":    {"q", "season", "ep", "imdbid", "tvdbid", "rid", "tvmazeid", "traktid", "tmdbid", "doubanid", "genre", "year"},
	"movie-search": {"q", "imdbid", "tmdbid", "imdbtitle", "imdbyear", "traktid", "genre", "doubanid", "year"},
	"music-search": {"q", "album", "artist", "label", "year", "genre", "track"},
	"book-search":  {"q", "title", "author", "publisher", "genre", "year"},
}

// parseSearchModes is ParseCardigannSearchModes: search must be exactly
// [q], and an unknown mode, an unknown parameter or a repeated one is an
// error. Parameters are lowercased.
func parseSearchModes(modes keyedList[[]string]) (map[string][]string, error) {
	search, ok := modes.Get("search")
	if !ok {
		return nil, errors.New("the search mode search is missing")
	}
	if len(search) != 1 || search[0] != "q" {
		return nil, errors.New("the search mode search takes only q")
	}
	out := map[string][]string{"search": {"q"}}
	for _, m := range modes {
		if m.Key == "search" {
			continue
		}
		known, ok := modeParams[m.Key]
		if !ok {
			return nil, fmt.Errorf("the search mode %s isn't one Prowlarr knows", m.Key)
		}
		params := []string{}
		for _, p := range m.Value {
			p = strings.ToLower(p)
			if !slices.Contains(known, p) {
				return nil, fmt.Errorf("%s doesn't take the parameter %s", m.Key, p)
			}
			if slices.Contains(params, p) {
				return nil, fmt.Errorf("%s lists the parameter %s twice", m.Key, p)
			}
			params = append(params, p)
		}
		out[m.Key] = params
	}
	return out, nil
}

type categories struct {
	mapping  newznab.Categories
	defaults []string
}

// buildCategories is the CardigannBase constructor's category setup
// (CardigannBase.cs:69-110). A name that isn't a Newznab category is
// skipped, as Prowlarr logs and skips it.
func buildCategories(c capabilitiesBlock) (categories, error) {
	var out categories
	for _, e := range c.Categories {
		if cat, ok := newznab.ByName(e.Value); ok {
			out.mapping.Add(e.Key, cat, "")
		}
	}
	for _, m := range c.CategoryMappings {
		if m.Cat == "" {
			return categories{}, fmt.Errorf("the category mapping %s has no cat", m.ID)
		}
		cat, ok := newznab.ByName(m.Cat)
		if !ok {
			continue
		}
		out.mapping.Add(m.ID, cat, m.Desc)
		if m.Default {
			out.defaults = append(out.defaults, m.ID)
		}
	}
	return out, nil
}

// ids are the Newznab categories mapped, the defaults' first, without
// repeats.
func (c categories) ids() []newznab.ID {
	var out []newznab.ID
	add := func(m newznab.Mapping) {
		if !slices.Contains(out, m.Newznab) {
			out = append(out, m.Newznab)
		}
	}
	for _, m := range c.mapping.Mappings() {
		if slices.Contains(c.defaults, m.Tracker) {
			add(m)
		}
	}
	for _, m := range c.mapping.Mappings() {
		add(m)
	}
	return out
}
