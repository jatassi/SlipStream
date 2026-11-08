// Command record fetches real movies from TMDB and real series from TVDB
// and TMDB once, with their ratings from OMDb, writes them into a catalog
// Scenario's tmdb, tvdb and omdb keys and saves their images in the
// artwork store, so the Fake world serves them offline. One run records
// every movie and series the Scenario holds, and rewrites those keys from
// exactly the ids it names. Run it from the repository root:
//
//	TMDB_API_KEY=… OMDB_API_KEY=… TVDB_API_KEY=… go run ./fakes/scenario/record -scenario fakes/scenario/catalog/real.yaml -movies 603,550 -series 81189
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"
)

func main() {
	c, err := parse(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, "record:", err)
		os.Exit(2)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	err = run(ctx, c)
	stop()
	if err != nil {
		fmt.Fprintln(os.Stderr, "record:", err)
		os.Exit(1)
	}
}

func parse(args []string) (config, error) {
	c := config{
		tmdbKey: os.Getenv("TMDB_API_KEY"), omdbKey: os.Getenv("OMDB_API_KEY"), tvdbKey: os.Getenv("TVDB_API_KEY"), tvdbPin: os.Getenv("TVDB_PIN"),
		client: &http.Client{Timeout: time.Minute},
	}
	var movies, series string
	fs := flag.NewFlagSet("record", flag.ContinueOnError)
	fs.StringVar(&c.scenario, "scenario", "", "the catalog Scenario file whose tmdb, tvdb and omdb keys to rewrite")
	fs.StringVar(&movies, "movies", "", "the TMDB ids of the movies to record, comma-separated")
	fs.StringVar(&series, "series", "", "the TVDB ids of the series to record, comma-separated")
	fs.StringVar(&c.store, "store", "fakes/internal/artwork/files", "the artwork store directory")
	fs.StringVar(&c.tmdbAPI, "tmdb-api", "https://api.themoviedb.org/3", "TMDB's API root")
	fs.StringVar(&c.tmdbImages, "tmdb-images", "https://image.tmdb.org/t/p", "TMDB's image host root")
	fs.StringVar(&c.omdbAPI, "omdb-api", "https://www.omdbapi.com/", "OMDb's API root")
	fs.StringVar(&c.tvdbAPI, "tvdb-api", "https://api4.thetvdb.com/v4", "TVDB's API root")
	fs.StringVar(&c.tvdbArtwork, "tvdb-artwork", "https://artworks.thetvdb.com", "TVDB's artwork host")
	fs.Usage = func() {
		_, _ = fmt.Fprintf(fs.Output(), "usage: record -scenario FILE -movies TMDB_ID,... -series TVDB_ID,...\n\n"+
			"TMDB_API_KEY, OMDB_API_KEY and TVDB_API_KEY hold the API keys, and TVDB_PIN a TVDB subscriber PIN.\n\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return config{}, err //nolint:wrapcheck // the flag package's words name the flag
	}
	var err error
	if c.movies, err = ids(movies, "TMDB movie"); err != nil {
		return config{}, err
	}
	if c.series, err = ids(series, "TVDB series"); err != nil {
		return config{}, err
	}
	switch {
	case c.scenario == "" || len(c.movies)+len(c.series) == 0 || fs.NArg() > 0:
		fs.Usage()
		return config{}, errors.New("name a Scenario file and at least one id with -movies or -series")
	case c.tmdbKey == "" || c.omdbKey == "" || len(c.series) > 0 && c.tvdbKey == "":
		return config{}, errors.New("set TMDB_API_KEY, OMDB_API_KEY and, for series, TVDB_API_KEY")
	}
	return c, nil
}

func ids(list, what string) ([]int64, error) {
	var out []int64
	for field := range strings.SplitSeq(list, ",") {
		if field = strings.TrimSpace(field); field == "" {
			continue
		}
		id, err := strconv.ParseInt(field, 10, 64)
		if err != nil || id < 1 {
			return nil, fmt.Errorf("%q isn't a %s id", field, what)
		}
		out = append(out, id)
	}
	return out, nil
}
