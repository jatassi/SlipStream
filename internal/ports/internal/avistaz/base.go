// Ported from Prowlarr's src/NzbDrone.Core/Indexers/Definitions/Avistaz/AvistazBase.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

// Package avistaz is AvistazBase, the family of AvistaZ, CinemaZ,
// PrivateHD, AnimeZ and ExoticaZ: a sign-in with username, password and
// PID to a bearer token, one search API, categories by type and video
// quality, and a minimum seed time that grows with size. A site's
// package declares its Entry and the members its C# class overrides,
// and Define does the rest.
package avistaz

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/newznab"
	"github.com/jatassi/slipstream/internal/ports/port"
	"github.com/jatassi/slipstream/internal/prowlarr/stringutil"
)

const (
	PageSize  = 50
	rateLimit = 6100 * time.Millisecond
	loginURL  = "api/v1/jackett/auth"
)

// Site is what a site's C# class overrides, its request generator's and
// parser's included. A nil member is the base's.
type Site struct {
	// GetBasicSearchParameters is AnimeZ's format[] parameters.
	GetBasicSearchParameters func(ix *port.Indexer[Settings], searchCriteria port.SearchCriteria) stringutil.KeyValuePairs
	// GetEpisodeSearchTerm is AvistaZ's episode without a season.
	GetEpisodeSearchTerm func(searchCriteria port.SearchCriteria) string
	// ParseCategories is AnimeZ's and ExoticaZ's mapped categories.
	ParseCategories func(ix *port.Indexer[Settings], row AvistazRelease) ([]newznab.Category, error)
	// ParseTitle is AnimeZ's release title.
	ParseTitle func(row AvistazRelease) string
	// ModifyRequest runs after the bearer token is set, on every request
	// to the Indexer's hosts: AnimeZ's GetDownloadRequest asks for JSON.
	ModifyRequest func(ix *port.Indexer[Settings], request *port.Request)
}

// Define is the site's Port: e with the base's rate limit and PageSize,
// and the modes its Capabilities declare.
func Define(e port.Entry, s Site) port.Port {
	e.RateLimit = rateLimit
	e.PageSize = PageSize
	if s.GetBasicSearchParameters == nil {
		s.GetBasicSearchParameters = getBasicSearchParameters
	}
	if s.GetEpisodeSearchTerm == nil {
		s.GetEpisodeSearchTerm = getEpisodeSearchTerm
	}
	if s.ParseCategories == nil {
		s.ParseCategories = parseCategories
	}
	if s.ParseTitle == nil {
		s.ParseTitle = parseTitle
	}
	g := requestGenerator{site: s}
	impl := port.Impl[Settings]{
		Login: port.TokenLogin[Settings]{DoLogin: doLogin, CheckIfLoginNeeded: checkIfLoginNeeded},
		Basic: g.basic,
		Parse: parserBase{site: s}.parseResponse,
	}
	if e.Capabilities.MovieSearchParams != nil {
		impl.Movie = g.movie
	}
	if e.Capabilities.TvSearchParams != nil {
		impl.TV = g.tv
	}
	if s.ModifyRequest != nil {
		impl.ModifyRequest = func(ix *port.Indexer[Settings], request *port.Request) {
			request.SetHeader("Authorization", "Bearer "+ix.Token())
			s.ModifyRequest(ix, request)
		}
	}
	return port.Define(e, impl)
}

// doLogin is DoLogin and GetToken. The framework holds the token as the
// session and sends it on every request to the Indexer's hosts, which is
// ModifyRequest.
func doLogin(in *port.SignIn[Settings]) (string, error) {
	authLoginRequest := port.Post(in.BaseURL.JoinPath(loginURL)).
		AddFormParameter("username", in.Settings.Username).
		AddFormParameter("password", string(in.Settings.Password)).
		AddFormParameter("pid", strings.TrimSpace(in.Settings.PID)).
		Accept("application/json")

	response, err := in.Do(authLoginRequest)
	if err != nil {
		return "", err
	}

	switch {
	case response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusUnprocessableEntity:
		var jsonResponse AvistazErrorResponse
		_ = json.Unmarshal(response.Body, &jsonResponse)
		// SlipStream: the outcome's sentence already says the sign-in was
		// rejected, so there is no upstream fallback message.
		return "", port.IndexerAuthError(jsonResponse.Message)
	case response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices:
		return "", port.IndexerError("the Indexer responded to the sign-in with status %d", response.StatusCode)
	}

	var authResponse AvistazAuthResponse
	if err := json.Unmarshal(response.Body, &authResponse); err != nil {
		return "", port.IndexerError("the Indexer's sign-in response isn't JSON")
	}

	return authResponse.Token, nil
}

func checkIfLoginNeeded(httpResponse *port.IndexerResponse) bool {
	return httpResponse.StatusCode == http.StatusUnauthorized || httpResponse.StatusCode == http.StatusPreconditionFailed
}
