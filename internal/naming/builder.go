package naming

import (
	"fmt"
	"slices"

	"github.com/jatassi/slipstream/internal/works"
)

// FieldTokens is the tokens one template setting may name, for the token
// builder and the suggestions.
type FieldTokens struct {
	// Key is the setting key, "<type>.naming." and the field.
	Key    string
	Tokens []Entry
}

// Entry is one name a template may use, as the token builder offers it.
type Entry struct {
	Token
	// Modifier is the one the token takes after ":", 0 for none.
	Modifier Modifier
	// Tags are the words of the tags that may wrap the token, as in
	// {imdb-{ImdbId}}.
	Tags []string
}

// Modifier is what a token takes after ":".
type Modifier uint8

const (
	// ModPad pads a number: {season:00}.
	ModPad Modifier = iota + 1
	// ModCut keeps the start or the end of a text: {Episode Title:30}.
	ModCut
	// ModLanguages keeps or drops languages: {MediaInfo AudioLanguages:EN+DE}.
	ModLanguages
	// ModFormats keeps or drops Custom Formats: {Custom Formats:-IMAX}.
	ModFormats
	// ModFormat names the one Custom Format to print: {Custom Format:IMAX}.
	ModFormat
)

// tagsOf are the tags the builder offers around each token, by the key
// of the token they wrap.
var tagsOf = map[Key][]tag{
	KeyOf("Edition"): {editionTag},
	KeyOf("ImdbId"):  {imdbTag, imdbidTag},
	KeyOf("TmdbId"):  {tmdbTag, tmdbidTag},
	KeyOf("TvdbId"):  {tvdbTag, tvdbidTag},
}

// Catalog is t's tokens for each template setting in card order: the
// folder templates outermost first, then the file templates.
func (n *Naming) Catalog(t works.TypeID) ([]FieldTokens, error) {
	d := n.cfg.Facets[t].Naming()
	c, err := newCatalog(d)
	if err != nil {
		return nil, fmt.Errorf("read the naming tokens of %s: %w", t, err)
	}
	prefix := prefixOf(t)
	of := func(key string, f field) FieldTokens {
		out := FieldTokens{Key: prefix + key}
		for _, e := range c.ordered {
			if slices.Contains(fieldScopes[f], e.Scope) {
				out.Tokens = append(out.Tokens, e.builder())
			}
		}
		return out
	}
	out := []FieldTokens{of(folderKey, folderField)}
	for i := range d.Levels {
		out = append(out, of(levelKey(i), levelField))
	}
	if d.Aside != "" {
		out = append(out, of(asideKey, levelField))
	}
	for _, v := range d.Files {
		out = append(out, of(fileKey(v), fileField))
	}
	return out, nil
}

func (e *entry) builder() Entry {
	out := Entry{Token: e.Token, Modifier: e.modifier()}
	if e.tag == noTag {
		for _, t := range tagsOf[e.key] {
			out.Tags = append(out.Tags, t.word())
		}
	}
	return out
}

// modifier is what modFor lets e take.
func (e *entry) modifier() Modifier {
	switch e.Kind {
	case KindNumber:
		return ModPad
	case KindTitle, KindText:
		return ModCut
	case KindList:
		switch e.list {
		case langList, audioLangList:
			return ModLanguages
		case pickList:
			return ModFormat
		case nameList:
			return ModFormats
		}
	case KindYear, KindDate, KindID:
	}
	return 0
}
