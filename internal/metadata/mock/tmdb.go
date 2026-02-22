// Package mock provides mock implementations of metadata providers for developer mode.
package mock

import (
	"context"
	"fmt"
	"strings"

	"github.com/slipstream/slipstream/internal/metadata/tmdb"
)

// TMDBClient is a mock implementation of the TMDB client.
type TMDBClient struct{}

// NewTMDBClient creates a new mock TMDB client.
func NewTMDBClient() *TMDBClient {
	return &TMDBClient{}
}

func (c *TMDBClient) Name() string {
	return "tmdb-mock"
}

func (c *TMDBClient) IsConfigured() bool {
	return true
}

func (c *TMDBClient) Test(ctx context.Context) error {
	return nil
}

func (c *TMDBClient) GetImageURL(path, size string) string {
	if path == "" {
		return ""
	}
	return "https://image.tmdb.org/t/p/" + size + path
}

func (c *TMDBClient) SearchMovies(ctx context.Context, query string, year int) ([]tmdb.NormalizedMovieResult, error) {
	query = strings.ToLower(query)
	var results []tmdb.NormalizedMovieResult

	for i := range mockMovies {
		movie := &mockMovies[i]
		if strings.Contains(strings.ToLower(movie.Title), query) {
			if year == 0 || movie.Year == year {
				results = append(results, *movie)
			}
		}
	}

	if len(results) == 0 {
		limit := 10
		if len(mockMovies) < limit {
			limit = len(mockMovies)
		}
		results = mockMovies[:limit]
	}

	return results, nil
}

func (c *TMDBClient) GetMovie(ctx context.Context, id int) (*tmdb.NormalizedMovieResult, error) {
	for i := range mockMovies {
		movie := &mockMovies[i]
		if movie.ID == id {
			return movie, nil
		}
	}
	if len(mockMovies) > 0 {
		return &mockMovies[0], nil
	}
	return nil, tmdb.ErrMovieNotFound
}

func (c *TMDBClient) GetMovieReleaseDates(ctx context.Context, id int) (digital, physical, theatrical string, err error) {
	for i := range mockMovies {
		movie := &mockMovies[i]
		if movie.ID == id {
			return movie.DigitalReleaseDate, movie.PhysicalReleaseDate, movie.ReleaseDate, nil
		}
	}
	return "2024-06-15", "2024-08-20", "2024-03-01", nil
}

func (c *TMDBClient) SearchSeries(ctx context.Context, query string) ([]tmdb.NormalizedSeriesResult, error) {
	query = strings.ToLower(query)
	var results []tmdb.NormalizedSeriesResult

	for i := range mockSeries {
		series := &mockSeries[i]
		if strings.Contains(strings.ToLower(series.Title), query) {
			results = append(results, *series)
		}
	}

	if len(results) == 0 {
		limit := 10
		if len(mockSeries) < limit {
			limit = len(mockSeries)
		}
		results = mockSeries[:limit]
	}

	return results, nil
}

func (c *TMDBClient) GetSeries(ctx context.Context, id int) (*tmdb.NormalizedSeriesResult, error) {
	for i := range mockSeries {
		series := &mockSeries[i]
		if series.ID == id || series.TmdbID == id {
			return series, nil
		}
	}
	if len(mockSeries) > 0 {
		return &mockSeries[0], nil
	}
	return nil, tmdb.ErrSeriesNotFound
}

func (c *TMDBClient) GetAllSeasons(ctx context.Context, seriesID int) ([]tmdb.NormalizedSeasonResult, error) {
	for _, series := range mockSeriesSeasons {
		if series.SeriesID == seriesID {
			return series.Seasons, nil
		}
	}
	return defaultSeasons, nil
}

func (c *TMDBClient) GetMovieCredits(ctx context.Context, id int) (*tmdb.NormalizedCredits, error) {
	credits, ok := mockMovieCredits[id]
	if ok {
		return &credits, nil
	}
	return &defaultMovieCredits, nil
}

func (c *TMDBClient) GetSeriesCredits(ctx context.Context, id int) (*tmdb.NormalizedCredits, error) {
	credits, ok := mockSeriesCredits[id]
	if ok {
		return &credits, nil
	}
	return &defaultSeriesCredits, nil
}

func (c *TMDBClient) GetMovieContentRating(ctx context.Context, id int) (string, error) {
	rating, ok := mockMovieRatings[id]
	if ok {
		return rating, nil
	}
	return "PG-13", nil
}

func (c *TMDBClient) GetSeriesContentRating(ctx context.Context, id int) (string, error) {
	rating, ok := mockSeriesRatings[id]
	if ok {
		return rating, nil
	}
	return "TV-MA", nil
}

func (c *TMDBClient) GetMovieStudio(ctx context.Context, id int) (string, error) {
	studio, ok := mockMovieStudios[id]
	if ok {
		return studio, nil
	}
	return "Warner Bros. Pictures", nil
}

func (c *TMDBClient) GetMovieLogoURL(ctx context.Context, id int) (string, error) {
	return fmt.Sprintf("https://image.tmdb.org/t/p/w500/mock_movie_logo_%d.png", id), nil
}

func (c *TMDBClient) GetSeriesLogoURL(ctx context.Context, id int) (string, error) {
	return fmt.Sprintf("https://image.tmdb.org/t/p/w500/mock_series_logo_%d.png", id), nil
}

func (c *TMDBClient) GetMovieTrailerURL(ctx context.Context, id int) (string, error) {
	return fmt.Sprintf("https://www.youtube.com/watch?v=mock_movie_%d", id), nil
}

func (c *TMDBClient) GetSeriesTrailerURL(ctx context.Context, id int) (string, error) {
	return fmt.Sprintf("https://www.youtube.com/watch?v=mock_series_%d", id), nil
}

var defaultMovieCredits = tmdb.NormalizedCredits{
	Directors: []tmdb.NormalizedPerson{{ID: 1, Name: "Christopher Nolan", PhotoURL: "https://image.tmdb.org/t/p/w185/xuAIuYSmsUzKlUMBFGVZaWsY3DZ.jpg"}},
	Writers:   []tmdb.NormalizedPerson{{ID: 2, Name: "Jonathan Nolan", Role: "Screenplay", PhotoURL: "https://image.tmdb.org/t/p/w185/dummy.jpg"}},
	Cast: []tmdb.NormalizedPerson{
		{ID: 3, Name: "Leonardo DiCaprio", Role: "Dom Cobb", PhotoURL: "https://image.tmdb.org/t/p/w185/wo2hJpn04vbtmh0B9utCFdsQhxM.jpg"},
		{ID: 4, Name: "Joseph Gordon-Levitt", Role: "Arthur", PhotoURL: "https://image.tmdb.org/t/p/w185/zvwJpU44vs1FfkBpCf5chCRfJo8.jpg"},
		{ID: 5, Name: "Elliot Page", Role: "Ariadne", PhotoURL: "https://image.tmdb.org/t/p/w185/dummy.jpg"},
	},
}

var defaultSeriesCredits = tmdb.NormalizedCredits{
	Creators: []tmdb.NormalizedPerson{{ID: 1, Name: "Vince Gilligan", PhotoURL: "https://image.tmdb.org/t/p/w185/wSTvJGz7QbJf1HK2Mv1Cev6W9TV.jpg"}},
	Cast: []tmdb.NormalizedPerson{
		{ID: 2, Name: "Bryan Cranston", Role: "Walter White", PhotoURL: "https://image.tmdb.org/t/p/w185/7Jahy5LZX2Fo8fGJltMreAI49hC.jpg"},
		{ID: 3, Name: "Aaron Paul", Role: "Jesse Pinkman", PhotoURL: "https://image.tmdb.org/t/p/w185/8Kce1FGpuSqnYasp5ahXVYFqGPn.jpg"},
		{ID: 4, Name: "Anna Gunn", Role: "Skyler White", PhotoURL: "https://image.tmdb.org/t/p/w185/dummy.jpg"},
	},
}

var mockMovieCredits = map[int]tmdb.NormalizedCredits{
	603: { // The Matrix
		Directors: []tmdb.NormalizedPerson{{ID: 9340, Name: "Lana Wachowski"}, {ID: 9339, Name: "Lilly Wachowski"}},
		Writers:   []tmdb.NormalizedPerson{{ID: 9340, Name: "Lana Wachowski", Role: "Screenplay"}, {ID: 9339, Name: "Lilly Wachowski", Role: "Screenplay"}},
		Cast: []tmdb.NormalizedPerson{
			{ID: 6384, Name: "Keanu Reeves", Role: "Neo", PhotoURL: "https://image.tmdb.org/t/p/w185/4D0PpNI0kmP58hgrwGC3wCjxhnm.jpg"},
			{ID: 2975, Name: "Laurence Fishburne", Role: "Morpheus", PhotoURL: "https://image.tmdb.org/t/p/w185/8suOhUmPbfKqDQ17jQ1Gy0mI3P4.jpg"},
			{ID: 530, Name: "Carrie-Anne Moss", Role: "Trinity", PhotoURL: "https://image.tmdb.org/t/p/w185/xD4jTA3KmVp5Rq3aHcymL9DwWl7.jpg"},
		},
	},
	27205: { // Inception
		Directors: []tmdb.NormalizedPerson{{ID: 525, Name: "Christopher Nolan", PhotoURL: "https://image.tmdb.org/t/p/w185/xuAIuYSmsUzKlUMBFGVZaWsY3DZ.jpg"}},
		Writers:   []tmdb.NormalizedPerson{{ID: 525, Name: "Christopher Nolan", Role: "Writer"}},
		Cast: []tmdb.NormalizedPerson{
			{ID: 6193, Name: "Leonardo DiCaprio", Role: "Dom Cobb", PhotoURL: "https://image.tmdb.org/t/p/w185/wo2hJpn04vbtmh0B9utCFdsQhxM.jpg"},
			{ID: 24045, Name: "Joseph Gordon-Levitt", Role: "Arthur", PhotoURL: "https://image.tmdb.org/t/p/w185/zvwJpU44vs1FfkBpCf5chCRfJo8.jpg"},
			{ID: 27578, Name: "Elliot Page", Role: "Ariadne", PhotoURL: "https://image.tmdb.org/t/p/w185/cJACXMKx7IKfDy4gfVKBfYxHvD.jpg"},
			{ID: 2524, Name: "Tom Hardy", Role: "Eames", PhotoURL: "https://image.tmdb.org/t/p/w185/sGMA6pA2D6X0gun49igJT3piHs3.jpg"},
		},
	},
}

var mockSeriesCredits = map[int]tmdb.NormalizedCredits{
	1396: { // Breaking Bad
		Creators: []tmdb.NormalizedPerson{{ID: 17419, Name: "Vince Gilligan", PhotoURL: "https://image.tmdb.org/t/p/w185/wSTvJGz7QbJf1HK2Mv1Cev6W9TV.jpg"}},
		Cast: []tmdb.NormalizedPerson{
			{ID: 17419, Name: "Bryan Cranston", Role: "Walter White", PhotoURL: "https://image.tmdb.org/t/p/w185/7Jahy5LZX2Fo8fGJltMreAI49hC.jpg"},
			{ID: 84497, Name: "Aaron Paul", Role: "Jesse Pinkman", PhotoURL: "https://image.tmdb.org/t/p/w185/8Kce1FGpuSqnYasp5ahXVYFqGPn.jpg"},
			{ID: 134531, Name: "Anna Gunn", Role: "Skyler White", PhotoURL: "https://image.tmdb.org/t/p/w185/lKlGjfmu8M8jPGp1wT2BWsSwXhk.jpg"},
		},
	},
	1399: { // Game of Thrones
		Creators: []tmdb.NormalizedPerson{
			{ID: 9813, Name: "David Benioff"},
			{ID: 228068, Name: "D. B. Weiss"},
		},
		Cast: []tmdb.NormalizedPerson{
			{ID: 239019, Name: "Kit Harington", Role: "Jon Snow", PhotoURL: "https://image.tmdb.org/t/p/w185/4MqUjb1SYrzHmFSyGiXnlZWG3X0.jpg"},
			{ID: 1223786, Name: "Emilia Clarke", Role: "Daenerys Targaryen", PhotoURL: "https://image.tmdb.org/t/p/w185/j7d083zIMhwnKro3tQqDz3XKZWJ.jpg"},
			{ID: 22970, Name: "Peter Dinklage", Role: "Tyrion Lannister", PhotoURL: "https://image.tmdb.org/t/p/w185/9CAd9YV9I1Qfpyhjfo1khtiJGqK.jpg"},
		},
	},
}

var mockMovieRatings = map[int]string{
	603:    "R",     // The Matrix
	550:    "R",     // Fight Club
	680:    "R",     // Pulp Fiction
	155:    "PG-13", // The Dark Knight
	278:    "R",     // The Shawshank Redemption
	238:    "R",     // The Godfather
	27205:  "PG-13", // Inception
	157336: "PG-13", // Interstellar
	438631: "PG-13", // Dune
	693134: "PG-13", // Dune: Part Two
	346698: "PG-13", // Barbie
	872585: "R",     // Oppenheimer
}

var mockSeriesRatings = map[int]string{
	1399:   "TV-MA", // Game of Thrones
	1396:   "TV-MA", // Breaking Bad
	66732:  "TV-14", // Stranger Things
	94997:  "TV-MA", // House of the Dragon
	76479:  "TV-MA", // The Boys
	93405:  "TV-MA", // Squid Game
	100088: "TV-MA", // The Last of Us
	82856:  "TV-PG", // The Mandalorian
}

var mockMovieStudios = map[int]string{
	603:    "Warner Bros. Pictures",     // The Matrix
	550:    "Fox 2000 Pictures",         // Fight Club
	680:    "Miramax",                   // Pulp Fiction
	155:    "Warner Bros. Pictures",     // The Dark Knight
	278:    "Castle Rock Entertainment", // The Shawshank Redemption
	238:    "Paramount Pictures",        // The Godfather
	27205:  "Warner Bros. Pictures",     // Inception
	157336: "Paramount Pictures",        // Interstellar
	438631: "Legendary Pictures",        // Dune
	693134: "Legendary Pictures",        // Dune: Part Two
	346698: "Warner Bros. Pictures",     // Barbie
	872585: "Universal Pictures",        // Oppenheimer
}

type mockSeriesWithSeasons struct {
	SeriesID int
	Seasons  []tmdb.NormalizedSeasonResult
}

var defaultSeasons = []tmdb.NormalizedSeasonResult{
	{SeasonNumber: 1, Name: "Season 1", Overview: "The first season.", AirDate: "2020-01-01", Episodes: []tmdb.NormalizedEpisodeResult{
		{EpisodeNumber: 1, SeasonNumber: 1, Title: "Episode 1", Overview: "The series premiere.", AirDate: "2020-01-01", Runtime: 45},
		{EpisodeNumber: 2, SeasonNumber: 1, Title: "Episode 2", Overview: "The second episode.", AirDate: "2020-01-08", Runtime: 45},
		{EpisodeNumber: 3, SeasonNumber: 1, Title: "Episode 3", Overview: "The third episode.", AirDate: "2020-01-15", Runtime: 45},
	}},
	{SeasonNumber: 2, Name: "Season 2", Overview: "The second season.", AirDate: "2021-01-01", Episodes: []tmdb.NormalizedEpisodeResult{
		{EpisodeNumber: 1, SeasonNumber: 2, Title: "Episode 1", Overview: "The season premiere.", AirDate: "2021-01-01", Runtime: 45},
		{EpisodeNumber: 2, SeasonNumber: 2, Title: "Episode 2", Overview: "The second episode.", AirDate: "2021-01-08", Runtime: 45},
		{EpisodeNumber: 3, SeasonNumber: 2, Title: "Episode 3", Overview: "The third episode.", AirDate: "2021-01-15", Runtime: 45},
	}},
}

// Real TMDB data fetched from API - DO NOT EDIT BELOW THIS LINE
var mockMovies = []tmdb.NormalizedMovieResult{
	{ID: 603, Title: "The Matrix", Year: 1999, Overview: "Set in the 22nd century, The Matrix tells the story of a computer hacker who joins a group of underground insurgents fighting the vast and powerful...", PosterURL: "https://image.tmdb.org/t/p/w500/p96dm7sCMn4VYAStA6siNz30G1r.jpg", BackdropURL: "https://image.tmdb.org/t/p/w780/tlm8UkiQsitc8rSuIAscQDCnP8d.jpg", ImdbID: "tt0133093", Genres: []string{"Action", "Science Fiction"}, Runtime: 136, ReleaseDate: "1999-03-31", DigitalReleaseDate: "", PhysicalReleaseDate: "2018-05-22"},
	{ID: 550, Title: "Fight Club", Year: 1999, Overview: "A ticking-time-bomb insomniac and a slippery soap salesman channel primal male aggression into a shocking new form of therapy. Their concept catche...", PosterURL: "https://image.tmdb.org/t/p/w500/pB8BM7pdSp6B6Ih7QZ4DrQ3PmJK.jpg", BackdropURL: "https://image.tmdb.org/t/p/w780/5TiwfWEaPSwD20uwXjCTUqpQX70.jpg", ImdbID: "tt0137523", Genres: []string{"Drama", "Thriller"}, Runtime: 139, ReleaseDate: "1999-10-15", DigitalReleaseDate: "", PhysicalReleaseDate: "2000-04-25"},
	{ID: 680, Title: "Pulp Fiction", Year: 1994, Overview: "A burger-loving hit man, his philosophical partner, a drug-addled gangster's moll and a washed-up boxer converge in this sprawling, comedic crime c...", PosterURL: "https://image.tmdb.org/t/p/w500/vQWk5YBFWF4bZaofAbv0tShwBvQ.jpg", BackdropURL: "https://image.tmdb.org/t/p/w780/96hiUXEuYsu4tcnvlaY8tEMFM0m.jpg", ImdbID: "tt0110912", Genres: []string{"Thriller", "Crime", "Comedy"}, Runtime: 154, ReleaseDate: "1994-09-10", DigitalReleaseDate: "", PhysicalReleaseDate: "2022-12-06"},
	{ID: 155, Title: "The Dark Knight", Year: 2008, Overview: "Batman raises the stakes in his war on crime. With the help of Lt. Jim Gordon and District Attorney Harvey Dent, Batman sets out to dismantle the r...", PosterURL: "https://image.tmdb.org/t/p/w500/qJ2tW6WMUDux911r6m7haRef0WH.jpg", BackdropURL: "https://image.tmdb.org/t/p/w780/cfT29Im5VDvjE0RpyKOSdCKZal7.jpg", ImdbID: "tt0468569", Genres: []string{"Drama", "Action", "Crime", "Thriller"}, Runtime: 152, ReleaseDate: "2008-07-16", DigitalReleaseDate: "", PhysicalReleaseDate: "2008-12-09"},
	{ID: 278, Title: "The Shawshank Redemption", Year: 1994, Overview: "Imprisoned in the 1940s for the double murder of his wife and her lover, upstanding banker Andy Dufresne begins a new life at the Shawshank prison,...", PosterURL: "https://image.tmdb.org/t/p/w500/9cqNxx0GxF0bflZmeSMuL5tnGzr.jpg", BackdropURL: "https://image.tmdb.org/t/p/w780/zfbjgQE1uSd9wiPTX4VzsLi0rGG.jpg", ImdbID: "tt0111161", Genres: []string{"Drama", "Crime"}, Runtime: 142, ReleaseDate: "1994-09-23", DigitalReleaseDate: "", PhysicalReleaseDate: ""},
	{ID: 238, Title: "The Godfather", Year: 1972, Overview: "Spanning the years 1945 to 1955, a chronicle of the fictional Italian-American Corleone crime family. When organized crime family patriarch, Vito C...", PosterURL: "https://image.tmdb.org/t/p/w500/3bhkrj58Vtu7enYsRolD1fZdja1.jpg", BackdropURL: "https://image.tmdb.org/t/p/w780/tmU7GeKVybMWFButWEGl2M4GeiP.jpg", ImdbID: "tt0068646", Genres: []string{"Drama", "Crime"}, Runtime: 175, ReleaseDate: "1972-03-14", DigitalReleaseDate: "", PhysicalReleaseDate: "2022-03-22"},
	{ID: 27205, Title: "Inception", Year: 2010, Overview: "Cobb, a skilled thief who commits corporate espionage by infiltrating the subconscious of his targets is offered a chance to regain his old life as...", PosterURL: "https://image.tmdb.org/t/p/w500/xlaY2zyzMfkhk0HSC5VUwzoZPU1.jpg", BackdropURL: "https://image.tmdb.org/t/p/w780/ii8QGacT3MXESqBckQlyrATY0lT.jpg", ImdbID: "tt1375666", Genres: []string{"Action", "Science Fiction", "Adventure"}, Runtime: 148, ReleaseDate: "2010-07-15", DigitalReleaseDate: "", PhysicalReleaseDate: "2010-12-07"},
	{ID: 157336, Title: "Interstellar", Year: 2014, Overview: "The adventures of a group of explorers who make use of a newly discovered wormhole to surpass the limitations on human space travel and conquer the...", PosterURL: "https://image.tmdb.org/t/p/w500/gEU2QniE6E77NI6lCU6MxlNBvIx.jpg", BackdropURL: "https://image.tmdb.org/t/p/w780/5XNQBqnBwPA9yT0jZ0p3s8bbLh0.jpg", ImdbID: "tt0816692", Genres: []string{"Adventure", "Drama", "Science Fiction"}, Runtime: 169, ReleaseDate: "2014-11-05", DigitalReleaseDate: "", PhysicalReleaseDate: "2015-03-31"},
	{ID: 120, Title: "The Lord of the Rings: The Fellowship of the Ring", Year: 2001, Overview: "Young hobbit Frodo Baggins, after inheriting a mysterious ring from his uncle Bilbo, must leave his home in order to keep it from falling into the ...", PosterURL: "https://image.tmdb.org/t/p/w500/6oom5QYQ2yQTMJIbnvbkBL9cHo6.jpg", BackdropURL: "https://image.tmdb.org/t/p/w780/a0lfia8tk8ifkrve0Tn8wkISUvs.jpg", ImdbID: "tt0120737", Genres: []string{"Adventure", "Fantasy", "Action"}, Runtime: 179, ReleaseDate: "2001-12-18", DigitalReleaseDate: "", PhysicalReleaseDate: "2002-08-06"},
	{ID: 24428, Title: "The Avengers", Year: 2012, Overview: "When an unexpected enemy emerges and threatens global safety and security, Nick Fury, director of the international peacekeeping agency known as S....", PosterURL: "https://image.tmdb.org/t/p/w500/RYMX2wcKCBAr24UyPD7xwmjaTn.jpg", BackdropURL: "https://image.tmdb.org/t/p/w780/9BBTo63ANSmhC4e6r62OJFuK2GL.jpg", ImdbID: "tt0848228", Genres: []string{"Science Fiction", "Action", "Adventure"}, Runtime: 143, ReleaseDate: "2012-04-25", DigitalReleaseDate: "2024-06-14", PhysicalReleaseDate: ""},
	{ID: 299536, Title: "Avengers: Infinity War", Year: 2018, Overview: "As the Avengers and their allies have continued to protect the world from threats too large for any one hero to handle, a new danger has emerged fr...", PosterURL: "https://image.tmdb.org/t/p/w500/7WsyChQLEftFiDOVTGkv3hFpyyt.jpg", BackdropURL: "https://image.tmdb.org/t/p/w780/mDfJG3LC3Dqb67AZ52x3Z0jU0uB.jpg", ImdbID: "tt4154756", Genres: []string{"Adventure", "Action", "Science Fiction"}, Runtime: 149, ReleaseDate: "2018-04-25", DigitalReleaseDate: "2018-07-31", PhysicalReleaseDate: "2018-08-14"},
	{ID: 299534, Title: "Avengers: Endgame", Year: 2019, Overview: "After the devastating events of Avengers: Infinity War, the universe is in ruins due to the efforts of the Mad Titan, Thanos. With the help of rema...", PosterURL: "https://image.tmdb.org/t/p/w500/bR8ISy1O9XQxqiy0fQFw2BX72RQ.jpg", BackdropURL: "https://image.tmdb.org/t/p/w780/7RyHsO4yDXtBv1zUU3mTpHeQ0d5.jpg", ImdbID: "tt4154796", Genres: []string{"Adventure", "Science Fiction", "Action"}, Runtime: 181, ReleaseDate: "2019-04-24", DigitalReleaseDate: "2019-07-30", PhysicalReleaseDate: "2019-08-13"},
	{ID: 569094, Title: "Spider-Man: Across the Spider-Verse", Year: 2023, Overview: "After reuniting with Gwen Stacy, Brooklyn’s full-time, friendly neighborhood Spider-Man is catapulted across the Multiverse, where he encounters ...", PosterURL: "https://image.tmdb.org/t/p/w500/8Vt6mWEReuy4Of61Lnj5Xj704m8.jpg", BackdropURL: "https://image.tmdb.org/t/p/w780/9xfDWXAUbFXQK585JvByT5pEAhe.jpg", ImdbID: "tt9362722", Genres: []string{"Animation", "Action", "Adventure", "Science Fiction"}, Runtime: 140, ReleaseDate: "2023-05-31", DigitalReleaseDate: "2023-08-08", PhysicalReleaseDate: "2023-09-05"},
	{ID: 438631, Title: "Dune", Year: 2021, Overview: "Paul Atreides, a brilliant and gifted young man born into a great destiny beyond his understanding, must travel to the most dangerous planet in the...", PosterURL: "https://image.tmdb.org/t/p/w500/d5NXSklXo0qyIYkgV94XAgMIckC.jpg", BackdropURL: "https://image.tmdb.org/t/p/w780/jYEW5xZkZk2WTrdbMGAPFuBqbDc.jpg", ImdbID: "tt1160419", Genres: []string{"Science Fiction", "Adventure"}, Runtime: 155, ReleaseDate: "2021-09-15", DigitalReleaseDate: "2021-10-21", PhysicalReleaseDate: "2022-01-11"},
	{ID: 693134, Title: "Dune: Part Two", Year: 2024, Overview: "Follow the mythic journey of Paul Atreides as he unites with Chani and the Fremen while on a path of revenge against the conspirators who destroyed...", PosterURL: "https://image.tmdb.org/t/p/w500/1pdfLvkbY9ohJlCjQH2CZjjYVvJ.jpg", BackdropURL: "https://image.tmdb.org/t/p/w780/xOMo8BRK7PfcJv9JCnx7s5hj0PX.jpg", ImdbID: "tt15239678", Genres: []string{"Science Fiction", "Adventure"}, Runtime: 167, ReleaseDate: "2024-02-27", DigitalReleaseDate: "2024-04-16", PhysicalReleaseDate: "2024-05-14"},
	{ID: 359724, Title: "Ford v Ferrari", Year: 2019, Overview: "American car designer Carroll Shelby and the British-born driver Ken Miles work together to battle corporate interference, the laws of physics, and...", PosterURL: "https://image.tmdb.org/t/p/w500/dR1Ju50iudrOh3YgfwkAU1g2HZe.jpg", BackdropURL: "https://image.tmdb.org/t/p/w780/2vq5GTJOahE03mNYZGxIynlHcWr.jpg", ImdbID: "tt1950186", Genres: []string{"Drama", "Action", "History"}, Runtime: 153, ReleaseDate: "2019-11-13", DigitalReleaseDate: "2020-01-28", PhysicalReleaseDate: ""},
	{ID: 346698, Title: "Barbie", Year: 2023, Overview: "Barbie and Ken are having the time of their lives in the colorful and seemingly perfect world of Barbie Land. However, when they get a chance to go...", PosterURL: "https://image.tmdb.org/t/p/w500/iuFNMS8U5cb6xfzi51Dbkovj7vM.jpg", BackdropURL: "https://image.tmdb.org/t/p/w780/ctMserH8g2SeOAnCw5gFjdQF8mo.jpg", ImdbID: "tt1517268", Genres: []string{"Comedy", "Adventure"}, Runtime: 114, ReleaseDate: "2023-07-19", DigitalReleaseDate: "2023-09-12", PhysicalReleaseDate: "2023-10-17"},
	{ID: 872585, Title: "Oppenheimer", Year: 2023, Overview: "The story of J. Robert Oppenheimer's role in the development of the atomic bomb during World War II.", PosterURL: "https://image.tmdb.org/t/p/w500/8Gxv8gSFCU0XGDykEGv7zR1n2ua.jpg", BackdropURL: "https://image.tmdb.org/t/p/w780/7CENyUim29IEsaJhUxIGymCRvPu.jpg", ImdbID: "tt15398776", Genres: []string{"Drama", "History"}, Runtime: 181, ReleaseDate: "2023-07-19", DigitalReleaseDate: "2023-11-21", PhysicalReleaseDate: "2023-11-21"},
	{ID: 76600, Title: "Avatar: The Way of Water", Year: 2022, Overview: "Set more than a decade after the events of the first film, learn the story of the Sully family (Jake, Neytiri, and their kids), the trouble that fo...", PosterURL: "https://image.tmdb.org/t/p/w500/t6HIqrRAclMCA60NsSmeqe9RmNV.jpg", BackdropURL: "https://image.tmdb.org/t/p/w780/cd8YDn7M0lfaHhZdU6MvCDxPalP.jpg", ImdbID: "tt1630029", Genres: []string{"Action", "Adventure", "Science Fiction"}, Runtime: 192, ReleaseDate: "2022-12-14", DigitalReleaseDate: "2023-03-28", PhysicalReleaseDate: "2023-06-20"},
	{ID: 19995, Title: "Avatar", Year: 2009, Overview: "In the 22nd century, a paraplegic Marine is dispatched to the moon Pandora on a unique mission, but becomes torn between following orders and prote...", PosterURL: "https://image.tmdb.org/t/p/w500/gKY6q7SjCkAU6FqvqWybDYgUKIF.jpg", BackdropURL: "https://image.tmdb.org/t/p/w780/vL5LR6WdxWPjLPFRLe133jXWsh5.jpg", ImdbID: "tt0499549", Genres: []string{"Action", "Adventure", "Fantasy", "Science Fiction"}, Runtime: 162, ReleaseDate: "2009-12-16", DigitalReleaseDate: "", PhysicalReleaseDate: "2010-04-22"},
	{ID: 533535, Title: "Deadpool & Wolverine", Year: 2024, Overview: "A listless Wade Wilson toils away in civilian life with his days as the morally flexible mercenary, Deadpool, behind him. But when his homeworld fa...", PosterURL: "https://image.tmdb.org/t/p/w500/8cdWjvZQUExUUTzyp4t6EDMubfO.jpg", BackdropURL: "https://image.tmdb.org/t/p/w780/ufpeVEM64uZHPpzzeiDNIAdaeOD.jpg", ImdbID: "tt6263850", Genres: []string{"Action", "Comedy", "Science Fiction"}, Runtime: 128, ReleaseDate: "2024-07-24", DigitalReleaseDate: "2024-10-01", PhysicalReleaseDate: "2024-10-22"},
	{ID: 545611, Title: "Everything Everywhere All at Once", Year: 2022, Overview: "An aging Chinese immigrant is swept up in an insane adventure, where she alone can save what's important to her by connecting with the lives she co...", PosterURL: "https://image.tmdb.org/t/p/w500/u68AjlvlutfEIcpmbYpKcdi09ut.jpg", BackdropURL: "https://image.tmdb.org/t/p/w780/ss0Os3uWJfQAENILHZUdX8Tt1OC.jpg", ImdbID: "tt6710474", Genres: []string{"Action", "Adventure", "Science Fiction"}, Runtime: 140, ReleaseDate: "2022-03-24", DigitalReleaseDate: "2022-06-07", PhysicalReleaseDate: "2022-07-05"},
	{ID: 447365, Title: "Guardians of the Galaxy Vol. 3", Year: 2023, Overview: "Peter Quill, still reeling from the loss of Gamora, must rally his team around him to defend the universe along with protecting one of their own. A...", PosterURL: "https://image.tmdb.org/t/p/w500/r2J02Z2OpNTctfOSN1Ydgii51I3.jpg", BackdropURL: "https://image.tmdb.org/t/p/w780/hoVj2lYW3i7oMd1o7bPQRZd1lk1.jpg", ImdbID: "tt6791350", Genres: []string{"Science Fiction", "Adventure", "Action"}, Runtime: 150, ReleaseDate: "2023-05-03", DigitalReleaseDate: "2023-08-02", PhysicalReleaseDate: "2023-08-01"},
	{ID: 912649, Title: "Venom: The Last Dance", Year: 2024, Overview: "Eddie and Venom are on the run. Hunted by both of their worlds and with the net closing in, the duo are forced into a devastating decision that wil...", PosterURL: "https://image.tmdb.org/t/p/w500/1RaSkWakWBxxYOWRrqmwo2my5zg.jpg", BackdropURL: "https://image.tmdb.org/t/p/w780/3V4kLQg0kSqPLctI5ziYWabAZYF.jpg", ImdbID: "tt16366836", Genres: []string{"Action", "Science Fiction", "Adventure"}, Runtime: 109, ReleaseDate: "2024-10-22", DigitalReleaseDate: "2024-12-10", PhysicalReleaseDate: "2025-01-21"},
	{ID: 1022789, Title: "Inside Out 2", Year: 2024, Overview: "Teenager Riley's mind headquarters is undergoing a sudden demolition to make room for something entirely unexpected: new Emotions! Joy, Sadness, An...", PosterURL: "https://image.tmdb.org/t/p/w500/vpnVM9B6NMmQpWeZvzLvDESb2QY.jpg", BackdropURL: "https://image.tmdb.org/t/p/w780/p5ozvmdgsmbWe0H8Xk7Rc8SCwAB.jpg", ImdbID: "tt22022452", Genres: []string{"Animation", "Adventure", "Comedy", "Family"}, Runtime: 97, ReleaseDate: "2024-06-11", DigitalReleaseDate: "2024-08-20", PhysicalReleaseDate: "2024-09-10"},
}

// TMDB Series - Copy this to mock/tmdb.go
var mockSeries = []tmdb.NormalizedSeriesResult{
	{ID: 1399, TmdbID: 1399, Title: "Game of Thrones", Year: 2011, Overview: "Seven noble families fight for control of the mythical land of Westeros. Friction between the houses leads to full-scale war. All while a very anci...", PosterURL: "https://image.tmdb.org/t/p/w500/1XS1oqL89opfnbLl8WnZY1O1uJx.jpg", BackdropURL: "https://image.tmdb.org/t/p/w780/zZqpAXxVSBtxV9qPBcscfXBcL2w.jpg", ImdbID: "tt0944947", TvdbID: 121361, Genres: []string{"Sci-Fi & Fantasy", "Drama", "Action & Adventure"}, Status: "ended", Runtime: 0, Network: "HBO"},
	{ID: 1396, TmdbID: 1396, Title: "Breaking Bad", Year: 2008, Overview: "Walter White, a New Mexico chemistry teacher, is diagnosed with Stage III cancer and given a prognosis of only two years left to live. He becomes f...", PosterURL: "https://image.tmdb.org/t/p/w500/ztkUQFLlC19CCMYHW9o1zWhJRNq.jpg", BackdropURL: "https://image.tmdb.org/t/p/w780/vFxjuhENDjEKzWXUGKmRFct15bA.jpg", ImdbID: "tt0903747", TvdbID: 81189, Genres: []string{"Drama", "Crime"}, Status: "ended", Runtime: 0, Network: "AMC"},
	{ID: 66732, TmdbID: 66732, Title: "Stranger Things", Year: 2016, Overview: "When a young boy vanishes, a small town uncovers a mystery involving secret experiments, terrifying supernatural forces, and one strange little girl.", PosterURL: "https://image.tmdb.org/t/p/w500/uOOtwVbSr4QDjAGIifLDwpb2Pdl.jpg", BackdropURL: "https://image.tmdb.org/t/p/w780/56v2KjBlU4XaOv9rVYEQypROD7P.jpg", ImdbID: "tt4574334", TvdbID: 305288, Genres: []string{"Sci-Fi & Fantasy", "Mystery", "Action & Adventure"}, Status: "ended", Runtime: 0, Network: "Netflix"},
	{ID: 94997, TmdbID: 94997, Title: "House of the Dragon", Year: 2022, Overview: "The Targaryen dynasty is at the absolute apex of its power, with more than 15 dragons under their yoke. Most empires crumble from such heights. In ...", PosterURL: "https://image.tmdb.org/t/p/w500/7QMsOTMUswlwxJP0rTTZfmz2tX2.jpg", BackdropURL: "https://image.tmdb.org/t/p/w780/2xGcSLyTAzConiHAByWqhfLiatT.jpg", ImdbID: "tt11198330", TvdbID: 371572, Genres: []string{"Sci-Fi & Fantasy", "Drama", "Action & Adventure"}, Status: "continuing", Runtime: 0, Network: "HBO"},
	{ID: 60735, TmdbID: 60735, Title: "The Flash", Year: 2014, Overview: "After being struck by lightning, CSI investigator Barry Allen awakens from a nine-month coma to discover he has been  granted the gift of super spe...", PosterURL: "https://image.tmdb.org/t/p/w500/yZevl2vHQgmosfwUdVNzviIfaWS.jpg", BackdropURL: "https://image.tmdb.org/t/p/w780/gFkHcIh7iE5G0oVOgpmY8ONQjhl.jpg", ImdbID: "tt3107288", TvdbID: 279121, Genres: []string{"Drama", "Sci-Fi & Fantasy"}, Status: "ended", Runtime: 0, Network: "The CW"},
	{ID: 84958, TmdbID: 84958, Title: "Loki", Year: 2021, Overview: "After stealing the Tesseract during the events of “Avengers: Endgame,” an alternate version of Loki is brought to the mysterious Time Variance ...", PosterURL: "https://image.tmdb.org/t/p/w500/oJdVHUYrjdS2IqiNztVIP4GPB1p.jpg", BackdropURL: "https://image.tmdb.org/t/p/w780/N1hWzVPpZ8lIQvQskgdQogxdsc.jpg", ImdbID: "tt9140554", TvdbID: 362472, Genres: []string{"Drama", "Sci-Fi & Fantasy"}, Status: "ended", Runtime: 0, Network: "Disney+"},
	{ID: 1418, TmdbID: 1418, Title: "The Big Bang Theory", Year: 2007, Overview: "Physicists Leonard and Sheldon find their nerd-centric social circle with pals Howard and Raj expanding when aspiring actress Penny moves in next d...", PosterURL: "https://image.tmdb.org/t/p/w500/ooBGRQBdbGzBxAVfExiO8r7kloA.jpg", BackdropURL: "https://image.tmdb.org/t/p/w780/7RySzFeK3LPVMXcPtqfZnl6u4p1.jpg", ImdbID: "tt0898266", TvdbID: 80379, Genres: []string{"Comedy"}, Status: "ended", Runtime: 22, Network: "CBS"},
	{ID: 1100, TmdbID: 1100, Title: "How I Met Your Mother", Year: 2005, Overview: "A father recounts to his children - through a series of flashbacks - the journey he and his four best friends took leading up to him meeting their ...", PosterURL: "https://image.tmdb.org/t/p/w500/b34jPzmB0wZy7EjUZoleXOl2RRI.jpg", BackdropURL: "https://image.tmdb.org/t/p/w780/gvEisYtZ0iBMjnO3zqFU2oM26oM.jpg", ImdbID: "tt0460649", TvdbID: 75760, Genres: []string{"Comedy"}, Status: "ended", Runtime: 22, Network: "CBS"},
	{ID: 456, TmdbID: 456, Title: "The Simpsons", Year: 1989, Overview: "Set in Springfield, the average American town, the show focuses on the antics and everyday adventures of the Simpson family; Homer, Marge, Bart, Li...", PosterURL: "https://image.tmdb.org/t/p/w500/uWpG7GqfKGQqX4YMAo3nv5OrglV.jpg", BackdropURL: "https://image.tmdb.org/t/p/w780/jvTeRgjFsp66xj8SWxhr7O2J4ud.jpg", ImdbID: "tt0096697", TvdbID: 71663, Genres: []string{"Family", "Animation", "Comedy"}, Status: "continuing", Runtime: 22, Network: "FOX"},
	{ID: 2190, TmdbID: 2190, Title: "South Park", Year: 1997, Overview: "Follow the misadventures of four irreverent grade-schoolers in the quiet, dysfunctional town of South Park, Colorado.", PosterURL: "https://image.tmdb.org/t/p/w500/1CGwZCFX2qerXaXQJJUB3qUvxq7.jpg", BackdropURL: "https://image.tmdb.org/t/p/w780/3UviYOlhn8EgXMBuiT6MnUuo1w9.jpg", ImdbID: "tt0121955", TvdbID: 75897, Genres: []string{"Animation", "Comedy"}, Status: "continuing", Runtime: 0, Network: "Comedy Central"},
	{ID: 1668, TmdbID: 1668, Title: "Friends", Year: 1994, Overview: "Six young people from New York City, on their own and struggling to survive in the real world, find the companionship, comfort and support they get...", PosterURL: "https://image.tmdb.org/t/p/w500/f496cm9enuEsZkSPzCwnTESEK5s.jpg", BackdropURL: "https://image.tmdb.org/t/p/w780/l0qVZIpXtIo7km9u5Yqh0nKPOr5.jpg", ImdbID: "tt0108778", TvdbID: 79168, Genres: []string{"Comedy"}, Status: "ended", Runtime: 0, Network: "NBC"},
	{ID: 1402, TmdbID: 1402, Title: "The Walking Dead", Year: 2010, Overview: "Sheriff's deputy Rick Grimes awakens from a coma to find a post-apocalyptic world dominated by flesh-eating zombies. He sets out to find his family...", PosterURL: "https://image.tmdb.org/t/p/w500/ng3cMtxYKt1OSQYqFlnKWnVsqNO.jpg", BackdropURL: "https://image.tmdb.org/t/p/w780/rAOjnEFTuNysY7bot8zonhImGMh.jpg", ImdbID: "tt1520211", TvdbID: 153021, Genres: []string{"Action & Adventure", "Drama", "Sci-Fi & Fantasy"}, Status: "ended", Runtime: 0, Network: "AMC"},
	{ID: 71446, TmdbID: 71446, Title: "Money Heist", Year: 2017, Overview: "To carry out the biggest heist in history, a mysterious man called The Professor recruits a band of eight robbers who have a single characteristic:...", PosterURL: "https://image.tmdb.org/t/p/w500/reEMJA1uzscCbkpeRJeTT2bjqUp.jpg", BackdropURL: "https://image.tmdb.org/t/p/w780/xGexTKCJDkl12dTW4YCBDXWb1AD.jpg", ImdbID: "tt6468322", TvdbID: 327417, Genres: []string{"Crime", "Drama"}, Status: "ended", Runtime: 0, Network: "Netflix"},
	{ID: 76479, TmdbID: 76479, Title: "The Boys", Year: 2019, Overview: "A group of vigilantes known informally as “The Boys” set out to take down corrupt superheroes with no more than blue-collar grit and a willingn...", PosterURL: "https://image.tmdb.org/t/p/w500/2zmTngn1tYC1AvfnrFLhxeD82hz.jpg", BackdropURL: "https://image.tmdb.org/t/p/w780/7cqKGQMnNabzOpi7qaIgZvQ7NGV.jpg", ImdbID: "tt1190634", TvdbID: 355567, Genres: []string{"Sci-Fi & Fantasy", "Action & Adventure"}, Status: "continuing", Runtime: 0, Network: "Prime Video"},
	{ID: 93405, TmdbID: 93405, Title: "Squid Game", Year: 2021, Overview: "Hundreds of cash-strapped players accept a strange invitation to compete in children's games. Inside, a tempting prize awaits — with deadly high ...", PosterURL: "https://image.tmdb.org/t/p/w500/1QdXdRYfktUSONkl1oD5gc6Be0s.jpg", BackdropURL: "https://image.tmdb.org/t/p/w780/2meX1nMdScFOoV4370rqHWKmXhY.jpg", ImdbID: "tt10919420", TvdbID: 383275, Genres: []string{"Action & Adventure", "Mystery", "Drama"}, Status: "ended", Runtime: 0, Network: "Netflix"},
	{ID: 100088, TmdbID: 100088, Title: "The Last of Us", Year: 2023, Overview: "Twenty years after modern civilization has been destroyed, Joel, a hardened survivor, is hired to smuggle Ellie, a 14-year-old girl, out of an oppr...", PosterURL: "https://image.tmdb.org/t/p/w500/dmo6TYuuJgaYinXBPjrgG9mB5od.jpg", BackdropURL: "https://image.tmdb.org/t/p/w780/lY2DhbA7Hy44fAKddr06UrXWWaQ.jpg", ImdbID: "tt3581920", TvdbID: 392256, Genres: []string{"Drama"}, Status: "continuing", Runtime: 0, Network: "HBO"},
	{ID: 60059, TmdbID: 60059, Title: "Better Call Saul", Year: 2015, Overview: "Six years before Saul Goodman meets Walter White. We meet him when the man who will become Saul Goodman is known as Jimmy McGill, a small-time lawy...", PosterURL: "https://image.tmdb.org/t/p/w500/fC2HDm5t0kHl7mTm7jxMR31b7by.jpg", BackdropURL: "https://image.tmdb.org/t/p/w780/t15KHp3iNfHVQBNIaqUGW12xQA4.jpg", ImdbID: "tt3032476", TvdbID: 273181, Genres: []string{"Crime", "Drama"}, Status: "ended", Runtime: 0, Network: "AMC"},
	{ID: 63174, TmdbID: 63174, Title: "Lucifer", Year: 2016, Overview: "Bored and unhappy as the Lord of Hell, Lucifer Morningstar abandoned his throne and retired to Los Angeles, where he has teamed up with LAPD detect...", PosterURL: "https://image.tmdb.org/t/p/w500/ekZobS8isE6mA53RAiGDG93hBxL.jpg", BackdropURL: "https://image.tmdb.org/t/p/w780/ncftkNAjIz2PBbUMY7T0CHVJP8d.jpg", ImdbID: "tt4052886", TvdbID: 295685, Genres: []string{"Crime", "Sci-Fi & Fantasy"}, Status: "ended", Runtime: 45, Network: "FOX"},
	{ID: 82856, TmdbID: 82856, Title: "The Mandalorian", Year: 2019, Overview: "After the fall of the Galactic Empire, lawlessness has spread throughout the galaxy. A lone gunfighter makes his way through the outer reaches, ear...", PosterURL: "https://image.tmdb.org/t/p/w500/sWgBv7LV2PRoQgkxwlibdGXKz1S.jpg", BackdropURL: "https://image.tmdb.org/t/p/w780/9zcbqSxdsRMZWHYtyCd1nXPr2xq.jpg", ImdbID: "tt8111088", TvdbID: 361753, Genres: []string{"Sci-Fi & Fantasy", "Action & Adventure", "Drama"}, Status: "ended", Runtime: 0, Network: "Disney+"},
	{ID: 95557, TmdbID: 95557, Title: "INVINCIBLE", Year: 2021, Overview: "Mark Grayson is a normal teenager except for the fact that his father is the most powerful superhero on the planet. Shortly after his seventeenth b...", PosterURL: "https://image.tmdb.org/t/p/w500/jBn4LWlgdsf6xIUYhYBwpctBVsj.jpg", BackdropURL: "https://image.tmdb.org/t/p/w780/dfmPbyeZZSz3bekeESvMJaH91gS.jpg", ImdbID: "tt6741278", TvdbID: 368207, Genres: []string{"Animation", "Sci-Fi & Fantasy", "Action & Adventure", "Drama"}, Status: "continuing", Runtime: 0, Network: "Prime Video"},
	{ID: 73586, TmdbID: 73586, Title: "Yellowstone", Year: 2018, Overview: "Follow the violent world of the Dutton family, who controls the largest contiguous ranch in the United States. Led by their patriarch John Dutton, ...", PosterURL: "https://image.tmdb.org/t/p/w500/s4QRRYc1V2e68Qy9Wel9MI8fhRP.jpg", BackdropURL: "https://image.tmdb.org/t/p/w780/ynSOcgDLZfdLCZfRSYZGiTgYJVo.jpg", ImdbID: "tt4236770", TvdbID: 341164, Genres: []string{"Western", "Drama"}, Status: "ended", Runtime: 0, Network: "Paramount Network"},
	{ID: 85271, TmdbID: 85271, Title: "WandaVision", Year: 2021, Overview: "Wanda Maximoff and Vision—two super-powered beings living idealized suburban lives—begin to suspect that everything is not as it seems.", PosterURL: "https://image.tmdb.org/t/p/w500/frobUz2X5Pc8OiVZU8Oo5K3NKMM.jpg", BackdropURL: "https://image.tmdb.org/t/p/w780/lOr9NKxh4vMweufMOUDJjJhCRHW.jpg", ImdbID: "tt9140560", TvdbID: 362392, Genres: []string{"Sci-Fi & Fantasy", "Mystery", "Drama"}, Status: "ended", Runtime: 0, Network: "Disney+"},
	{ID: 114461, TmdbID: 114461, Title: "Ahsoka", Year: 2023, Overview: "Former Jedi Knight Ahsoka Tano investigates an emerging threat to a vulnerable galaxy.", PosterURL: "https://image.tmdb.org/t/p/w500/eiJeWeCAEZAmRppnXHiTWDcCd3Q.jpg", BackdropURL: "https://image.tmdb.org/t/p/w780/loDy1LWCkPjECjVTRmyKtOoUpNN.jpg", ImdbID: "tt13622776", TvdbID: 393187, Genres: []string{"Sci-Fi & Fantasy", "Action & Adventure", "Drama"}, Status: "continuing", Runtime: 0, Network: "Disney+"},
	{ID: 94605, TmdbID: 94605, Title: "Arcane", Year: 2021, Overview: "Amid the stark discord of twin cities Piltover and Zaun, two sisters fight on rival sides of a war between magic technologies and clashing convicti...", PosterURL: "https://image.tmdb.org/t/p/w500/wwbHr8MPErMbmiYNaxDgTWyewOX.jpg", BackdropURL: "https://image.tmdb.org/t/p/w780/q8eejQcg1bAqImEV8jh8RtBD4uH.jpg", ImdbID: "tt11126994", TvdbID: 371028, Genres: []string{"Animation", "Action & Adventure"}, Status: "ended", Runtime: 0, Network: "Netflix"},
}

// TMDB Series Seasons - Real episode data from TMDB API
var mockSeriesSeasons = []mockSeriesWithSeasons{
	{
		SeriesID: 1396,
		Seasons: []tmdb.NormalizedSeasonResult{
			{SeasonNumber: 1, Name: "Season 1", Overview: "High school chemistry teacher Walter White's life is suddenly transformed by ...", AirDate: "2008-01-20", Episodes: []tmdb.NormalizedEpisodeResult{
				{EpisodeNumber: 1, SeasonNumber: 1, Title: "Pilot", Overview: "When an unassuming high school chemistry teacher discovers he has a rare form of lung cancer, he ...", AirDate: "2008-01-20", Runtime: 59},
				{EpisodeNumber: 2, SeasonNumber: 1, Title: "Cat's in the Bag...", Overview: "Walt and Jesse attempt to tie up loose ends. The desperate situation gets more complicated with t...", AirDate: "2008-01-27", Runtime: 49},
				{EpisodeNumber: 3, SeasonNumber: 1, Title: "...And the Bag's in the River", Overview: "Walter fights with Jesse over his drug use, causing him to leave Walter alone with their captive,...", AirDate: "2008-02-10", Runtime: 49},
				{EpisodeNumber: 4, SeasonNumber: 1, Title: "Cancer Man", Overview: "Walter finally tells his family that he has been stricken with cancer. Meanwhile, the DEA believe...", AirDate: "2008-02-17", Runtime: 49},
				{EpisodeNumber: 5, SeasonNumber: 1, Title: "Gray Matter", Overview: "Walter and Skyler attend a former colleague's party. Jesse tries to free himself from the drugs, ...", AirDate: "2008-02-24", Runtime: 49},
				{EpisodeNumber: 6, SeasonNumber: 1, Title: "Crazy Handful of Nothin'", Overview: "The side effects of chemo begin to plague Walt. Meanwhile, the DEA rounds up suspected dealers.", AirDate: "2008-03-02", Runtime: 49},
				{EpisodeNumber: 7, SeasonNumber: 1, Title: "A No Rough Stuff Type Deal", Overview: "Walter accepts his new identity as a drug dealer after a PTA meeting. Elsewhere, Jesse decides to...", AirDate: "2008-03-09", Runtime: 48},
			}},
			{SeasonNumber: 2, Name: "Season 2", Overview: "Walt must deal with the chain reaction of his choice, as he and Jesse face ne...", AirDate: "2009-03-08", Episodes: []tmdb.NormalizedEpisodeResult{
				{EpisodeNumber: 1, SeasonNumber: 2, Title: "Seven Thirty-Seven", Overview: "Walt and Jesse are vividly reminded of Tuco's volatile nature, and try to figure a way out of t...", AirDate: "2009-03-08", Runtime: 48},
				{EpisodeNumber: 2, SeasonNumber: 2, Title: "Grilled", Overview: "Walt and Jesse find themselves in close quarters with an unhinged Tuco. Marie and Hank comfort Sk...", AirDate: "2009-03-15", Runtime: 48},
				{EpisodeNumber: 3, SeasonNumber: 2, Title: "Bit by a Dead Bee", Overview: "Walt and Jesse become short on cash when they try to cover their tracks. Meanwhile, the DEA has a...", AirDate: "2009-03-22", Runtime: 47},
				{EpisodeNumber: 4, SeasonNumber: 2, Title: "Down", Overview: "Walt attempts to reconnect with his family, while Jesse struggles to rebuild his life.", AirDate: "2009-03-29", Runtime: 48},
				{EpisodeNumber: 5, SeasonNumber: 2, Title: "Breakage", Overview: "Hank suffers from the aftermath of his encounter with Tuco. Meanwhile, Jesse hires a crew to get ...", AirDate: "2009-04-05", Runtime: 48},
				{EpisodeNumber: 6, SeasonNumber: 2, Title: "Peekaboo", Overview: "Walt's secret is in jeopardy when Skyler thanks Gretchen for paying for his treatment.", AirDate: "2009-04-12", Runtime: 48},
				{EpisodeNumber: 7, SeasonNumber: 2, Title: "Negro y Azul", Overview: "Jesse and Walt discuss expanding into new territories; Hank struggles to fit in; Skyler pursues a...", AirDate: "2009-04-19", Runtime: 48},
				{EpisodeNumber: 8, SeasonNumber: 2, Title: "Better Call Saul", Overview: "Walt and Jesse seek advice from a shady attorney when Badger gets in trouble with the law; the DE...", AirDate: "2009-04-26", Runtime: 48},
				{EpisodeNumber: 9, SeasonNumber: 2, Title: "4 Days Out", Overview: "Walt and his family wait for news after he undergoes a PET-CT scan. Walt follows Saul's advice; J...", AirDate: "2009-05-03", Runtime: 48},
				{EpisodeNumber: 10, SeasonNumber: 2, Title: "Over", Overview: "Walt and Hank get into a heated argument at a party. Skyler opens up to her boss. Jane hides her ...", AirDate: "2009-05-10", Runtime: 48},
				{EpisodeNumber: 11, SeasonNumber: 2, Title: "Mandala", Overview: "As the end of her pregnancy finds Skyler conflicted about her feelings, a dealer's death forces W...", AirDate: "2009-05-17", Runtime: 48},
				{EpisodeNumber: 12, SeasonNumber: 2, Title: "Phoenix", Overview: "As Walt explores money laundering options, he and Jesse spar over the profits from their latest d...", AirDate: "2009-05-24", Runtime: 48},
				{EpisodeNumber: 13, SeasonNumber: 2, Title: "ABQ", Overview: "Skyler confronts Walt about his secrecy; Jesse falls apart; and Jane's grief-stricken father take...", AirDate: "2009-05-31", Runtime: 48},
			}},
			{SeasonNumber: 3, Name: "Season 3", Overview: "Walt continues to battle dueling identities: a desperate husband and father t...", AirDate: "2010-03-21", Episodes: []tmdb.NormalizedEpisodeResult{
				{EpisodeNumber: 1, SeasonNumber: 3, Title: "No Más", Overview: "Walt faces a new threat, on a new front and deals with an increasingly angry Skyler, who must con...", AirDate: "2010-03-21", Runtime: 48},
				{EpisodeNumber: 2, SeasonNumber: 3, Title: "Caballo sin Nombre", Overview: "Despite ever-increasing tension between Walt and Skyler, he pulls out all the stops in an effort ...", AirDate: "2010-03-28", Runtime: 48},
				{EpisodeNumber: 3, SeasonNumber: 3, Title: "I.F.T.", Overview: "Walt ignores Skyler's demands, furthering the rift between them and pushing her to break bad. Sti...", AirDate: "2010-04-04", Runtime: 48},
				{EpisodeNumber: 4, SeasonNumber: 3, Title: "Green Light", Overview: "Walt loses control as he reacts to Skyler's news, endangering his job and relationships with Saul...", AirDate: "2010-04-11", Runtime: 48},
				{EpisodeNumber: 5, SeasonNumber: 3, Title: "Más", Overview: "Gus increases his efforts to lure Walt back into business, forcing a rift between Walt and Jesse....", AirDate: "2010-04-18", Runtime: 48},
				{EpisodeNumber: 6, SeasonNumber: 3, Title: "Sunset", Overview: "Walt settles into his new surroundings; Walt, Jr. wants answers about his parents' relationship; ...", AirDate: "2010-04-25", Runtime: 48},
				{EpisodeNumber: 7, SeasonNumber: 3, Title: "One Minute", Overview: "Hank's increasing volatility forces a confrontation with Jesse and trouble at work. Skyler pressu...", AirDate: "2010-05-02", Runtime: 48},
				{EpisodeNumber: 8, SeasonNumber: 3, Title: "I See You", Overview: "The family waits for news about Hank. While Jesse covers at the lab, Walt attempts to placate Gus...", AirDate: "2010-05-09", Runtime: 48},
				{EpisodeNumber: 9, SeasonNumber: 3, Title: "Kafkaesque", Overview: "As Hank's hospital bills stack up, Skyler hatches a plan. Walt and Gus come to a better understan...", AirDate: "2010-05-16", Runtime: 48},
				{EpisodeNumber: 10, SeasonNumber: 3, Title: "Fly", Overview: "Walt becomes obsessed with a contaminant in the lab and refuses to finish the cook until it is el...", AirDate: "2010-05-23", Runtime: 48},
				{EpisodeNumber: 11, SeasonNumber: 3, Title: "Abiquiu", Overview: "Skyler gets involved with Walt's business. Hank struggles with his recovery. Jesse makes a startl...", AirDate: "2010-05-30", Runtime: 48},
				{EpisodeNumber: 12, SeasonNumber: 3, Title: "Half Measures", Overview: "Against Walt's advice, Jesse lashes out. Fearing for Jesse's safety, Walt takes drastic action to...", AirDate: "2010-06-06", Runtime: 48},
				{EpisodeNumber: 13, SeasonNumber: 3, Title: "Full Measure", Overview: "With Jesse on the run and Mike in hot pursuit, Walt negotiates a bargain with Gus and concocts a ...", AirDate: "2010-06-13", Runtime: 48},
			}},
			{SeasonNumber: 4, Name: "Season 4", Overview: "Walt and Jesse must cope with the fallout of their previous actions, both per...", AirDate: "2011-07-17", Episodes: []tmdb.NormalizedEpisodeResult{
				{EpisodeNumber: 1, SeasonNumber: 4, Title: "Box Cutter", Overview: "Walt and Jesse face the deadly consequences of their actions. Skyler deals with a puzzling disapp...", AirDate: "2011-07-17", Runtime: 48},
				{EpisodeNumber: 2, SeasonNumber: 4, Title: "Thirty-Eight Snub", Overview: "Walt attempts to form a new alliance as he plans his next move. Skyler pushes Walt towards a busi...", AirDate: "2011-07-24", Runtime: 46},
				{EpisodeNumber: 3, SeasonNumber: 4, Title: "Open House", Overview: "Events spiral out of control at Jesse's place. Skyler reluctantly asks for Saul's help. Marie ret...", AirDate: "2011-07-31", Runtime: 47},
				{EpisodeNumber: 4, SeasonNumber: 4, Title: "Bullet Points", Overview: "The Cartel takes steps to gain the upper hand. Walt and Skyler share an embarrassing secret with ...", AirDate: "2011-08-07", Runtime: 46},
				{EpisodeNumber: 5, SeasonNumber: 4, Title: "Shotgun", Overview: "When Jesse goes missing, Walt fears the worst. Skyler has an unlikely reunion. Hank shares some b...", AirDate: "2011-08-14", Runtime: 48},
				{EpisodeNumber: 6, SeasonNumber: 4, Title: "Cornered", Overview: "Skyler makes an unsettling discovery. Walter, Jr. pushes his dad into a questionable purchase. Je...", AirDate: "2011-08-21", Runtime: 48},
				{EpisodeNumber: 7, SeasonNumber: 4, Title: "Problem Dog", Overview: "A frustrated Walt gambles on a risky new plan. Skyler's business venture hits a snag. Hank recrui...", AirDate: "2011-08-28", Runtime: 48},
				{EpisodeNumber: 8, SeasonNumber: 4, Title: "Hermanos", Overview: "Skyler develops an unusual solution to her money troubles. Hank enlists Walt to investigate a the...", AirDate: "2011-09-04", Runtime: 48},
				{EpisodeNumber: 9, SeasonNumber: 4, Title: "Bug", Overview: "Skyler's past mistakes come back to haunt her. Gus takes action to thwart his rivals. Jesse seeks...", AirDate: "2011-09-11", Runtime: 48},
				{EpisodeNumber: 10, SeasonNumber: 4, Title: "Salud", Overview: "Walt's family worries when he doesn't turn up for Walter, Jr.'s 16th birthday. Jesse is forced to...", AirDate: "2011-09-18", Runtime: 48},
				{EpisodeNumber: 11, SeasonNumber: 4, Title: "Crawl Space", Overview: "Walt takes drastic action to protect his secret and Gus. Skyler's efforts to solve Ted's financia...", AirDate: "2011-09-25", Runtime: 47},
				{EpisodeNumber: 12, SeasonNumber: 4, Title: "End Times", Overview: "Hank pushes Gomez to pursue one last lead, while Walt struggles to protect the family. Jesse gets...", AirDate: "2011-10-02", Runtime: 46},
				{EpisodeNumber: 13, SeasonNumber: 4, Title: "Face Off", Overview: "Walt and Jesse team up to take on Gus. With Saul's help, Walt finds an unexpected ally.", AirDate: "2011-10-09", Runtime: 51},
			}},
			{SeasonNumber: 5, Name: "Season 5", Overview: "Walt is faced with the prospect of moving on in a world without his enemy. As...", AirDate: "2012-07-15", Episodes: []tmdb.NormalizedEpisodeResult{
				{EpisodeNumber: 1, SeasonNumber: 5, Title: "Live Free or Die", Overview: "As Walt deals with the aftermath of the Casa Tranquila explosion, Hank works to wrap up his inves...", AirDate: "2012-07-15", Runtime: 43},
				{EpisodeNumber: 2, SeasonNumber: 5, Title: "Madrigal", Overview: "Walt and Jesse pursue an unlikely business partner. The DEA filters through various leads in hope...", AirDate: "2012-07-22", Runtime: 48},
				{EpisodeNumber: 3, SeasonNumber: 5, Title: "Hazard Pay", Overview: "Walt and Jesse formulate a new business plan. Walt shares a secret with Marie.", AirDate: "2012-07-29", Runtime: 48},
				{EpisodeNumber: 4, SeasonNumber: 5, Title: "Fifty-One", Overview: "Walt celebrates another birthday. Skyler contemplates her options, and an associate puts a crimp ...", AirDate: "2012-08-05", Runtime: 48},
				{EpisodeNumber: 5, SeasonNumber: 5, Title: "Dead Freight", Overview: "Walt's team gets creative to obtain the methylamine they need to continue their operation.", AirDate: "2012-08-12", Runtime: 49},
				{EpisodeNumber: 6, SeasonNumber: 5, Title: "Buyout", Overview: "Walt, Jesse, and Mike struggle over the future of their business, as occupational hazards weigh o...", AirDate: "2012-08-19", Runtime: 48},
				{EpisodeNumber: 7, SeasonNumber: 5, Title: "Say My Name", Overview: "Walt takes control of business matters; Mike deals with the consequences of his actions.", AirDate: "2012-08-26", Runtime: 48},
				{EpisodeNumber: 8, SeasonNumber: 5, Title: "Gliding Over All", Overview: "Walt takes care of loose ends; Walt makes a dangerous decision.", AirDate: "2012-09-02", Runtime: 48},
				{EpisodeNumber: 9, SeasonNumber: 5, Title: "Blood Money", Overview: "As Walt and Jesse adjust to life out of the business, Hank grapples with a troubling lead.", AirDate: "2013-08-11", Runtime: 48},
				{EpisodeNumber: 10, SeasonNumber: 5, Title: "Buried", Overview: "While Skyler's past catches up with her, Walt covers his tracks. Jesse continues to struggle with...", AirDate: "2013-08-18", Runtime: 48},
				{EpisodeNumber: 11, SeasonNumber: 5, Title: "Confessions", Overview: "Jesse decides to make a change, while Walt and Skyler try to deal with an unexpected demand.", AirDate: "2013-08-25", Runtime: 48},
				{EpisodeNumber: 12, SeasonNumber: 5, Title: "Rabid Dog", Overview: "An unusual strategy starts to bear fruit, while plans are set in motion that could change everyth...", AirDate: "2013-09-01", Runtime: 48},
				{EpisodeNumber: 13, SeasonNumber: 5, Title: "To'hajiilee", Overview: "Things heat up for Walt in unexpected ways.", AirDate: "2013-09-08", Runtime: 47},
				{EpisodeNumber: 14, SeasonNumber: 5, Title: "Ozymandias", Overview: "Everyone copes with radically changed circumstances.", AirDate: "2013-09-15", Runtime: 48},
				{EpisodeNumber: 15, SeasonNumber: 5, Title: "Granite State", Overview: "Events set in motion long ago move toward a conclusion.", AirDate: "2013-09-22", Runtime: 54},
				{EpisodeNumber: 16, SeasonNumber: 5, Title: "Felina", Overview: "All bad things must come to an end.", AirDate: "2013-09-29", Runtime: 56},
			}},
		},
	},
	{
		SeriesID: 1399,
		Seasons: []tmdb.NormalizedSeasonResult{
			{SeasonNumber: 1, Name: "Season 1", Overview: "Trouble is brewing in the Seven Kingdoms of Westeros. For the driven inhabita...", AirDate: "2011-04-17", Episodes: []tmdb.NormalizedEpisodeResult{
				{EpisodeNumber: 1, SeasonNumber: 1, Title: "Winter Is Coming", Overview: "Jon Arryn, the Hand of the King, is dead. King Robert Baratheon plans to ask his oldest friend, E...", AirDate: "2011-04-17", Runtime: 62},
				{EpisodeNumber: 2, SeasonNumber: 1, Title: "The Kingsroad", Overview: "While Bran recovers from his fall, Ned takes only his daughters to Kings Landing. Jon Snow goes w...", AirDate: "2011-04-24", Runtime: 56},
				{EpisodeNumber: 3, SeasonNumber: 1, Title: "Lord Snow", Overview: "Lord Stark and his daughters arrive at King's Landing to discover the intrigues of the king's realm.", AirDate: "2011-05-01", Runtime: 58},
				{EpisodeNumber: 4, SeasonNumber: 1, Title: "Cripples, Bastards, and Broken Things", Overview: "Eddard investigates Jon Arryn's murder. Jon befriends Samwell Tarly, a coward who has come to joi...", AirDate: "2011-05-08", Runtime: 56},
				{EpisodeNumber: 5, SeasonNumber: 1, Title: "The Wolf and the Lion", Overview: "Catelyn has captured Tyrion and plans to bring him to her sister, Lysa Arryn, at The Vale, to be ...", AirDate: "2011-05-15", Runtime: 55},
				{EpisodeNumber: 6, SeasonNumber: 1, Title: "A Golden Crown", Overview: "While recovering from his battle with Jamie, Eddard is forced to run the kingdom while Robert goe...", AirDate: "2011-05-22", Runtime: 53},
				{EpisodeNumber: 7, SeasonNumber: 1, Title: "You Win or You Die", Overview: "Robert has been injured while hunting and is dying. Jon and the others finally take their vows to...", AirDate: "2011-05-29", Runtime: 58},
				{EpisodeNumber: 8, SeasonNumber: 1, Title: "The Pointy End", Overview: "Eddard and his men are betrayed and captured by the Lannisters. When word reaches Robb, he plans ...", AirDate: "2011-06-05", Runtime: 59},
				{EpisodeNumber: 9, SeasonNumber: 1, Title: "Baelor", Overview: "Robb goes to war against the Lannisters. Jon finds himself struggling on deciding if his place is...", AirDate: "2011-06-12", Runtime: 57},
				{EpisodeNumber: 10, SeasonNumber: 1, Title: "Fire and Blood", Overview: "With Ned dead, Robb vows to get revenge on the Lannisters. Jon must officially decide if his plac...", AirDate: "2011-06-19", Runtime: 53},
			}},
			{SeasonNumber: 2, Name: "Season 2", Overview: "The cold winds of winter are rising in Westeros...war is coming...and five ki...", AirDate: "2012-04-01", Episodes: []tmdb.NormalizedEpisodeResult{
				{EpisodeNumber: 1, SeasonNumber: 2, Title: "The North Remembers", Overview: "Tyrion arrives to save Joffrey's crown from threats old and new; Daenerys searches for allies and...", AirDate: "2012-04-01", Runtime: 53},
				{EpisodeNumber: 2, SeasonNumber: 2, Title: "The Night Lands", Overview: "Arya shares a secret with a familiar recruit; a scout returns to Dany with disturbing news; Theon...", AirDate: "2012-04-08", Runtime: 54},
				{EpisodeNumber: 3, SeasonNumber: 2, Title: "What is Dead May Never Die", Overview: "Tyrion roots out a spy; Catelyn meets a new king and queen; Bran dreams; Theon drowns.", AirDate: "2012-04-15", Runtime: 53},
				{EpisodeNumber: 4, SeasonNumber: 2, Title: "Garden of Bones", Overview: "Catelyn tries to save two kings from themselves; Tyrion practices coercion; Robb meets a foreigne...", AirDate: "2012-04-22", Runtime: 51},
				{EpisodeNumber: 5, SeasonNumber: 2, Title: "The Ghost of Harrenhal", Overview: "The Baratheon rivalry ends; Tyrion learns of Cersei's secret weapon; Dany suffers a loss; Arya co...", AirDate: "2012-04-29", Runtime: 55},
				{EpisodeNumber: 6, SeasonNumber: 2, Title: "The Old Gods and the New", Overview: "Arya has a surprise visitor; Dany vows to take what is hers; Joffrey meets his subjects; Qhorin g...", AirDate: "2012-05-06", Runtime: 54},
				{EpisodeNumber: 7, SeasonNumber: 2, Title: "A Man Without Honor", Overview: "Jaime meets a distant relative; Dany receives an invitation; Theon leads a search party; Jon lose...", AirDate: "2012-05-13", Runtime: 56},
				{EpisodeNumber: 8, SeasonNumber: 2, Title: "The Prince of Winterfell", Overview: "Theon holds down the fort; Arya calls in her debt with Jaqen; Robb is betrayed; Stannis and Davos...", AirDate: "2012-05-20", Runtime: 54},
				{EpisodeNumber: 9, SeasonNumber: 2, Title: "Blackwater", Overview: "Tyrion and the Lannisters fight for their lives as Stannis' fleet assaults King's Landing.", AirDate: "2012-05-27", Runtime: 55},
				{EpisodeNumber: 10, SeasonNumber: 2, Title: "Valar Morghulis", Overview: "Tyrion wakes up in a new world; Dany goes to a strange place; Jon proves himself.", AirDate: "2012-06-03", Runtime: 64},
			}},
			{SeasonNumber: 3, Name: "Season 3", Overview: "Duplicity and treachery...nobility and honor...conquest and triumph...and, of...", AirDate: "2013-03-31", Episodes: []tmdb.NormalizedEpisodeResult{
				{EpisodeNumber: 1, SeasonNumber: 3, Title: "Valar Dohaeris", Overview: "Jon is tested by the wildling king; Tyrion asks for his reward; Dany sails to Slaver's Bay.", AirDate: "2013-03-31", Runtime: 55},
				{EpisodeNumber: 2, SeasonNumber: 3, Title: "Dark Wings, Dark Words", Overview: "Sansa says too much. Shae asks Tyrion for a favor. Jaime finds a way to pass the time, while Arya...", AirDate: "2013-04-07", Runtime: 58},
				{EpisodeNumber: 3, SeasonNumber: 3, Title: "Walk of Punishment", Overview: "Tyrion shoulders new responsibilities. Jon is taken to the Fist of the First Men. Daenerys meets ...", AirDate: "2013-04-14", Runtime: 53},
				{EpisodeNumber: 4, SeasonNumber: 3, Title: "And Now His Watch Is Ended", Overview: "The Night's Watch takes stock. Varys meets his better. Arya is taken to the commander of the Brot...", AirDate: "2013-04-21", Runtime: 54},
				{EpisodeNumber: 5, SeasonNumber: 3, Title: "Kissed by Fire", Overview: "The Hound is judged by the gods. Jaime is judged by men. Jon proves himself. Robb is betrayed. Ty...", AirDate: "2013-04-28", Runtime: 58},
				{EpisodeNumber: 6, SeasonNumber: 3, Title: "The Climb", Overview: "Four Houses consider make-or-break alliances. Jon and the Wildlings face a daunting climb.", AirDate: "2013-05-05", Runtime: 54},
				{EpisodeNumber: 7, SeasonNumber: 3, Title: "The Bear and the Maiden Fair", Overview: "Dany exchanges gifts in Yunkai; Brienne faces a formidable foe in Harrenhal.", AirDate: "2013-05-12", Runtime: 58},
				{EpisodeNumber: 8, SeasonNumber: 3, Title: "Second Sons", Overview: "Dany meets the Titan's Bastard; King's Landing hosts a royal wedding.", AirDate: "2013-05-19", Runtime: 57},
				{EpisodeNumber: 9, SeasonNumber: 3, Title: "The Rains of Castamere", Overview: "House Frey joins forces with House Tully. Jon faces his most difficult test yet.", AirDate: "2013-06-02", Runtime: 51},
				{EpisodeNumber: 10, SeasonNumber: 3, Title: "Mhysa", Overview: "Joffrey challenges Tywin. Dany waits to see if she is a conqueror or a liberator.", AirDate: "2013-06-09", Runtime: 63},
			}},
			{SeasonNumber: 4, Name: "Season 4", Overview: "The War of the Five Kings is drawing to a close, but new intrigues and plots ...", AirDate: "2014-04-06", Episodes: []tmdb.NormalizedEpisodeResult{
				{EpisodeNumber: 1, SeasonNumber: 4, Title: "Two Swords", Overview: "King's Landing prepares for a royal wedding; Dany finds the way to Meereen; the Night's Watch bra...", AirDate: "2014-04-06", Runtime: 59},
				{EpisodeNumber: 2, SeasonNumber: 4, Title: "The Lion and the Rose", Overview: "The Lannisters and their guests gather in King's Landing; Stannis loses patience with Davos; Rams...", AirDate: "2014-04-13", Runtime: 53},
				{EpisodeNumber: 3, SeasonNumber: 4, Title: "Breaker of Chains", Overview: "Tyrion ponders his options; Tywin extends an olive branch; Jon proposes a bold plan; The Hound te...", AirDate: "2014-04-20", Runtime: 57},
				{EpisodeNumber: 4, SeasonNumber: 4, Title: "Oathkeeper", Overview: "Dany balances justice and mercy; Jaime tasks Brienne with his honor; Jon readies his men.", AirDate: "2014-04-27", Runtime: 56},
				{EpisodeNumber: 5, SeasonNumber: 4, Title: "First of His Name", Overview: "Cersei and Tywin plot the Crown's next move. Dany discusses future plans. Jon Snow embarks on a n...", AirDate: "2014-05-04", Runtime: 54},
				{EpisodeNumber: 6, SeasonNumber: 4, Title: "The Laws of Gods and Men", Overview: "Stannis and Davos set sail with a new strategy. Dany meets with supplicants. Tyrion faces down hi...", AirDate: "2014-05-11", Runtime: 51},
				{EpisodeNumber: 7, SeasonNumber: 4, Title: "Mockingbird", Overview: "Tyrion enlists an unlikely ally. Daario entreats Dany to allow him to do what he does best. Jon's...", AirDate: "2014-05-18", Runtime: 52},
				{EpisodeNumber: 8, SeasonNumber: 4, Title: "The Mountain and the Viper", Overview: "Mole's Town receives some unexpected visitors. Littlefinger's motives are questioned. Tyrion's fa...", AirDate: "2014-06-01", Runtime: 53},
				{EpisodeNumber: 9, SeasonNumber: 4, Title: "The Watchers on the Wall", Overview: "Jon Snow and the rest of the Night's Watch face the biggest challenge to the Wall yet.", AirDate: "2014-06-08", Runtime: 51},
				{EpisodeNumber: 10, SeasonNumber: 4, Title: "The Children", Overview: "Circumstances change after an unexpected arrival from north of the Wall. Dany must face harsh rea...", AirDate: "2014-06-15", Runtime: 66},
			}},
			{SeasonNumber: 5, Name: "Season 5", Overview: "The War of the Five Kings, once thought to be drawing to a close, is instead ...", AirDate: "2015-04-12", Episodes: []tmdb.NormalizedEpisodeResult{
				{EpisodeNumber: 1, SeasonNumber: 5, Title: "The Wars to Come", Overview: "Cersei and Jaime adjust to a world without Tywin. Varys reveals a conspiracy to Tyrion. Dany face...", AirDate: "2015-04-12", Runtime: 53},
				{EpisodeNumber: 2, SeasonNumber: 5, Title: "The House of Black and White", Overview: "Arya arrives in Braavos. Podrick and Brienne run into trouble on the road. Cersei fears for her d...", AirDate: "2015-04-19", Runtime: 56},
				{EpisodeNumber: 3, SeasonNumber: 5, Title: "High Sparrow", Overview: "In Braavos, Arya sees the Many-Faced God. In King's Landing, Queen Margaery enjoys her new husband...", AirDate: "2015-04-26", Runtime: 60},
				{EpisodeNumber: 4, SeasonNumber: 5, Title: "Sons of the Harpy", Overview: "The Faith Militant grow increasingly aggressive. Jaime and Bronn head south. Ellaria and the Sand...", AirDate: "2015-05-03", Runtime: 51},
				{EpisodeNumber: 5, SeasonNumber: 5, Title: "Kill the Boy", Overview: "Dany makes a difficult decision in Meereen. Jon recruits the help of an unexpected ally. Brienne ...", AirDate: "2015-05-10", Runtime: 57},
				{EpisodeNumber: 6, SeasonNumber: 5, Title: "Unbowed, Unbent, Unbroken", Overview: "Arya trains. Jorah and Tyrion run into slavers. Trystane and Myrcella make plans. Jaime and Bronn...", AirDate: "2015-05-17", Runtime: 54},
				{EpisodeNumber: 7, SeasonNumber: 5, Title: "The Gift", Overview: "Jon prepares for conflict. Sansa tries to talk to Theon. Brienne waits for a sign. Stannis remain...", AirDate: "2015-05-24", Runtime: 59},
				{EpisodeNumber: 8, SeasonNumber: 5, Title: "Hardhome", Overview: "Arya makes progress in her training. Sansa confronts an old friend. Cersei struggles. Jon travels.", AirDate: "2015-05-31", Runtime: 60},
				{EpisodeNumber: 9, SeasonNumber: 5, Title: "The Dance of Dragons", Overview: "Stannis confronts a troubling decision. Jon returns to The Wall. Mace visits the Iron Bank. Arya ...", AirDate: "2015-06-07", Runtime: 53},
				{EpisodeNumber: 10, SeasonNumber: 5, Title: "Mother's Mercy", Overview: "Stannis marches. Dany is surrounded by strangers. Cersei seeks forgiveness. Jon is challenged.", AirDate: "2015-06-14", Runtime: 61},
			}},
			{SeasonNumber: 6, Name: "Season 6", Overview: "Following the shocking developments at the conclusion of season five, survivo...", AirDate: "2016-04-24", Episodes: []tmdb.NormalizedEpisodeResult{
				{EpisodeNumber: 1, SeasonNumber: 6, Title: "The Red Woman", Overview: "The fate of Jon Snow is revealed. Daenerys meets a strong man. Cersei sees her daughter once again.", AirDate: "2016-04-24", Runtime: 51},
				{EpisodeNumber: 2, SeasonNumber: 6, Title: "Home", Overview: "Bran trains with the Three-Eyed Raven. In King's Landing, Jaime advises Tommen. Tyrion demands ...", AirDate: "2016-05-01", Runtime: 54},
				{EpisodeNumber: 3, SeasonNumber: 6, Title: "Oathbreaker", Overview: "Daenerys meets her future. Arya trains to be No One.", AirDate: "2016-05-08", Runtime: 53},
				{EpisodeNumber: 4, SeasonNumber: 6, Title: "Book of the Stranger", Overview: "Tyrion strikes a deal. Jorah and Daario undertake a difficult task. Jaime and Cersei try to impro...", AirDate: "2016-05-15", Runtime: 59},
				{EpisodeNumber: 5, SeasonNumber: 6, Title: "The Door", Overview: "Tyrion seeks a strange ally. Bran learns a great deal. Brienne goes on a mission. Arya is given a...", AirDate: "2016-05-22", Runtime: 57},
				{EpisodeNumber: 6, SeasonNumber: 6, Title: "Blood of My Blood", Overview: "An old foe comes back into the picture. Gilly meets Sam's family. Arya faces a difficult choice...", AirDate: "2016-05-29", Runtime: 52},
				{EpisodeNumber: 7, SeasonNumber: 6, Title: "The Broken Man", Overview: "The High Sparrow eyes another target. Jaime confronts a hero. Arya makes a plan. The North is rem...", AirDate: "2016-06-05", Runtime: 51},
				{EpisodeNumber: 8, SeasonNumber: 6, Title: "No One", Overview: "While Jaime weighs his options, Cersei answers a request. Tyrion's plans bear fruit. Arya faces a...", AirDate: "2016-06-12", Runtime: 59},
				{EpisodeNumber: 9, SeasonNumber: 6, Title: "Battle of the Bastards", Overview: "Slaver envoys demand terms of surrender in Meereen. Ramsay plays the odds in defense of Winterfell.", AirDate: "2016-06-19", Runtime: 60},
				{EpisodeNumber: 10, SeasonNumber: 6, Title: "The Winds of Winter", Overview: "Cersei faces a day of reckoning. Daenerys antes up for the 'Great Game.'", AirDate: "2016-06-26", Runtime: 68},
			}},
			{SeasonNumber: 7, Name: "Season 7", Overview: "The long winter is here. And with it comes a convergence of armies and attitu...", AirDate: "2017-07-16", Episodes: []tmdb.NormalizedEpisodeResult{
				{EpisodeNumber: 1, SeasonNumber: 7, Title: "Dragonstone", Overview: "Jon Snow organizes the defense of the North. Cersei tries to even the odds. Daenerys comes home.", AirDate: "2017-07-16", Runtime: 59},
				{EpisodeNumber: 2, SeasonNumber: 7, Title: "Stormborn", Overview: "Daenerys receives an unexpected visitor. Jon faces a revolt. Sam risks his career and life. Tyrio...", AirDate: "2017-07-23", Runtime: 59},
				{EpisodeNumber: 3, SeasonNumber: 7, Title: "The Queen's Justice", Overview: "Daenerys holds court. Tyrion backchannels. Cersei returns a gift. Jaime learns from his mistakes.", AirDate: "2017-07-30", Runtime: 63},
				{EpisodeNumber: 4, SeasonNumber: 7, Title: "The Spoils of War", Overview: "The Lannisters pay their debts. Daenerys weighs her options. Arya comes home.", AirDate: "2017-08-06", Runtime: 50},
				{EpisodeNumber: 5, SeasonNumber: 7, Title: "Eastwatch", Overview: "Daenerys offers a choice. Arya grows suspicious. Tyrion answers a good question.", AirDate: "2017-08-13", Runtime: 59},
				{EpisodeNumber: 6, SeasonNumber: 7, Title: "Beyond the Wall", Overview: "Jon and the Brotherhood hunt the dead. Arya confronts Sansa. Tyrion thinks about the future.", AirDate: "2017-08-20", Runtime: 70},
				{EpisodeNumber: 7, SeasonNumber: 7, Title: "The Dragon and the Wolf", Overview: "Tyrion tries to save Westeros from itself. Sansa questions loyalties.", AirDate: "2017-08-27", Runtime: 80},
			}},
			{SeasonNumber: 8, Name: "Season 8", Overview: "The Great War has come, the Wall has fallen and the Night King's army of the ...", AirDate: "2019-04-14", Episodes: []tmdb.NormalizedEpisodeResult{
				{EpisodeNumber: 1, SeasonNumber: 8, Title: "Winterfell", Overview: "Arriving at Winterfell, Jon and Daenerys struggle to unite a divided North. Jon Snow gets some bi...", AirDate: "2019-04-14", Runtime: 55},
				{EpisodeNumber: 2, SeasonNumber: 8, Title: "A Knight of the Seven Kingdoms", Overview: "The battle at Winterfell is approaching. Jaime is confronted with the consequences of the past. A...", AirDate: "2019-04-21", Runtime: 59},
				{EpisodeNumber: 3, SeasonNumber: 8, Title: "The Long Night", Overview: "The Night King and his army have arrived at Winterfell and the great battle begins. Arya looks to...", AirDate: "2019-04-28", Runtime: 82},
				{EpisodeNumber: 4, SeasonNumber: 8, Title: "The Last of the Starks", Overview: "In the wake of a costly victory, Jon and Daenerys look to the south as Tyrion eyes a compromise t...", AirDate: "2019-05-05", Runtime: 79},
				{EpisodeNumber: 5, SeasonNumber: 8, Title: "The Bells", Overview: "Daenerys brings her forces to King's Landing.", AirDate: "2019-05-12", Runtime: 80},
				{EpisodeNumber: 6, SeasonNumber: 8, Title: "The Iron Throne", Overview: "In the aftermath of the devastating attack on King's Landing, Daenerys must face the survivors.", AirDate: "2019-05-19", Runtime: 80},
			}},
		},
	},
	{
		SeriesID: 66732,
		Seasons: []tmdb.NormalizedSeasonResult{
			{SeasonNumber: 1, Name: "Season 1", Overview: "Strange things are afoot in Hawkins, Indiana, where a young boy's sudden disa...", AirDate: "2016-07-15", Episodes: []tmdb.NormalizedEpisodeResult{
				{EpisodeNumber: 1, SeasonNumber: 1, Title: "Chapter One: The Vanishing of Will Byers", Overview: "On his way home from a friend's house, young Will sees something terrifying. Nearby, a sinister s...", AirDate: "2016-07-15", Runtime: 48},
				{EpisodeNumber: 2, SeasonNumber: 1, Title: "Chapter Two: The Weirdo on Maple Street", Overview: "Lucas, Mike and Dustin try to talk to the girl they found in the woods. Hopper questions an anxio...", AirDate: "2016-07-15", Runtime: 55},
				{EpisodeNumber: 3, SeasonNumber: 1, Title: "Chapter Three: Holly, Jolly", Overview: "An increasingly concerned Nancy looks for Barb and finds out what Jonathan's been up to. Joyce is...", AirDate: "2016-07-15", Runtime: 51},
				{EpisodeNumber: 4, SeasonNumber: 1, Title: "Chapter Four: The Body", Overview: "Refusing to believe Will is dead, Joyce tries to connect with her son. The boys give Eleven a mak...", AirDate: "2016-07-15", Runtime: 50},
				{EpisodeNumber: 5, SeasonNumber: 1, Title: "Chapter Five: The Flea and the Acrobat", Overview: "Hopper breaks into the lab while Nancy and Jonathan confront the force that took Will. The boys a...", AirDate: "2016-07-15", Runtime: 52},
				{EpisodeNumber: 6, SeasonNumber: 1, Title: "Chapter Six: The Monster", Overview: "A frantic Jonathan looks for Nancy in the darkness, but Steve's looking for her, too. Hopper and ...", AirDate: "2016-07-15", Runtime: 46},
				{EpisodeNumber: 7, SeasonNumber: 1, Title: "Chapter Seven: The Bathtub", Overview: "Eleven struggles to reach Will, while Lucas warns that \"the bad men are coming.\" Nancy and Jonath...", AirDate: "2016-07-15", Runtime: 42},
				{EpisodeNumber: 8, SeasonNumber: 1, Title: "Chapter Eight: The Upside Down", Overview: "Dr. Brenner holds Hopper and Joyce for questioning while the boys wait with Eleven in the gym. Ba...", AirDate: "2016-07-15", Runtime: 54},
			}},
			{SeasonNumber: 2, Name: "Stranger Things 2", Overview: "It's been nearly a year since Will's strange disappearance. But life's hardly...", AirDate: "2017-10-27", Episodes: []tmdb.NormalizedEpisodeResult{
				{EpisodeNumber: 1, SeasonNumber: 2, Title: "Chapter One: MADMAX", Overview: "As the town preps for Halloween, a high-scoring rival shakes things up at the arcade, and a skept...", AirDate: "2017-10-27", Runtime: 48},
				{EpisodeNumber: 2, SeasonNumber: 2, Title: "Chapter Two: Trick or Treat, Freak", Overview: "After Will sees something terrible on trick-or-treat night, Mike wonders whether Eleven's still o...", AirDate: "2017-10-27", Runtime: 56},
				{EpisodeNumber: 3, SeasonNumber: 2, Title: "Chapter Three: The Pollywog", Overview: "Dustin adopts a strange new pet, and Eleven grows increasingly impatient. A well-meaning Bob urge...", AirDate: "2017-10-27", Runtime: 51},
				{EpisodeNumber: 4, SeasonNumber: 2, Title: "Chapter Four: Will the Wise", Overview: "An ailing Will opens up to Joyce -- with disturbing results. While Hopper digs for the truth, Ele...", AirDate: "2017-10-27", Runtime: 46},
				{EpisodeNumber: 5, SeasonNumber: 2, Title: "Chapter Five: Dig Dug", Overview: "Nancy and Jonathan swap conspiracy theories with a new ally as Eleven searches for someone from h...", AirDate: "2017-10-27", Runtime: 58},
				{EpisodeNumber: 6, SeasonNumber: 2, Title: "Chapter Six: The Spy", Overview: "Will's connection to a shadowy evil grows stronger, but no one's quite sure how to stop it. Elsew...", AirDate: "2017-10-27", Runtime: 52},
				{EpisodeNumber: 7, SeasonNumber: 2, Title: "Chapter Seven: The Lost Sister", Overview: "Psychic visions draw Eleven to a band of violent outcasts and an angry girl with a shadowy past.", AirDate: "2017-10-27", Runtime: 46},
				{EpisodeNumber: 8, SeasonNumber: 2, Title: "Chapter Eight: The Mind Flayer", Overview: "An unlikely hero steps forward when a deadly development puts the Hawkins Lab on lockdown, trappi...", AirDate: "2017-10-27", Runtime: 48},
				{EpisodeNumber: 9, SeasonNumber: 2, Title: "Chapter Nine: The Gate", Overview: "Eleven makes plans to finish what she started while the survivors turn up the heat on the monstro...", AirDate: "2017-10-27", Runtime: 62},
			}},
			{SeasonNumber: 3, Name: "Stranger Things 3", Overview: "Budding romance. A brand-new mall. And rabid rats running toward danger. It's...", AirDate: "2019-07-04", Episodes: []tmdb.NormalizedEpisodeResult{
				{EpisodeNumber: 1, SeasonNumber: 3, Title: "Chapter One: Suzie, Do You Copy?", Overview: "Summer brings new jobs and budding romance. But the mood shifts when Dustin's radio picks up a Ru...", AirDate: "2019-07-04", Runtime: 51},
				{EpisodeNumber: 2, SeasonNumber: 3, Title: "Chapter Two: The Mall Rats", Overview: "Nancy and Jonathan follow a lead, Steve and Robin sign on to a secret mission, and Max and Eleven...", AirDate: "2019-07-04", Runtime: 51},
				{EpisodeNumber: 3, SeasonNumber: 3, Title: "Chapter Three: The Case of the Missing Lifeguard", Overview: "With El and Max looking for Billy, Will declares a day without girls. Steve and Dustin go on a st...", AirDate: "2019-07-04", Runtime: 50},
				{EpisodeNumber: 4, SeasonNumber: 3, Title: "Chapter Four: The Sauna Test", Overview: "A code red brings the gang back together to face a frighteningly familiar evil. Karen urges Nancy...", AirDate: "2019-07-04", Runtime: 53},
				{EpisodeNumber: 5, SeasonNumber: 3, Title: "Chapter Five: The Flayed", Overview: "Strange surprises lurk inside an old farmhouse and deep beneath the Starcourt Mall. Meanwhile, th...", AirDate: "2019-07-04", Runtime: 52},
				{EpisodeNumber: 6, SeasonNumber: 3, Title: "Chapter Six: E Pluribus Unum", Overview: "Dr. Alexei reveals what the Russians have been building, and Eleven sees where Billy has been. Du...", AirDate: "2019-07-04", Runtime: 60},
				{EpisodeNumber: 7, SeasonNumber: 3, Title: "Chapter Seven: The Bite", Overview: "With time running out -- and an assassin close behind -- Hopper's crew races back to Hawkins, whe...", AirDate: "2019-07-04", Runtime: 56},
				{EpisodeNumber: 8, SeasonNumber: 3, Title: "Chapter Eight: The Battle of Starcourt", Overview: "Terror reigns in the food court when the Mind Flayer comes to collect. But down below, in the dar...", AirDate: "2019-07-04", Runtime: 78},
			}},
			{SeasonNumber: 4, Name: "Stranger Things 4", Overview: "Darkness returns to Hawkins just in time for spring break, igniting fresh ter...", AirDate: "2022-05-27", Episodes: []tmdb.NormalizedEpisodeResult{
				{EpisodeNumber: 1, SeasonNumber: 4, Title: "Chapter One: The Hellfire Club", Overview: "El is bullied at school. Joyce opens a mysterious package. A scrappy player shakes up D&D night.", AirDate: "2022-05-27", Runtime: 79},
				{EpisodeNumber: 2, SeasonNumber: 4, Title: "Chapter Two: Vecna's Curse", Overview: "A plane brings Mike to California — and a dead body brings Hawkins to a halt. Nancy goes lookin...", AirDate: "2022-05-27", Runtime: 78},
				{EpisodeNumber: 3, SeasonNumber: 4, Title: "Chapter Three: The Monster and the Superhero", Overview: "Murray and Joyce fly to Alaska, and El faces serious consequences. Robin and Nancy dig up dirt on...", AirDate: "2022-05-27", Runtime: 64},
				{EpisodeNumber: 4, SeasonNumber: 4, Title: "Chapter Four: Dear Billy", Overview: "Max is in grave danger... and running out of time. A patient at Pennhurst asylum has visitors. El...", AirDate: "2022-05-27", Runtime: 79},
				{EpisodeNumber: 5, SeasonNumber: 4, Title: "Chapter Five: The Nina Project", Overview: "Owens takes El to Nevada, where she's forced to confront her past, while the Hawkins kids comb a ...", AirDate: "2022-05-27", Runtime: 75},
				{EpisodeNumber: 6, SeasonNumber: 4, Title: "Chapter Six: The Dive", Overview: "Behind the Iron Curtain, a risky rescue mission gets underway. The California crew seeks help fro...", AirDate: "2022-05-27", Runtime: 74},
				{EpisodeNumber: 7, SeasonNumber: 4, Title: "Chapter Seven: The Massacre at Hawkins Lab", Overview: "As Hopper braces to battle a monster, Dustin dissects Vecna's motives — and decodes a message f...", AirDate: "2022-05-27", Runtime: 100},
				{EpisodeNumber: 8, SeasonNumber: 4, Title: "Chapter Eight: Papa", Overview: "Nancy has sobering visions, and El passes an important test. Back in Hawkins, the gang gathers su...", AirDate: "2022-07-01", Runtime: 86},
				{EpisodeNumber: 9, SeasonNumber: 4, Title: "Chapter Nine: The Piggyback", Overview: "With selfless hearts and a clash of metal, heroes fight from every corner of the battlefield to s...", AirDate: "2022-07-01", Runtime: 143},
			}},
			{SeasonNumber: 5, Name: "Stranger Things 5", Overview: "The fall of 1987. Hawkins is scarred by rifts. Vecna has vanished and the gov...", AirDate: "2025-11-26", Episodes: []tmdb.NormalizedEpisodeResult{
				{EpisodeNumber: 1, SeasonNumber: 5, Title: "Chapter One: The Crawl", Overview: "November, 1987. The gang evades the military to scour the Upside Down for Vecna — but fails to ...", AirDate: "2025-11-26", Runtime: 72},
				{EpisodeNumber: 2, SeasonNumber: 5, Title: "Chapter Two: The Vanishing of...", Overview: "After a vicious attack at the Wheeler home, Mike and Nancy confront the cost of secrecy, while El...", AirDate: "2025-11-26", Runtime: 58},
				{EpisodeNumber: 3, SeasonNumber: 5, Title: "Chapter Three: The Turnbow Trap", Overview: "Will gains unique insight into Vecna's next move, giving the crew an opportunity to set a trap. H...", AirDate: "2025-11-26", Runtime: 70},
				{EpisodeNumber: 4, SeasonNumber: 5, Title: "Chapter Four: Sorcerer", Overview: "The military tightens its grip on the town. Mike, Lucas and Robin orchestrate a daring escape. El...", AirDate: "2025-11-26", Runtime: 87},
				{EpisodeNumber: 5, SeasonNumber: 5, Title: "Chapter Five: Shock Jock", Overview: "The gang hatches an electrifying plan to reconnect Will to the hive mind. Tensions flare during a...", AirDate: "2025-12-25", Runtime: 69},
				{EpisodeNumber: 6, SeasonNumber: 5, Title: "Chapter Six: Escape from Camazotz", Overview: "As Holly and Max fight to escape Vecna's mind, El must find a way into Will's. Joyce wrestles wit...", AirDate: "2025-12-25", Runtime: 76},
				{EpisodeNumber: 7, SeasonNumber: 5, Title: "Chapter Seven: The Bridge", Overview: "On the anniversary of Will's disappearance, the party reunites to prepare for a battle with world...", AirDate: "2025-12-25", Runtime: 67},
				{EpisodeNumber: 8, SeasonNumber: 5, Title: "Chapter Eight: The Rightside Up", Overview: "As Vecna prepares to destroy the world as we know it, the party must put everything on the line t...", AirDate: "2025-12-31", Runtime: 129},
			}},
		},
	},
	{
		SeriesID: 82856,
		Seasons: []tmdb.NormalizedSeasonResult{
			{SeasonNumber: 1, Name: "Season 1", Overview: "Set after the fall of the Empire and before the emergence of the First Order....", AirDate: "2019-11-12", Episodes: []tmdb.NormalizedEpisodeResult{
				{EpisodeNumber: 1, SeasonNumber: 1, Title: "Chapter 1: The Mandalorian", Overview: "A Mandalorian bounty hunter tracks a target for a well-paying client.", AirDate: "2019-11-12", Runtime: 41},
				{EpisodeNumber: 2, SeasonNumber: 1, Title: "Chapter 2: The Child", Overview: "Target in-hand, the Mandalorian must now contend with scavengers.", AirDate: "2019-11-15", Runtime: 34},
				{EpisodeNumber: 3, SeasonNumber: 1, Title: "Chapter 3: The Sin", Overview: "The battered Mandalorian returns to his client for reward.", AirDate: "2019-11-22", Runtime: 39},
				{EpisodeNumber: 4, SeasonNumber: 1, Title: "Chapter 4: Sanctuary", Overview: "The Mandalorian teams up with an ex-soldier to protect a village from raiders.", AirDate: "2019-11-29", Runtime: 43},
				{EpisodeNumber: 5, SeasonNumber: 1, Title: "Chapter 5: The Gunslinger", Overview: "The Mandalorian helps a rookie bounty hunter who is in over his head.", AirDate: "2019-12-06", Runtime: 37},
				{EpisodeNumber: 6, SeasonNumber: 1, Title: "Chapter 6: The Prisoner", Overview: "The Mandalorian joins a crew of mercenaries on a dangerous mission.", AirDate: "2019-12-13", Runtime: 45},
				{EpisodeNumber: 7, SeasonNumber: 1, Title: "Chapter 7: The Reckoning", Overview: "An old rival extends an invitation for The Mandalorian to make peace.", AirDate: "2019-12-18", Runtime: 42},
				{EpisodeNumber: 8, SeasonNumber: 1, Title: "Chapter 8: Redemption", Overview: "The Mandalorian comes face-to-face with an unexpected enemy.", AirDate: "2019-12-27", Runtime: 50},
			}},
			{SeasonNumber: 2, Name: "Season 2", Overview: "The Mandalorian and the Child continue their journey, facing enemies and rall...", AirDate: "2020-10-30", Episodes: []tmdb.NormalizedEpisodeResult{
				{EpisodeNumber: 1, SeasonNumber: 2, Title: "Chapter 9: The Marshal", Overview: "The Mandalorian is drawn to the Outer Rim in search of others of his kind.", AirDate: "2020-10-30", Runtime: 56},
				{EpisodeNumber: 2, SeasonNumber: 2, Title: "Chapter 10: The Passenger", Overview: "The Mandalorian must ferry a passenger with precious cargo on a risky journey.", AirDate: "2020-11-06", Runtime: 43},
				{EpisodeNumber: 3, SeasonNumber: 2, Title: "Chapter 11: The Heiress", Overview: "The Mandalorian braves high seas and meets unexpected allies.", AirDate: "2020-11-13", Runtime: 37},
				{EpisodeNumber: 4, SeasonNumber: 2, Title: "Chapter 12: The Siege", Overview: "The Mandalorian rejoins old allies for a new mission.", AirDate: "2020-11-20", Runtime: 41},
				{EpisodeNumber: 5, SeasonNumber: 2, Title: "Chapter 13: The Jedi", Overview: "The Mandalorian journeys to a world ruled by a cruel magistrate who has made a powerful enemy.", AirDate: "2020-11-27", Runtime: 48},
				{EpisodeNumber: 6, SeasonNumber: 2, Title: "Chapter 14: The Tragedy", Overview: "The Mandalorian and Child travel to an ancient site.", AirDate: "2020-12-04", Runtime: 35},
				{EpisodeNumber: 7, SeasonNumber: 2, Title: "Chapter 15: The Believer", Overview: "To move against the Empire, the Mandalorian needs the help of an old enemy.", AirDate: "2020-12-11", Runtime: 40},
				{EpisodeNumber: 8, SeasonNumber: 2, Title: "Chapter 16: The Rescue", Overview: "The Mandalorian and his allies attempt a daring rescue.", AirDate: "2020-12-18", Runtime: 48},
			}},
			{SeasonNumber: 3, Name: "Season 3", Overview: "The journeys of the Mandalorian through the Star Wars galaxy continue. Once a...", AirDate: "2023-03-01", Episodes: []tmdb.NormalizedEpisodeResult{
				{EpisodeNumber: 1, SeasonNumber: 3, Title: "Chapter 17: The Apostate", Overview: "The Mandalorian begins an important journey.", AirDate: "2023-03-01", Runtime: 37},
				{EpisodeNumber: 2, SeasonNumber: 3, Title: "Chapter 18: The Mines of Mandalore", Overview: "The Mandalorian and Grogu explore the ruins of a destroyed planet.", AirDate: "2023-03-08", Runtime: 45},
				{EpisodeNumber: 3, SeasonNumber: 3, Title: "Chapter 19: The Convert", Overview: "On Coruscant, former Imperials find amnesty in the New Republic.", AirDate: "2023-03-15", Runtime: 58},
				{EpisodeNumber: 4, SeasonNumber: 3, Title: "Chapter 20: The Foundling", Overview: "Din Djarin returns to the hidden Mandalorian covert.", AirDate: "2023-03-22", Runtime: 33},
				{EpisodeNumber: 5, SeasonNumber: 3, Title: "Chapter 21: The Pirate", Overview: "The people of Nevarro need protection from rampant pirate attacks.", AirDate: "2023-03-29", Runtime: 44},
				{EpisodeNumber: 6, SeasonNumber: 3, Title: "Chapter 22: Guns for Hire", Overview: "The Mandalorian visits an opulent world.", AirDate: "2023-04-05", Runtime: 44},
				{EpisodeNumber: 7, SeasonNumber: 3, Title: "Chapter 23: The Spies", Overview: "Survivors come out of hiding.", AirDate: "2023-04-12", Runtime: 50},
				{EpisodeNumber: 8, SeasonNumber: 3, Title: "Chapter 24: The Return", Overview: "The Mandalorian and his allies confront their enemies.", AirDate: "2023-04-19", Runtime: 39},
			}},
		},
	},
	{
		SeriesID: 76479,
		Seasons: []tmdb.NormalizedSeasonResult{
			{SeasonNumber: 1, Name: "Season 1", Overview: "Superpowered individuals are recognized as superheroes, but in reality, abuse...", AirDate: "2019-07-25", Episodes: []tmdb.NormalizedEpisodeResult{
				{EpisodeNumber: 1, SeasonNumber: 1, Title: "The Name of the Game", Overview: "When a Supe kills the love of his life, A/V salesman Hughie Campbell teams up with Billy Butcher,...", AirDate: "2019-07-25", Runtime: 60},
				{EpisodeNumber: 2, SeasonNumber: 1, Title: "Cherry", Overview: "The Boys get themselves a Superhero, Starlight gets payback, Homelander gets naughty, and a Senat...", AirDate: "2019-07-25", Runtime: 59},
				{EpisodeNumber: 3, SeasonNumber: 1, Title: "Get Some", Overview: "It's the race of the century. A-Train versus Shockwave, vying for the title of World's Fastes...", AirDate: "2019-07-25", Runtime: 55},
				{EpisodeNumber: 4, SeasonNumber: 1, Title: "The Female of the Species", Overview: "On a very special episode of The Boys… an hour of guts, gutterballs, airplane hijackings, madne...", AirDate: "2019-07-25", Runtime: 56},
				{EpisodeNumber: 5, SeasonNumber: 1, Title: "Good for the Soul", Overview: "The Boys head to the Believe Expo to follow a promising lead in their ongoing war against the...", AirDate: "2019-07-25", Runtime: 60},
				{EpisodeNumber: 6, SeasonNumber: 1, Title: "The Innocents", Overview: "SUPER IN AMERICA (2019). Vought Studios. Genre: Reality. Starring: Homelander, Queen Maeve, Black...", AirDate: "2019-07-25", Runtime: 60},
				{EpisodeNumber: 7, SeasonNumber: 1, Title: "The Self-Preservation Society", Overview: "Never trust a washed-up Supe — the Boys learn this lesson the hard way. Meanwhile, Homelander d...", AirDate: "2019-07-25", Runtime: 56},
				{EpisodeNumber: 8, SeasonNumber: 1, Title: "You Found Me", Overview: "Season Finale Time! Questions answered! Secrets revealed! Conflicts… conflicted! Characters exp...", AirDate: "2019-07-25", Runtime: 66},
			}},
			{SeasonNumber: 2, Name: "Season 2", Overview: "The even more intense, more insane season two finds The Boys on the run from ...", AirDate: "2020-09-03", Episodes: []tmdb.NormalizedEpisodeResult{
				{EpisodeNumber: 1, SeasonNumber: 2, Title: "The Big Ride", Overview: "Season 2! New and improved! Now with 50% more explosive decapitations, terrorists, S&M hookers, c...", AirDate: "2020-09-03", Runtime: 63},
				{EpisodeNumber: 2, SeasonNumber: 2, Title: "Proper Preparation and Planning", Overview: "The Boys get themselves a Super Terrorist, Starlight gets evidence against Vought, The Deep gets ...", AirDate: "2020-09-03", Runtime: 59},
				{EpisodeNumber: 3, SeasonNumber: 2, Title: "Over the Hill with the Swords of a Thousand Men", Overview: "Attention: If you or a loved one were exposed to Compound V, you may be entitled to financial com...", AirDate: "2020-09-03", Runtime: 61},
				{EpisodeNumber: 4, SeasonNumber: 2, Title: "Nothing Like It in the World", Overview: "Road trip! The Boys head to North Carolina to follow a lead on a mysterious Supe named Liberty. A...", AirDate: "2020-09-10", Runtime: 70},
				{EpisodeNumber: 5, SeasonNumber: 2, Title: "We Gotta Go Now", Overview: "VoughtStudios is pleased to announce that filming has begun on #DawnOfTheSeven. 12 years of VCU m...", AirDate: "2020-09-17", Runtime: 63},
				{EpisodeNumber: 6, SeasonNumber: 2, Title: "The Bloody Doors Off", Overview: "The Sage Grove Center® is dedicated to caring for those struggling with mental illness. Our comp...", AirDate: "2020-09-24", Runtime: 66},
				{EpisodeNumber: 7, SeasonNumber: 2, Title: "Butcher, Baker, Candlestick Maker", Overview: "Congresswoman Victoria Neuman's sham Congressional Hearing against Vought takes place in 3 DAYS...", AirDate: "2020-10-01", Runtime: 56},
				{EpisodeNumber: 8, SeasonNumber: 2, Title: "What I Know", Overview: "***SUPER VILLAIN ALERT*** YOU ARE RECEIVING THIS NOTIFICATION FROM THE DEPARTMENT OF HOMELAND SEC...", AirDate: "2020-10-08", Runtime: 70},
			}},
			{SeasonNumber: 3, Name: "Season 3", Overview: "It's been a year of calm. Homelander's subdued. Butcher works for the gov...", AirDate: "2022-06-02", Episodes: []tmdb.NormalizedEpisodeResult{
				{EpisodeNumber: 1, SeasonNumber: 3, Title: "Payback", Overview: "You and a guest are invited to the premiere of DAWN OF THE SEVEN this Tuesday at 7PM in Vought To...", AirDate: "2022-06-02", Runtime: 64},
				{EpisodeNumber: 2, SeasonNumber: 3, Title: "The Only Man in the Sky", Overview: "Homelander. America's greatest Superhero. Defending our shores from sea to shining sea. Today, ...", AirDate: "2022-06-02", Runtime: 62},
				{EpisodeNumber: 3, SeasonNumber: 3, Title: "Barbary Coast", Overview: "Tonight at 9/8C on Vought Plus, it's the season finale of #AmericanHero! Three contestants rema...", AirDate: "2022-06-02", Runtime: 63},
				{EpisodeNumber: 4, SeasonNumber: 3, Title: "Glorious Five Year Plan", Overview: "Tonight, streaming live exclusively for Supeporn.com Super-Subscribers, it's the #ClashOfTheDil...", AirDate: "2022-06-09", Runtime: 62},
				{EpisodeNumber: 5, SeasonNumber: 3, Title: "The Last Time to Look on This World of Lies", Overview: "Did you know chimpanzees are an endangered species largely because of human activity? But you can...", AirDate: "2022-06-16", Runtime: 63},
				{EpisodeNumber: 6, SeasonNumber: 3, Title: "Herogasm", Overview: "You're invited to the 70th Annual Herogasm! You must present this invitation in order to be adm...", AirDate: "2022-06-23", Runtime: 63},
				{EpisodeNumber: 7, SeasonNumber: 3, Title: "Here Comes a Candle to Light You to Bed", Overview: "Did someone say birthday? Come celebrate at Buster Beaver's with our new Deluxe VIP Birthday Pa...", AirDate: "2022-06-30", Runtime: 67},
				{EpisodeNumber: 8, SeasonNumber: 3, Title: "The Instant White-Hot Wild", Overview: "Calling all patriots! Let's show Homelander we've got his back and we're not going to let S...", AirDate: "2022-07-07", Runtime: 64},
			}},
			{SeasonNumber: 4, Name: "Season 4", Overview: "The world is on the brink. Victoria Neuman is closer than ever to the Oval Of...", AirDate: "2024-06-13", Episodes: []tmdb.NormalizedEpisodeResult{
				{EpisodeNumber: 1, SeasonNumber: 4, Title: "Department of Dirty Tricks", Overview: "CALLING ALL PATRIOTS. BE AT THE COURTHOUSE TOMORROW FOR HOMELANDER'S #VERDICT AND BE READY. IF TH...", AirDate: "2024-06-13", Runtime: 66},
				{EpisodeNumber: 2, SeasonNumber: 4, Title: "Life Among the Septics", Overview: "Did you know globalists put chemicals in food to make us gay, Dakota Bob is a demon from hell, an...", AirDate: "2024-06-13", Runtime: 62},
				{EpisodeNumber: 3, SeasonNumber: 4, Title: "We'll Keep the Red Flag Flying Here", Overview: "This December at VoughtCoin Arena, experience the story of Christmas the way it was meant to be t...", AirDate: "2024-06-13", Runtime: 61},
				{EpisodeNumber: 4, SeasonNumber: 4, Title: "Wisdom of the Ages", Overview: "Vought News Network is proud to announce its new series #Truthbomb! Join host Firecracker and her...", AirDate: "2024-06-20", Runtime: 67},
				{EpisodeNumber: 5, SeasonNumber: 4, Title: "Beware the Jabberwock, My Son", Overview: "Attention #superfans! This year at #V52 see A-Train live and in person, as he presents an exclusi...", AirDate: "2024-06-27", Runtime: 69},
				{EpisodeNumber: 6, SeasonNumber: 4, Title: "Dirty Business", Overview: "Vernon Correctional Services provides compassionate rehabilitation to those in our care to prepar...", AirDate: "2024-07-04", Runtime: 66},
				{EpisodeNumber: 7, SeasonNumber: 4, Title: "The Insider", Overview: "Hey kids! Did you know your neighbor, uncle, or even Mom and Dad might be trying to destroy Ameri...", AirDate: "2024-07-11", Runtime: 65},
				{EpisodeNumber: 8, SeasonNumber: 4, Title: "Season Four Finale", Overview: "Calling all patriots! We will not allow this stolen election to be certified tomorrow! We must st...", AirDate: "2024-07-18", Runtime: 69},
			}},
			{SeasonNumber: 5, Name: "Season 5", Overview: "It's Homelander's world, completely subject to his erratic, egomaniacal w...", AirDate: "2026-04-08", Episodes: []tmdb.NormalizedEpisodeResult{
				{EpisodeNumber: 1, SeasonNumber: 5, Title: "Fifteen Inches of Sheer Dynamite", Overview: "", AirDate: "2026-04-08", Runtime: 0},
				{EpisodeNumber: 2, SeasonNumber: 5, Title: "Episode 2", Overview: "", AirDate: "2026-04-08", Runtime: 0},
				{EpisodeNumber: 3, SeasonNumber: 5, Title: "Episode 3", Overview: "", AirDate: "2026-04-15", Runtime: 0},
				{EpisodeNumber: 4, SeasonNumber: 5, Title: "Episode 4", Overview: "", AirDate: "2026-04-22", Runtime: 0},
				{EpisodeNumber: 5, SeasonNumber: 5, Title: "Episode 5", Overview: "", AirDate: "2026-04-29", Runtime: 0},
				{EpisodeNumber: 6, SeasonNumber: 5, Title: "Episode 6", Overview: "", AirDate: "2026-05-06", Runtime: 0},
				{EpisodeNumber: 7, SeasonNumber: 5, Title: "Episode 7", Overview: "", AirDate: "2026-05-13", Runtime: 0},
				{EpisodeNumber: 8, SeasonNumber: 5, Title: "Episode 8", Overview: "", AirDate: "2026-05-20", Runtime: 0},
			}},
		},
	},
}
