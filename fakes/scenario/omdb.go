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
	// Type is what OMDb calls the title; empty is a movie.
	Type string `jsonschema:"enum=movie,enum=series" yaml:"type,omitempty"`
	// Seasons are a series' episode ratings, which ?Season= answers.
	Seasons []OMDbSeason `yaml:"seasons,omitempty"`
}

// OMDbSeason is one season of a series as OMDb numbers it.
type OMDbSeason struct {
	Number   int           `jsonschema:"required,minimum=1" yaml:"number"`
	Episodes []OMDbEpisode `yaml:"episodes,omitempty"`
}

// OMDbEpisode is one episode's IMDb rating; a rating left out is one IMDb
// has none of.
type OMDbEpisode struct {
	Episode int      `jsonschema:"required,minimum=1"              yaml:"episode"`
	IMDbID  string   `jsonschema:"required,pattern=^tt[0-9]{7\\,}$" yaml:"imdb_id"`
	Rating  *float64 `jsonschema:"minimum=0,maximum=10"            yaml:"rating,omitempty"`
	Votes   *int64   `jsonschema:"minimum=0"                       yaml:"votes,omitempty"`
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
		if len(t.Seasons) > 0 && t.Type != "series" {
			add("omdb title %s lists seasons and isn't a series", t.IMDbID)
		}
		for name, score := range map[string]*int{"rotten_tomatoes": t.RottenTomatoes, "metacritic": t.Metacritic} {
			if score != nil && (*score < 0 || *score > 100) {
				add("omdb title %s %s %d is outside 0 to 100", t.IMDbID, name, *score)
			}
		}
	}
}
