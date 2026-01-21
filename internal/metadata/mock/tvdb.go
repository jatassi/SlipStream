package mock

import (
	"context"
	"fmt"
	"strings"

	"github.com/slipstream/slipstream/internal/metadata/tvdb"
)

// TVDBClient is a mock implementation of the TVDB client.
type TVDBClient struct{}

// NewTVDBClient creates a new mock TVDB client.
func NewTVDBClient() *TVDBClient {
	return &TVDBClient{}
}

func (c *TVDBClient) Name() string {
	return "tvdb-mock"
}

func (c *TVDBClient) IsConfigured() bool {
	return true
}

func (c *TVDBClient) Test(ctx context.Context) error {
	return nil
}

func (c *TVDBClient) SearchSeries(ctx context.Context, query string) ([]tvdb.NormalizedSeriesResult, error) {
	query = strings.ToLower(query)
	var results []tvdb.NormalizedSeriesResult

	for _, series := range tvdbMockSeries {
		if strings.Contains(strings.ToLower(series.Title), query) {
			results = append(results, series)
		}
	}

	if len(results) == 0 {
		limit := 10
		if len(tvdbMockSeries) < limit {
			limit = len(tvdbMockSeries)
		}
		results = tvdbMockSeries[:limit]
	}

	return results, nil
}

func (c *TVDBClient) GetSeries(ctx context.Context, id int) (*tvdb.NormalizedSeriesResult, error) {
	for _, series := range tvdbMockSeries {
		if series.ID == id || series.TvdbID == id {
			return &series, nil
		}
	}
	if len(tvdbMockSeries) > 0 {
		return &tvdbMockSeries[0], nil
	}
	return nil, fmt.Errorf("series not found")
}

func (c *TVDBClient) GetSeriesEpisodes(ctx context.Context, id int) ([]tvdb.NormalizedSeasonResult, error) {
	for _, series := range tvdbMockSeriesSeasons {
		if series.SeriesID == id {
			return series.Seasons, nil
		}
	}
	return tvdbDefaultSeasons, nil
}

type tvdbMockSeriesWithSeasons struct {
	SeriesID int
	Seasons  []tvdb.NormalizedSeasonResult
}

func tvdbGenerateEpisodes(season, count int) []tvdb.NormalizedEpisodeResult {
	episodes := make([]tvdb.NormalizedEpisodeResult, count)
	for i := 0; i < count; i++ {
		episodes[i] = tvdb.NormalizedEpisodeResult{
			EpisodeNumber: i + 1,
			SeasonNumber:  season,
			Title:         fmt.Sprintf("Episode %d", i+1),
			Overview:      "Episode overview placeholder.",
			AirDate:       "2020-01-01",
			Runtime:       45,
		}
	}
	return episodes
}

var tvdbMockSeriesSeasons = []tvdbMockSeriesWithSeasons{
	{
		SeriesID: 121361, // Game of Thrones
		Seasons: []tvdb.NormalizedSeasonResult{
			{SeasonNumber: 1, Name: "Season 1", Overview: "The first season introduces the Stark family.", AirDate: "2011-04-17", Episodes: tvdbGenerateEpisodes(1, 10)},
			{SeasonNumber: 2, Name: "Season 2", Overview: "War erupts in the Seven Kingdoms.", AirDate: "2012-04-01", Episodes: tvdbGenerateEpisodes(2, 10)},
			{SeasonNumber: 3, Name: "Season 3", Overview: "The War of Five Kings continues.", AirDate: "2013-03-31", Episodes: tvdbGenerateEpisodes(3, 10)},
		},
	},
	{
		SeriesID: 81189, // Breaking Bad
		Seasons: []tvdb.NormalizedSeasonResult{
			{SeasonNumber: 1, Name: "Season 1", Overview: "Walter White begins his transformation.", AirDate: "2008-01-20", Episodes: tvdbGenerateEpisodes(1, 7)},
			{SeasonNumber: 2, Name: "Season 2", Overview: "Walt and Jesse expand their operation.", AirDate: "2009-03-08", Episodes: tvdbGenerateEpisodes(2, 13)},
			{SeasonNumber: 3, Name: "Season 3", Overview: "The consequences of Walt's actions escalate.", AirDate: "2010-03-21", Episodes: tvdbGenerateEpisodes(3, 13)},
			{SeasonNumber: 4, Name: "Season 4", Overview: "Walt faces off against Gus Fring.", AirDate: "2011-07-17", Episodes: tvdbGenerateEpisodes(4, 13)},
			{SeasonNumber: 5, Name: "Season 5", Overview: "Walt builds his empire.", AirDate: "2012-07-15", Episodes: tvdbGenerateEpisodes(5, 16)},
		},
	},
	{
		SeriesID: 305288, // Stranger Things
		Seasons: []tvdb.NormalizedSeasonResult{
			{SeasonNumber: 1, Name: "Season 1", Overview: "A boy vanishes, and friends uncover supernatural secrets.", AirDate: "2016-07-15", Episodes: tvdbGenerateEpisodes(1, 8)},
			{SeasonNumber: 2, Name: "Season 2", Overview: "Will returns but strange things continue.", AirDate: "2017-10-27", Episodes: tvdbGenerateEpisodes(2, 9)},
			{SeasonNumber: 3, Name: "Season 3", Overview: "The summer of 1985 brings new threats.", AirDate: "2019-07-04", Episodes: tvdbGenerateEpisodes(3, 8)},
			{SeasonNumber: 4, Name: "Season 4", Overview: "Six months after the Battle of Starcourt.", AirDate: "2022-05-27", Episodes: tvdbGenerateEpisodes(4, 9)},
		},
	},
	{
		SeriesID: 355567, // The Boys
		Seasons: []tvdb.NormalizedSeasonResult{
			{SeasonNumber: 1, Name: "Season 1", Overview: "The Boys begins a war against Vought and the Seven.", AirDate: "2019-07-26", Episodes: tvdbGenerateEpisodes(1, 8)},
			{SeasonNumber: 2, Name: "Season 2", Overview: "The Boys are on the run and Homelander takes control.", AirDate: "2020-09-04", Episodes: tvdbGenerateEpisodes(2, 8)},
			{SeasonNumber: 3, Name: "Season 3", Overview: "The Boys search for a weapon against Homelander.", AirDate: "2022-06-03", Episodes: tvdbGenerateEpisodes(3, 8)},
			{SeasonNumber: 4, Name: "Season 4", Overview: "Homelander becomes more unhinged.", AirDate: "2024-06-13", Episodes: tvdbGenerateEpisodes(4, 8)},
		},
	},
	{
		SeriesID: 361753, // The Mandalorian
		Seasons: []tvdb.NormalizedSeasonResult{
			{SeasonNumber: 1, Name: "Season 1", Overview: "A lone bounty hunter protects a mysterious child.", AirDate: "2019-11-12", Episodes: tvdbGenerateEpisodes(1, 8)},
			{SeasonNumber: 2, Name: "Season 2", Overview: "The Mandalorian continues his quest to reunite Grogu with his kind.", AirDate: "2020-10-30", Episodes: tvdbGenerateEpisodes(2, 8)},
			{SeasonNumber: 3, Name: "Season 3", Overview: "The Mandalorian seeks to atone for his transgressions.", AirDate: "2023-03-01", Episodes: tvdbGenerateEpisodes(3, 8)},
		},
	},
	{
		SeriesID: 362472, // Loki
		Seasons: []tvdb.NormalizedSeasonResult{
			{SeasonNumber: 1, Name: "Season 1", Overview: "Loki finds himself at the TVA after stealing the Tesseract.", AirDate: "2021-06-09", Episodes: tvdbGenerateEpisodes(1, 6)},
			{SeasonNumber: 2, Name: "Season 2", Overview: "Loki and Mobius hunt down Sylvie and prevent the multiversal war.", AirDate: "2023-10-05", Episodes: tvdbGenerateEpisodes(2, 6)},
		},
	},
}

var tvdbDefaultSeasons = []tvdb.NormalizedSeasonResult{
	{SeasonNumber: 1, Name: "Season 1", Overview: "The first season.", AirDate: "2020-01-01", Episodes: tvdbGenerateEpisodes(1, 10)},
	{SeasonNumber: 2, Name: "Season 2", Overview: "The second season.", AirDate: "2021-01-01", Episodes: tvdbGenerateEpisodes(2, 10)},
}

// Real TVDB data fetched from API - DO NOT EDIT BELOW THIS LINE
var tvdbMockSeries = []tvdb.NormalizedSeriesResult{
	{ID: 121361, TvdbID: 121361, TmdbID: 1399, Title: "Game of Thrones", Year: 2011, Overview: "Seven noble families fight for control of the mythical land of Westeros. Friction between the houses leads to full-scale war. All while a very anci...", PosterURL: "https://artworks.thetvdb.com/banners/posters/121361-4.jpg", BackdropURL: "https://artworks.thetvdb.com/banners/fanart/original/121361-19.jpg", ImdbID: "tt0944947", Genres: []string{"Fantasy", "Drama", "Adventure", "Action"}, Status: "ended", Runtime: 57},
	{ID: 81189, TvdbID: 81189, TmdbID: 1396, Title: "Breaking Bad", Year: 2008, Overview: "When Walter White, a chemistry teacher, is diagnosed with Stage III cancer and given a prognosis of two years left to live, he chooses to enter a d...", PosterURL: "https://artworks.thetvdb.com/banners/posters/81189-10.jpg", BackdropURL: "https://artworks.thetvdb.com/banners/fanart/original/81189-21.jpg", ImdbID: "tt0903747", Genres: []string{"Drama", "Crime", "Thriller", "Western"}, Status: "ended", Runtime: 48},
	{ID: 305288, TvdbID: 305288, TmdbID: 66732, Title: "Stranger Things", Year: 2016, Overview: "When a young boy vanishes, a small town uncovers a mystery involving secret experiments, terrifying supernatural forces and one strange little girl.", PosterURL: "https://artworks.thetvdb.com/banners/posters/305288-4.jpg", BackdropURL: "https://artworks.thetvdb.com/banners/v4/series/305288/backgrounds/62907d929e73d.jpg", ImdbID: "tt4574334", Genres: []string{"Science Fiction", "Horror", "Fantasy", "Drama", "Adventure", "Suspense", "Thriller", "Mystery"}, Status: "ended", Runtime: 65},
	{ID: 371572, TvdbID: 371572, TmdbID: 94997, Title: "House of the Dragon", Year: 2022, Overview: "The story of House Targaryen, 200 years before the events of Game of Thrones.", PosterURL: "https://artworks.thetvdb.com/banners/v4/series/371572/posters/624838567e159.jpg", BackdropURL: "https://artworks.thetvdb.com/banners/v4/series/371572/backgrounds/62d27eb474e3c.jpg", ImdbID: "tt11198330", Genres: []string{"Fantasy", "Drama", "Adventure", "Action", "War"}, Status: "continuing", Runtime: 63},
	{ID: 279121, TvdbID: 279121, TmdbID: 60735, Title: "The Flash (2014)", Year: 2014, Overview: "After being struck by lightning, Barry Allen wakes up from his coma to discover he's been given the power of super speed, becoming the Flash, fight...", PosterURL: "https://artworks.thetvdb.com/banners/posters/279121-5.jpg", BackdropURL: "https://artworks.thetvdb.com/banners/fanart/original/279121-9.jpg", ImdbID: "tt3107288", Genres: []string{"Science Fiction", "Drama", "Adventure", "Action"}, Status: "ended", Runtime: 42},
	// ERROR fetching series 369459: series not found
	{ID: 80379, TvdbID: 80379, TmdbID: 1418, Title: "The Big Bang Theory", Year: 2007, Overview: "A woman who moves into an apartment across the hall from two brilliant but socially awkward physicists shows them how little they know about life o...", PosterURL: "https://artworks.thetvdb.com/banners/posters/80379-18.jpg", BackdropURL: "https://artworks.thetvdb.com/banners/fanart/original/80379-2.jpg", ImdbID: "tt0898266", Genres: []string{"Comedy", "Romance"}, Status: "ended", Runtime: 25},
	{ID: 75760, TvdbID: 75760, TmdbID: 1100, Title: "How I Met Your Mother", Year: 2005, Overview: "A father tells his children, through a series of flashbacks, the journey that he and his four best friends undertook, which would lead him to meet ...", PosterURL: "https://artworks.thetvdb.com/banners/posters/75760-34.jpg", BackdropURL: "https://artworks.thetvdb.com/banners/fanart/original/75760-50.jpg", ImdbID: "tt0460649", Genres: []string{"Comedy", "Romance"}, Status: "ended", Runtime: 25},
	{ID: 71663, TvdbID: 71663, TmdbID: 456, Title: "The Simpsons", Year: 1989, Overview: "Set in Springfield, the average American town, the show focuses on the antics and everyday adventures of the Simpson family; Homer, Marge, Bart, Li...", PosterURL: "https://artworks.thetvdb.com/banners/posters/71663-15.jpg", BackdropURL: "https://artworks.thetvdb.com/banners/fanart/original/71663-10.jpg", ImdbID: "tt0096697", Genres: []string{"Comedy", "Animation"}, Status: "continuing", Runtime: 24},
	{ID: 75897, TvdbID: 75897, TmdbID: 2190, Title: "South Park", Year: 1997, Overview: "South Park is an animated series featuring four boys who live in the Colorado town of South Park, which is beset by frequent odd occurrences. The s...", PosterURL: "https://artworks.thetvdb.com/banners/posters/75897-5.jpg", BackdropURL: "https://artworks.thetvdb.com/banners/fanart/original/75897-43.jpg", ImdbID: "tt0121955", Genres: []string{"Comedy", "Animation"}, Status: "continuing", Runtime: 22},
	{ID: 79168, TvdbID: 79168, TmdbID: 1668, Title: "Friends", Year: 1994, Overview: "Rachel Green, Ross Geller, Monica Geller, Joey Tribbiani, Chandler Bing and Phoebe Buffay are six 20 something year-olds, living off one another in...", PosterURL: "https://artworks.thetvdb.com/banners/posters/79168-27.jpg", BackdropURL: "https://artworks.thetvdb.com/banners/fanart/original/79168-6.jpg", ImdbID: "tt0108778", Genres: []string{"Comedy"}, Status: "ended", Runtime: 22},
	{ID: 153021, TvdbID: 153021, TmdbID: 1402, Title: "The Walking Dead", Year: 2010, Overview: "The world we knew is gone. An epidemic of apocalyptic proportions has swept the globe causing the dead to rise and feed on the living. In a matter ...", PosterURL: "https://artworks.thetvdb.com/banners/v4/series/153021/posters/60fd8577d1a96.jpg", BackdropURL: "https://artworks.thetvdb.com/banners/fanart/original/153021-83.jpg", ImdbID: "tt1520211", Genres: []string{"Horror", "Drama", "Adventure", "Thriller"}, Status: "ended", Runtime: 46},
	{ID: 327417, TvdbID: 327417, TmdbID: 71446, Title: "La casa de papel", Year: 2017, Overview: "Un golpe maestro ideado y perfeccionado durante años, planificado durante meses y ejecutado en pocos minutos para que el elegido grupo de ladrones...", PosterURL: "https://artworks.thetvdb.com/banners/posters/5d30cc0b4a75d.jpg", BackdropURL: "https://artworks.thetvdb.com/banners/series/327417/backgrounds/5e75f224ac334.jpg", ImdbID: "tt6468322", Genres: []string{"Drama", "Crime", "Action", "Thriller", "Mystery"}, Status: "ended", Runtime: 57},
	{ID: 355567, TvdbID: 355567, TmdbID: 76479, Title: "The Boys", Year: 2019, Overview: "In a world where superheroes embrace the darker side of their massive celebrity and fame, a group of vigilantes known informally as \"The Boys\" set ...", PosterURL: "https://artworks.thetvdb.com/banners/posters/5c5c402b075cc.jpg", BackdropURL: "https://artworks.thetvdb.com/banners/fanart/original/5d120186c0c5f.jpg", ImdbID: "tt1190634", Genres: []string{"Science Fiction", "Fantasy", "Drama", "Crime", "Comedy", "Action"}, Status: "continuing", Runtime: 63},
	{ID: 361753, TvdbID: 361753, TmdbID: 82856, Title: "The Mandalorian", Year: 2019, Overview: "After the fall of the Galactic Empire, lawlessness has spread throughout the galaxy. A lone gunfighter makes his way through the outer reaches, earning his keep as a bounty hunter.", PosterURL: "https://artworks.thetvdb.com/banners/v4/series/361753/posters/5d6d8722680d0.jpg", BackdropURL: "https://artworks.thetvdb.com/banners/v4/series/361753/backgrounds/5e5d4c0c9f0f5.jpg", ImdbID: "tt8111088", Genres: []string{"Science Fiction", "Adventure", "Action", "Western"}, Status: "continuing", Runtime: 40},
	{ID: 362472, TvdbID: 362472, TmdbID: 84958, Title: "Loki", Year: 2021, Overview: "After stealing the Tesseract in Avengers: Endgame, Loki lands before the Time Variance Authority and is forced to fix the timeline.", PosterURL: "https://artworks.thetvdb.com/banners/v4/series/362472/posters/60fd7ef29e24e.jpg", BackdropURL: "https://artworks.thetvdb.com/banners/v4/series/362472/backgrounds/60c87ef5e2dd6.jpg", ImdbID: "tt9140554", Genres: []string{"Science Fiction", "Fantasy", "Drama", "Adventure", "Action"}, Status: "ended", Runtime: 50},
}
