// Ported from Prowlarr's src/NzbDrone.Core/Indexers/Definitions/Avistaz/AvistazRequestGenerator.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

package avistaz

import (
	"slices"
	"strconv"
	"strings"

	"github.com/jatassi/slipstream/internal/newznab"
	"github.com/jatassi/slipstream/internal/ports/port"
	"github.com/jatassi/slipstream/internal/prowlarr/stringutil"
)

type requestGenerator struct{ site Site }

const searchURL = "api/v1/jackett/torrents"

func getBasicSearchParameters(ix *port.Indexer[Settings], searchCriteria port.SearchCriteria) stringutil.KeyValuePairs {
	categories := searchCriteria.Categories

	categoryMapping := ix.Categories.TrackerIDs(categories)

	limit := PageSize
	if searchCriteria.Limit > 0 {
		limit = min(PageSize, searchCriteria.Limit)
	}

	qc := stringutil.KeyValuePairs{
		{Key: "in", Value: "1"},
		{Key: "type", Value: firstIfSingleOrDefault(categoryMapping, "0")},
		{Key: "limit", Value: strconv.Itoa(limit)},
	}

	if searchCriteria.Limit > 0 && searchCriteria.Offset > 0 {
		page := searchCriteria.Offset/searchCriteria.Limit + 1
		qc.Add("page", strconv.Itoa(page))
	}

	if ix.Settings.FreeleechOnly {
		qc.Add("discount[]", "1")
	}

	// SlipStream: SearchCriteria carries no genre, so there is no tags parameter.

	if !slices.Contains(categories, newznab.Movies.ID) &&
		!slices.Contains(categories, newznab.TV.ID) &&
		!slices.Contains(categories, newznab.Audio.ID) {
		if slices.Contains(categories, newznab.MoviesUHD.ID) || slices.Contains(categories, newznab.TVUHD.ID) {
			qc.Add("video_quality[]", "6") // 2160p
		}

		if slices.Contains(categories, newznab.MoviesHD.ID) || slices.Contains(categories, newznab.TVHD.ID) {
			qc.Add("video_quality[]", "2") // 720p
			qc.Add("video_quality[]", "7") // 1080i
			qc.Add("video_quality[]", "3") // 1080p
		}

		if slices.Contains(categories, newznab.MoviesSD.ID) || slices.Contains(categories, newznab.TVSD.ID) {
			qc.Add("video_quality[]", "1") // SD
		}
	}

	return qc
}

func firstIfSingleOrDefault(s []string, def string) string {
	if len(s) == 1 {
		return s[0]
	}
	return def
}

// SlipStream: the framework sets the bearer token and hands every
// status to the parser, so the request carries neither.
func getRequest(ix *port.Indexer[Settings], searchParameters stringutil.KeyValuePairs) (port.Chain, error) {
	request := port.Get(ix.URL(searchURL)).Accept("application/json")
	for _, p := range searchParameters {
		request.AddQueryParam(p.Key, p.Value)
	}

	return port.One(request)
}

func (g requestGenerator) movie(ix *port.Indexer[Settings], searchCriteria port.SearchCriteria) (port.Chain, error) {
	parameters := g.site.GetBasicSearchParameters(ix, searchCriteria)

	switch {
	case searchCriteria.ImdbID != "":
		parameters.Add("imdb", searchCriteria.FullImdbID())
	case searchCriteria.TmdbID != 0:
		parameters.Add("tmdb", strconv.Itoa(searchCriteria.TmdbID))
	default:
		parameters.Add("search", strings.TrimSpace(searchCriteria.SanitizedSearchTerm()))
	}

	return getRequest(ix, parameters)
}

// SlipStream: no music search, which the framework doesn't run.

func (g requestGenerator) tv(ix *port.Indexer[Settings], searchCriteria port.SearchCriteria) (port.Chain, error) {
	parameters := g.site.GetBasicSearchParameters(ix, searchCriteria)

	switch {
	case searchCriteria.ImdbID != "":
		parameters.Add("imdb", searchCriteria.FullImdbID())
		parameters.Add("search", strings.TrimSpace(g.site.GetEpisodeSearchTerm(searchCriteria)))
	case searchCriteria.TvdbID != 0:
		parameters.Add("tvdb", strconv.Itoa(searchCriteria.TvdbID))
		parameters.Add("search", strings.TrimSpace(g.site.GetEpisodeSearchTerm(searchCriteria)))
	default:
		parameters.Add("search", strings.TrimSpace(searchCriteria.SanitizedSearchTerm()+" "+g.site.GetEpisodeSearchTerm(searchCriteria)))
	}

	return getRequest(ix, parameters)
}

// SlipStream: no book search, which upstream leaves empty.

func (g requestGenerator) basic(ix *port.Indexer[Settings], searchCriteria port.SearchCriteria) (port.Chain, error) {
	parameters := g.site.GetBasicSearchParameters(ix, searchCriteria)

	parameters.Add("search", strings.TrimSpace(searchCriteria.SanitizedSearchTerm()))

	return getRequest(ix, parameters)
}

func getEpisodeSearchTerm(searchCriteria port.SearchCriteria) string {
	return searchCriteria.EpisodeSearchString()
}
