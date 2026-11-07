// Command record fetches real movies from TMDB and their ratings from
// OMDb once, writes them into a catalog Scenario's tmdb and omdb keys and
// saves their images in the artwork store, so the Fake world serves them
// offline. Run it from the repository root:
//
//	TMDB_API_KEY=… OMDB_API_KEY=… go run ./fakes/scenario/record -scenario fakes/scenario/catalog/real.yaml 603 550
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
	c := config{tmdbKey: os.Getenv("TMDB_API_KEY"), omdbKey: os.Getenv("OMDB_API_KEY"), client: &http.Client{Timeout: time.Minute}}
	fs := flag.NewFlagSet("record", flag.ContinueOnError)
	fs.StringVar(&c.scenario, "scenario", "", "the catalog Scenario file whose tmdb and omdb keys to rewrite")
	fs.StringVar(&c.store, "store", "fakes/internal/artwork/files", "the artwork store directory")
	fs.StringVar(&c.tmdbAPI, "tmdb-api", "https://api.themoviedb.org/3", "TMDB's API root")
	fs.StringVar(&c.tmdbImages, "tmdb-images", "https://image.tmdb.org/t/p", "TMDB's image host root")
	fs.StringVar(&c.omdbAPI, "omdb-api", "https://www.omdbapi.com/", "OMDb's API root")
	fs.Usage = func() {
		_, _ = fmt.Fprintf(fs.Output(), "usage: record -scenario FILE TMDB_ID...\n\nTMDB_API_KEY and OMDB_API_KEY hold the API keys.\n\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return config{}, err //nolint:wrapcheck // the flag package's words name the flag
	}
	for _, arg := range fs.Args() {
		id, err := strconv.ParseInt(arg, 10, 64)
		if err != nil || id < 1 {
			return config{}, fmt.Errorf("%q isn't a TMDB movie id", arg)
		}
		c.ids = append(c.ids, id)
	}
	switch {
	case c.scenario == "" || len(c.ids) == 0:
		fs.Usage()
		return config{}, errors.New("name a Scenario file and at least one TMDB movie id")
	case c.tmdbKey == "" || c.omdbKey == "":
		return config{}, errors.New("set TMDB_API_KEY and OMDB_API_KEY")
	}
	return c, nil
}
