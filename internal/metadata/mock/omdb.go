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

func (c *OMDBClient) GetSeasonEpisodes(ctx context.Context, imdbID string, season int) (map[int]float64, error) {
	ratings, ok := mockEpisodeRatings[imdbID]
	if ok {
		if seasonRatings, ok := ratings[season]; ok {
			return seasonRatings, nil
		}
	}
	// Return default ratings for up to 13 episodes
	result := make(map[int]float64)
	for i := 1; i <= 13; i++ {
		result[i] = 7.5 + float64(i%5)*0.3
	}
	return result, nil
}

// mockEpisodeRatings maps IMDb series ID → season number → episode number → rating
var mockEpisodeRatings = map[string]map[int]map[int]float64{
	"tt0903747": { // Breaking Bad
		1: {1: 9.0, 2: 8.3, 3: 8.1, 4: 8.1, 5: 8.4, 6: 8.7, 7: 9.3},
		2: {1: 8.4, 2: 8.1, 3: 7.9, 4: 8.4, 5: 8.4, 6: 8.5, 7: 8.2, 8: 8.0, 9: 8.9, 10: 8.3, 11: 8.4, 12: 8.5, 13: 9.1},
		3: {1: 8.5, 2: 8.0, 3: 7.8, 4: 7.7, 5: 7.7, 6: 8.1, 7: 8.8, 8: 8.2, 9: 7.8, 10: 8.8, 11: 8.4, 12: 9.0, 13: 9.7},
	},
	"tt0944947": { // Game of Thrones
		1: {1: 9.1, 2: 8.8, 3: 8.7, 4: 8.4, 5: 9.1, 6: 9.2, 7: 9.2, 8: 9.0, 9: 9.6, 10: 9.5},
		2: {1: 8.7, 2: 8.8, 3: 8.6, 4: 8.7, 5: 8.7, 6: 9.0, 7: 8.9, 8: 8.7, 9: 9.7, 10: 9.4},
		3: {1: 8.5, 2: 8.6, 3: 8.5, 4: 9.1, 5: 8.7, 6: 8.6, 7: 8.4, 8: 8.3, 9: 9.9, 10: 9.0},
	},
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
