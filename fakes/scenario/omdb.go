package scenario

import "slices"

// OMDb is what fake-omdb serves: ratings and awards per IMDb id, and how
// many calls one key gets a UTC day.
type OMDb struct {
	// DailyLimit is the calls one key gets a UTC day before OMDb answers
	// "Request limit reached!"; zero is no limit.
	DailyLimit int         `jsonschema:"minimum=0" yaml:"daily_limit,omitempty"`
	Titles     []OMDbTitle `yaml:"titles,omitempty"`
}

// OMDbTitle is one title OMDb knows. A rating left out is one OMDb has
// none of.
type OMDbTitle struct {
	IMDbID     string   `jsonschema:"required,pattern=^tt[0-9]{7\\,}$" yaml:"imdb_id"`
	IMDbRating *float64 `jsonschema:"minimum=0,maximum=10"            yaml:"imdb_rating,omitempty"`
	IMDbVotes  *int64   `jsonschema:"minimum=0"                       yaml:"imdb_votes,omitempty"`
	// RottenTomatoes is the critics score, a percentage.
	RottenTomatoes *int   `jsonschema:"minimum=0,maximum=100" yaml:"rotten_tomatoes,omitempty"`
	Metacritic     *int   `jsonschema:"minimum=0,maximum=100" yaml:"metacritic,omitempty"`
	Awards         string `yaml:"awards,omitempty"`
}

func (o OMDb) check(add func(format string, args ...any)) {
	if o.DailyLimit < 0 {
		add("omdb daily_limit %d is below zero", o.DailyLimit)
	}
	for i, t := range o.Titles {
		if !imdbID.MatchString(t.IMDbID) {
			add("omdb title imdb_id %q is not tt and seven or more digits", t.IMDbID)
		}
		if slices.ContainsFunc(o.Titles[:i], func(p OMDbTitle) bool { return p.IMDbID == t.IMDbID }) {
			add("omdb title %s is listed twice", t.IMDbID)
		}
		if t.IMDbRating != nil && (*t.IMDbRating < 0 || *t.IMDbRating > 10) {
			add("omdb title %s imdb_rating %v is outside 0 to 10", t.IMDbID, *t.IMDbRating)
		}
		if t.IMDbVotes != nil && *t.IMDbVotes < 0 {
			add("omdb title %s imdb_votes %d is below zero", t.IMDbID, *t.IMDbVotes)
		}
		for name, score := range map[string]*int{"rotten_tomatoes": t.RottenTomatoes, "metacritic": t.Metacritic} {
			if score != nil && (*score < 0 || *score > 100) {
				add("omdb title %s %s %d is outside 0 to 100", t.IMDbID, name, *score)
			}
		}
	}
}
