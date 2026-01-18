package mock

import (
	"context"

	"github.com/slipstream/slipstream/internal/metadata/omdb"
)

// OMDBClient is a mock implementation of the OMDb client.
type OMDBClient struct{}

// NewOMDBClient creates a new mock OMDb client.
func NewOMDBClient() *OMDBClient {
	return &OMDBClient{}
}

func (c *OMDBClient) Name() string {
	return "omdb-mock"
}

func (c *OMDBClient) IsConfigured() bool {
	return true
}

func (c *OMDBClient) Test(ctx context.Context) error {
	return nil
}

func (c *OMDBClient) GetByIMDbID(ctx context.Context, imdbID string) (*omdb.NormalizedRatings, error) {
	ratings, ok := mockRatings[imdbID]
	if ok {
		return &ratings, nil
	}
	return &defaultRatings, nil
}

var defaultRatings = omdb.NormalizedRatings{
	ImdbRating:     8.0,
	ImdbVotes:      500000,
	RottenTomatoes: 85,
	Metacritic:     75,
	Awards:         "Nominated for 1 Oscar",
}

var mockRatings = map[string]omdb.NormalizedRatings{
	"tt0133093": { // The Matrix
		ImdbRating:     8.7,
		ImdbVotes:      2000000,
		RottenTomatoes: 83,
		Metacritic:     73,
		Awards:         "Won 4 Oscars. 42 wins & 52 nominations total",
	},
	"tt0137523": { // Fight Club
		ImdbRating:     8.8,
		ImdbVotes:      2200000,
		RottenTomatoes: 79,
		Metacritic:     66,
		Awards:         "Nominated for 1 Oscar. 11 wins & 38 nominations total",
	},
	"tt0110912": { // Pulp Fiction
		ImdbRating:     8.9,
		ImdbVotes:      2100000,
		RottenTomatoes: 92,
		Metacritic:     95,
		Awards:         "Won 1 Oscar. 70 wins & 75 nominations total",
	},
	"tt0468569": { // The Dark Knight
		ImdbRating:     9.0,
		ImdbVotes:      2700000,
		RottenTomatoes: 94,
		Metacritic:     84,
		Awards:         "Won 2 Oscars. 159 wins & 163 nominations total",
	},
	"tt0111161": { // The Shawshank Redemption
		ImdbRating:     9.3,
		ImdbVotes:      2800000,
		RottenTomatoes: 90,
		Metacritic:     82,
		Awards:         "Nominated for 7 Oscars. 21 wins & 43 nominations total",
	},
	"tt0068646": { // The Godfather
		ImdbRating:     9.2,
		ImdbVotes:      1900000,
		RottenTomatoes: 97,
		Metacritic:     100,
		Awards:         "Won 3 Oscars. 31 wins & 30 nominations total",
	},
	"tt1375666": { // Inception
		ImdbRating:     8.8,
		ImdbVotes:      2400000,
		RottenTomatoes: 87,
		Metacritic:     74,
		Awards:         "Won 4 Oscars. 157 wins & 220 nominations total",
	},
	"tt0816692": { // Interstellar
		ImdbRating:     8.7,
		ImdbVotes:      1900000,
		RottenTomatoes: 73,
		Metacritic:     74,
		Awards:         "Won 1 Oscar. 44 wins & 148 nominations total",
	},
	"tt1160419": { // Dune (2021)
		ImdbRating:     8.0,
		ImdbVotes:      800000,
		RottenTomatoes: 83,
		Metacritic:     74,
		Awards:         "Won 6 Oscars. 174 wins & 292 nominations total",
	},
	"tt15239678": { // Dune: Part Two
		ImdbRating:     8.6,
		ImdbVotes:      500000,
		RottenTomatoes: 92,
		Metacritic:     79,
		Awards:         "1 win & 4 nominations",
	},
	"tt1517268": { // Barbie
		ImdbRating:     6.8,
		ImdbVotes:      500000,
		RottenTomatoes: 88,
		Metacritic:     80,
		Awards:         "Won 1 Oscar. 98 wins & 334 nominations total",
	},
	"tt15398776": { // Oppenheimer
		ImdbRating:     8.3,
		ImdbVotes:      700000,
		RottenTomatoes: 93,
		Metacritic:     90,
		Awards:         "Won 7 Oscars. 346 wins & 362 nominations total",
	},
	// TV Series
	"tt0944947": { // Game of Thrones
		ImdbRating:     9.2,
		ImdbVotes:      2200000,
		RottenTomatoes: 89,
		Metacritic:     0,
		Awards:         "Won 59 Primetime Emmys. 395 wins & 648 nominations total",
	},
	"tt0903747": { // Breaking Bad
		ImdbRating:     9.5,
		ImdbVotes:      2000000,
		RottenTomatoes: 96,
		Metacritic:     0,
		Awards:         "Won 16 Primetime Emmys. 166 wins & 264 nominations total",
	},
	"tt4574334": { // Stranger Things
		ImdbRating:     8.7,
		ImdbVotes:      1200000,
		RottenTomatoes: 91,
		Metacritic:     0,
		Awards:         "Won 7 Primetime Emmys. 71 wins & 266 nominations total",
	},
	"tt1190634": { // The Boys
		ImdbRating:     8.7,
		ImdbVotes:      600000,
		RottenTomatoes: 93,
		Metacritic:     0,
		Awards:         "Won 1 Primetime Emmy. 16 wins & 81 nominations total",
	},
	"tt10919420": { // Squid Game
		ImdbRating:     8.0,
		ImdbVotes:      500000,
		RottenTomatoes: 95,
		Metacritic:     0,
		Awards:         "Won 6 Primetime Emmys. 65 wins & 79 nominations total",
	},
	"tt3581920": { // The Last of Us
		ImdbRating:     8.7,
		ImdbVotes:      400000,
		RottenTomatoes: 96,
		Metacritic:     0,
		Awards:         "Won 8 Primetime Emmys. 67 wins & 168 nominations total",
	},
}
