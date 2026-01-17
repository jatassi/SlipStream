// Command fetchmockdata fetches real data from TMDB, TVDB, and indexer APIs
// to generate mock data for developer mode.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/config"
	"github.com/slipstream/slipstream/internal/database"
	"github.com/slipstream/slipstream/internal/indexer"
	"github.com/slipstream/slipstream/internal/indexer/cardigann"
	"github.com/slipstream/slipstream/internal/indexer/search"
	"github.com/slipstream/slipstream/internal/indexer/types"
	"github.com/slipstream/slipstream/internal/metadata/tmdb"
	"github.com/slipstream/slipstream/internal/metadata/tvdb"
)

func main() {
	// Use a no-op logger to suppress debug output when running the script
	logger := zerolog.Nop()

	// Load config to get API keys
	cfg, err := config.Load("")
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to load config")
	}

	ctx := context.Background()

	// Check which mode to run
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "tmdb":
			fetchTMDBData(ctx, cfg.Metadata.TMDB, logger)
			return
		case "tvdb":
			fetchTVDBData(ctx, cfg.Metadata.TVDB, logger)
			return
		case "indexer":
			fetchIndexerData(ctx, cfg, logger)
			return
		}
	}

	// Default: run all
	fmt.Println("=== FETCHING TMDB DATA ===")
	fetchTMDBData(ctx, cfg.Metadata.TMDB, logger)

	fmt.Println("\n\n=== FETCHING TVDB DATA ===")
	fetchTVDBData(ctx, cfg.Metadata.TVDB, logger)

	fmt.Println("\n\n=== FETCHING INDEXER DATA ===")
	fetchIndexerData(ctx, cfg, logger)
}

func fetchTMDBData(ctx context.Context, cfg config.TMDBConfig, logger zerolog.Logger) {
	client := tmdb.NewClient(cfg, logger)

	if !client.IsConfigured() {
		fmt.Println("TMDB not configured!")
		return
	}

	// Popular movie IDs to fetch
	movieIDs := []int{
		603,    // The Matrix
		550,    // Fight Club
		680,    // Pulp Fiction
		155,    // The Dark Knight
		278,    // The Shawshank Redemption
		238,    // The Godfather
		27205,  // Inception
		157336, // Interstellar
		120,    // LOTR: Fellowship
		24428,  // The Avengers
		299536, // Avengers: Infinity War
		299534, // Avengers: Endgame
		569094, // Spider-Man: Across the Spider-Verse
		438631, // Dune
		693134, // Dune: Part Two
		359724, // Ford v Ferrari
		346698, // Barbie
		872585, // Oppenheimer
		76600,  // Avatar: The Way of Water
		19995,  // Avatar
		533535, // Deadpool & Wolverine
		545611, // Everything Everywhere All at Once
		447365, // Guardians of the Galaxy Vol. 3
		912649, // Venom: The Last Dance
		1022789, // Inside Out 2
	}

	fmt.Println("\n// TMDB Movies - Copy this to mock/tmdb.go")
	fmt.Println("var mockMovies = []tmdb.NormalizedMovieResult{")

	for _, id := range movieIDs {
		movie, err := client.GetMovie(ctx, id)
		if err != nil {
			fmt.Printf("\t// ERROR fetching movie %d: %v\n", id, err)
			continue
		}

		// Fetch release dates
		digital, physical, _ := client.GetMovieReleaseDates(ctx, id)

		genres := formatStringSlice(movie.Genres)
		fmt.Printf("\t{ID: %d, Title: %q, Year: %d, Overview: %q, PosterURL: %q, BackdropURL: %q, ImdbID: %q, Genres: %s, Runtime: %d, ReleaseDate: %q, DigitalReleaseDate: %q, PhysicalReleaseDate: %q},\n",
			movie.ID, movie.Title, movie.Year, truncate(movie.Overview, 150), movie.PosterURL, movie.BackdropURL, movie.ImdbID, genres, movie.Runtime, movie.ReleaseDate, digital, physical)

		time.Sleep(250 * time.Millisecond) // Rate limiting
	}
	fmt.Println("}")

	// Popular series IDs to fetch
	seriesIDs := []int{
		1399,   // Game of Thrones
		1396,   // Breaking Bad
		66732,  // Stranger Things
		94997,  // House of the Dragon
		60735,  // The Flash
		84958,  // Loki
		1418,   // The Big Bang Theory
		1100,   // How I Met Your Mother
		456,    // The Simpsons
		2190,   // South Park
		1668,   // Friends
		1402,   // The Walking Dead
		71446,  // Money Heist
		76479,  // The Boys
		93405,  // Squid Game
		100088, // The Last of Us
		60059,  // Better Call Saul
		63174,  // Lucifer
		82856,  // The Mandalorian
		95557,  // Invincible
		73586,  // Yellowstone
		85271,  // WandaVision
		114461, // Ahsoka
		94605,  // Arcane
	}

	fmt.Println("\n// TMDB Series - Copy this to mock/tmdb.go")
	fmt.Println("var mockSeries = []tmdb.NormalizedSeriesResult{")

	for _, id := range seriesIDs {
		series, err := client.GetSeries(ctx, id)
		if err != nil {
			fmt.Printf("\t// ERROR fetching series %d: %v\n", id, err)
			continue
		}

		genres := formatStringSlice(series.Genres)
		fmt.Printf("\t{ID: %d, TmdbID: %d, Title: %q, Year: %d, Overview: %q, PosterURL: %q, BackdropURL: %q, ImdbID: %q, TvdbID: %d, Genres: %s, Status: %q, Runtime: %d, Network: %q},\n",
			series.ID, series.TmdbID, series.Title, series.Year, truncate(series.Overview, 150), series.PosterURL, series.BackdropURL, series.ImdbID, series.TvdbID, genres, series.Status, series.Runtime, series.Network)

		time.Sleep(250 * time.Millisecond)
	}
	fmt.Println("}")

	// Fetch seasons for series used in dev mode (TMDB IDs)
	// These match the series in populateMockSeries in server.go
	seriesWithSeasons := []int{
		1396,  // Breaking Bad (TVDB 81189)
		1399,  // Game of Thrones (TVDB 121361)
		66732, // Stranger Things (TVDB 305288)
		82856, // The Mandalorian (TVDB 361753)
		76479, // The Boys (TVDB 355567)
	}
	fmt.Println("\n// TMDB Series Seasons - Copy this to mock/tmdb.go")
	fmt.Println("var mockSeriesSeasons = []mockSeriesWithSeasons{")

	for _, id := range seriesWithSeasons {
		seasons, err := client.GetAllSeasons(ctx, id)
		if err != nil {
			fmt.Printf("\t// ERROR fetching seasons for %d: %v\n", id, err)
			continue
		}

		fmt.Printf("\t{\n\t\tSeriesID: %d,\n\t\tSeasons: []tmdb.NormalizedSeasonResult{\n", id)
		for _, season := range seasons {
			if season.SeasonNumber == 0 {
				continue // Skip specials
			}
			// Output season with real episode data
			fmt.Printf("\t\t\t{SeasonNumber: %d, Name: %q, Overview: %q, AirDate: %q, Episodes: []tmdb.NormalizedEpisodeResult{\n",
				season.SeasonNumber, season.Name, truncate(season.Overview, 80), season.AirDate)
			for _, ep := range season.Episodes {
				fmt.Printf("\t\t\t\t{EpisodeNumber: %d, SeasonNumber: %d, Title: %q, Overview: %q, AirDate: %q, Runtime: %d},\n",
					ep.EpisodeNumber, ep.SeasonNumber, ep.Title, truncate(ep.Overview, 100), ep.AirDate, ep.Runtime)
			}
			fmt.Println("\t\t\t}},")
		}
		fmt.Println("\t\t},")
		fmt.Println("\t},")

		time.Sleep(500 * time.Millisecond)
	}
	fmt.Println("}")
}

func fetchTVDBData(ctx context.Context, cfg config.TVDBConfig, logger zerolog.Logger) {
	client := tvdb.NewClient(cfg, logger)

	if !client.IsConfigured() {
		fmt.Println("TVDB not configured!")
		return
	}

	// Test connectivity
	if err := client.Test(ctx); err != nil {
		fmt.Printf("TVDB test failed: %v\n", err)
		return
	}

	// TVDB IDs for series (matching TMDB series above where possible)
	seriesIDs := []int{
		121361, // Game of Thrones
		81189,  // Breaking Bad
		305288, // Stranger Things
		371572, // House of the Dragon
		279121, // The Flash
		369459, // Loki
		80379,  // The Big Bang Theory
		75760,  // How I Met Your Mother
		71663,  // The Simpsons
		75897,  // South Park
		79168,  // Friends
		153021, // The Walking Dead
		327417, // Money Heist
		355567, // The Boys
		386050, // Squid Game
	}

	fmt.Println("\n// TVDB Series - Copy this to mock/tvdb.go")
	fmt.Println("var tvdbMockSeries = []tvdb.NormalizedSeriesResult{")

	for _, id := range seriesIDs {
		series, err := client.GetSeries(ctx, id)
		if err != nil {
			fmt.Printf("\t// ERROR fetching series %d: %v\n", id, err)
			continue
		}

		genres := formatStringSlice(series.Genres)
		fmt.Printf("\t{ID: %d, TvdbID: %d, TmdbID: %d, Title: %q, Year: %d, Overview: %q, PosterURL: %q, BackdropURL: %q, ImdbID: %q, Genres: %s, Status: %q, Runtime: %d},\n",
			series.ID, series.TvdbID, series.TmdbID, series.Title, series.Year, truncate(series.Overview, 150), series.PosterURL, series.BackdropURL, series.ImdbID, genres, series.Status, series.Runtime)

		time.Sleep(500 * time.Millisecond)
	}
	fmt.Println("}")
}

func fetchIndexerData(ctx context.Context, cfg *config.Config, logger zerolog.Logger) {
	// Open database
	db, err := database.New(cfg.Database.Path)
	if err != nil {
		fmt.Printf("Failed to open database: %v\n", err)
		return
	}
	defer db.Close()

	// Initialize cardigann manager
	cardigannManager, err := cardigann.NewManager(cardigann.DefaultManagerConfig(), logger)
	if err != nil {
		fmt.Printf("Failed to create cardigann manager: %v\n", err)
		return
	}

	// Initialize indexer service
	indexerService := indexer.NewService(db.Conn(), cardigannManager, logger)

	// Initialize search service
	searchService := search.NewService(indexerService, logger)

	// Search queries
	queries := []struct {
		name  string
		query string
		stype string // movie or tvsearch
	}{
		{"matrix", "The Matrix", "movie"},
		{"inception", "Inception", "movie"},
		{"dune", "Dune", "movie"},
		{"oppenheimer", "Oppenheimer", "movie"},
		{"barbie", "Barbie", "movie"},
		{"got", "Game of Thrones", "tvsearch"},
		{"breaking_bad", "Breaking Bad", "tvsearch"},
		{"stranger_things", "Stranger Things", "tvsearch"},
		{"the_boys", "The Boys", "tvsearch"},
		{"mandalorian", "The Mandalorian", "tvsearch"},
	}

	fmt.Println("\n// Indexer Search Results - Raw JSON output")
	fmt.Println("// Each query's results are printed as JSON for inspection")

	for _, q := range queries {
		fmt.Printf("\n// === %s (%s) ===\n", q.query, q.stype)

		criteria := types.SearchCriteria{
			Query: q.query,
			Type:  q.stype,
		}

		result, err := searchService.Search(ctx, criteria)
		if err != nil {
			fmt.Printf("// ERROR: %v\n", err)
			continue
		}

		fmt.Printf("// Total results: %d from %d indexers\n", result.TotalResults, result.IndexersUsed)

		if len(result.IndexerErrors) > 0 {
			for _, e := range result.IndexerErrors {
				fmt.Printf("// Indexer error (%s): %s\n", e.IndexerName, e.Error)
			}
		}

		// Output first 10 results as JSON
		limit := 10
		if len(result.Releases) < limit {
			limit = len(result.Releases)
		}

		if limit > 0 {
			subset := result.Releases[:limit]
			jsonData, _ := json.MarshalIndent(subset, "", "  ")
			fmt.Printf("var %sResults = `%s`\n", q.name, string(jsonData))
		}

		time.Sleep(1 * time.Second) // Rate limiting
	}
}

func formatStringSlice(s []string) string {
	if len(s) == 0 {
		return "nil"
	}
	quoted := make([]string, len(s))
	for i, v := range s {
		quoted[i] = fmt.Sprintf("%q", v)
	}
	return "[]string{" + strings.Join(quoted, ", ") + "}"
}

func truncate(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", "")
	if len(s) > maxLen {
		return s[:maxLen-3] + "..."
	}
	return s
}

func prettyJSON(v interface{}) string {
	b, _ := json.MarshalIndent(v, "", "  ")
	return string(b)
}
