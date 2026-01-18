package omdb

// Response represents the OMDb API response.
type Response struct {
	Title      string   `json:"Title"`
	Year       string   `json:"Year"`
	Rated      string   `json:"Rated"`
	Released   string   `json:"Released"`
	Runtime    string   `json:"Runtime"`
	Genre      string   `json:"Genre"`
	Director   string   `json:"Director"`
	Writer     string   `json:"Writer"`
	Actors     string   `json:"Actors"`
	Plot       string   `json:"Plot"`
	Awards     string   `json:"Awards"`
	Poster     string   `json:"Poster"`
	Ratings    []Rating `json:"Ratings"`
	Metascore  string   `json:"Metascore"`
	ImdbRating string   `json:"imdbRating"`
	ImdbVotes  string   `json:"imdbVotes"`
	ImdbID     string   `json:"imdbID"`
	Type       string   `json:"Type"`
	Response   string   `json:"Response"`
	Error      string   `json:"Error,omitempty"`
}

// Rating represents a single rating from a source.
type Rating struct {
	Source string `json:"Source"`
	Value  string `json:"Value"`
}

// NormalizedRatings is the normalized ratings output.
type NormalizedRatings struct {
	ImdbRating     float64 `json:"imdbRating,omitempty"`
	ImdbVotes      int     `json:"imdbVotes,omitempty"`
	RottenTomatoes int     `json:"rottenTomatoes,omitempty"`
	RottenAudience int     `json:"rottenAudience,omitempty"`
	Metacritic     int     `json:"metacritic,omitempty"`
	Awards         string  `json:"awards,omitempty"`
}
