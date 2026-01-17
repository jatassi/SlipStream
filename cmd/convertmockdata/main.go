// Script to convert mock_indexer_results.txt to Go map format
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
)

type Release struct {
	TmdbID int `json:"tmdbId"`
	TvdbID int `json:"tvdbId"`
}

func main() {
	data, err := os.ReadFile("data/mock_indexer_results.txt")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}

	content := string(data)

	// Find all JSON arrays
	re := regexp.MustCompile(`(?s)// === ([^(]+) \((\w+)\) ===.*?var \w+Results = ` + "`" + `(\[.*?\])` + "`")
	matches := re.FindAllStringSubmatch(content, -1)

	movieResults := make(map[int]string)
	tvResults := make(map[int]string)

	for _, match := range matches {
		if len(match) < 4 {
			continue
		}
		name := strings.TrimSpace(match[1])
		searchType := match[2]
		jsonData := match[3]

		// Parse the JSON to get the first result's TMDB/TVDB ID
		var releases []Release
		if err := json.Unmarshal([]byte(jsonData), &releases); err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing JSON for %s: %v\n", name, err)
			continue
		}

		if len(releases) == 0 {
			continue
		}

		// Use first result's ID as the key
		if searchType == "movie" && releases[0].TmdbID > 0 {
			movieResults[releases[0].TmdbID] = jsonData
			fmt.Fprintf(os.Stderr, "Movie: %s -> TMDB %d (%d results)\n", name, releases[0].TmdbID, len(releases))
		} else if searchType == "tvsearch" && releases[0].TvdbID > 0 {
			tvResults[releases[0].TvdbID] = jsonData
			fmt.Fprintf(os.Stderr, "TV: %s -> TVDB %d (%d results)\n", name, releases[0].TvdbID, len(releases))
		}
	}

	// Output Go code
	fmt.Println("package mock")
	fmt.Println()
	fmt.Println("// movieResultsJSON contains pre-loaded search results keyed by TMDB ID.")
	fmt.Println("var movieResultsJSON = map[int]string{")
	for id, jsonData := range movieResults {
		// Escape backticks in JSON
		escaped := strings.ReplaceAll(jsonData, "`", "` + \"`\" + `")
		fmt.Printf("\t%d: `%s`,\n", id, escaped)
	}
	fmt.Println("}")
	fmt.Println()
	fmt.Println("// tvResultsJSON contains pre-loaded search results keyed by TVDB ID.")
	fmt.Println("var tvResultsJSON = map[int]string{")
	for id, jsonData := range tvResults {
		escaped := strings.ReplaceAll(jsonData, "`", "` + \"`\" + `")
		fmt.Printf("\t%d: `%s`,\n", id, escaped)
	}
	fmt.Println("}")
}
