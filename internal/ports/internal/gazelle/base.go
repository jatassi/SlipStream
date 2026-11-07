// Ported from Prowlarr's src/NzbDrone.Core/Indexers/Definitions/Gazelle/GazelleBase.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

// Package gazelle is GazelleBase, the family of AlphaRatio,
// GreatPosterWall, SecretCinema, BrokenStones, DICMusic and DesiGaane: a
// login.php sign-in to a cookie session, the ajax.php browse API, a JSON
// parser with categories by description, and the freeleech-token
// setting. A site's package declares its Entry, its settings (which
// embed Settings or UserPassOrCookieSettings) and the members its C#
// classes override, and Define does the rest. Orpheus and Redacted,
// which sign in with an API key instead, read the API types and
// Download.
package gazelle

import (
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/jatassi/slipstream/internal/ports/port"
	"github.com/jatassi/slipstream/internal/prowlarr/stringutil"
)

const (
	PageSize = 50
	loginURL = "login.php"
)

// Site is what a site's C# classes override, its request generator's and
// parser's included. A nil member is the base's.
type Site[S settings] struct {
	// ImdbInTags is the request generator's: AlphaRatio sends the IMDb id
	// as taglist in place of cataloguenumber.
	ImdbInTags bool
	// GetMovieSearchRequests is the request generator's movie
	// GetSearchRequests, the one overload C# makes virtual:
	// GreatPosterWall's, which calls GetRequest.
	GetMovieSearchRequests func(ix *port.Indexer[S], searchCriteria port.SearchCriteria) (port.Chain, error)
	// GetBasicSearchParameters is AlphaRatio's and GreatPosterWall's.
	// Each calls the package's GetBasicSearchParameters first, as C#'s
	// base.GetBasicSearchParameters.
	GetBasicSearchParameters func(ix *port.Indexer[S], searchCriteria port.SearchCriteria, term string) stringutil.NameValueCollection
	// ParseResponse is GreatPosterWall's and SecretCinema's parser, in
	// place of the family's.
	ParseResponse func(ix *port.Indexer[S], indexerResponse *port.IndexerResponse) ([]port.TorrentInfo, error)
	// GetDownloadURL is the parser's GetDownloadUrl: AlphaRatio's and
	// GreatPosterWall's.
	GetDownloadURL func(ix *port.Indexer[S], torrentID int, canUseToken bool) string
	// CleanupReleases is GreatPosterWall's, which keeps the first 50 of
	// an RSS search.
	CleanupReleases func(ix *port.Indexer[S], searchCriteria port.SearchCriteria, releases []port.TorrentInfo) []port.TorrentInfo
}

// Define is the site's Port: e with the base's PageSize when it sets
// none, and the modes its Capabilities declare.
func Define[S settings](e port.Entry, s Site[S]) port.Port {
	if e.PageSize == 0 {
		e.PageSize = PageSize
	}
	if s.GetBasicSearchParameters == nil {
		s.GetBasicSearchParameters = GetBasicSearchParameters[S]
	}
	if s.GetDownloadURL == nil {
		s.GetDownloadURL = GetDownloadURL[S]
	}
	if s.ParseResponse == nil {
		s.ParseResponse = parser[S]{site: s}.parseResponse
	}
	g := requestGenerator[S]{site: s}
	impl := port.Impl[S]{
		Login:          port.FormLogin[S]{DoLogin: doLogin[S], CheckIfLoginNeeded: checkIfLoginNeeded},
		Basic:          g.basic,
		Parse:          s.ParseResponse,
		Download:       download[S],
		FilterReleases: s.CleanupReleases,
	}
	if e.Capabilities.MovieSearchParams != nil {
		impl.Movie = g.movie
		if s.GetMovieSearchRequests != nil {
			impl.Movie = s.GetMovieSearchRequests
		}
	}
	if e.Capabilities.TvSearchParams != nil {
		impl.TV = g.tv
	}
	return port.Define(e, impl)
}

// doLogin is DoLogin. The framework keeps the cookies the sign-in earns
// as the session.
func doLogin[S settings](in *port.SignIn[S]) error {
	// SlipStream: GetCookies' cookie is the session in place of the form.
	if cookie := in.Settings.cookie(); strings.TrimSpace(cookie) != "" {
		return in.UseCookie(cookie)
	}

	authLoginRequestBuilder := authLoginRequestBuilder(in)

	response, err := in.Do(authLoginRequestBuilder)
	if err != nil {
		return err
	}

	return checkForLoginError(response)
}

// SlipStream: HttpRequestBuilder doesn't follow a redirect, so the
// sign-in's own response sets the session cookie. The form body sets
// Content-Type.
func authLoginRequestBuilder[S settings](in *port.SignIn[S]) *port.Request {
	settings := in.Settings.gazelle()
	loginURL := in.BaseURL.JoinPath(loginURL)

	return port.Post(loginURL).
		AddFormParameter("username", settings.Username).
		AddFormParameter("password", string(settings.Password)).
		AddFormParameter("keeplogged", "1").
		SetHeader("Referer", loginURL.String()).
		Accept("application/json").
		NoRedirect()
}

// SlipStream: upstream accepts every sign-in response. Gazelle answers a
// sign-in it accepts with a redirect and shows its form again with a 200
// for a wrong username or password, so that 200 is the rejection.
func checkForLoginError(response *port.IndexerResponse) error {
	switch {
	case response.HasHTTPRedirect:
		return nil
	case response.StatusCode == http.StatusOK:
		return port.IndexerAuthError("")
	}
	return port.IndexerError("the Indexer responded to the sign-in with status %d", response.StatusCode)
}

func download[S settings](ix *port.Indexer[S], link *url.URL) (port.Got, error) {
	return Download(ix, link, ix.Settings.gazelle().UseFreeleechToken)
}

// Download is GazelleBase.Download: a download without the token again when Preferred
// asked for a token the site won't take. Orpheus and Redacted keep the
// same.
func Download[S any](ix *port.Indexer[S], link *url.URL, useFreeleechToken GazelleFreeleechTokenAction) (port.Got, error) {
	downloadResponse, err := baseDownload(ix, link)
	if err != nil {
		return nil, err
	}

	fileData := downloadResponse.Body

	if useFreeleechToken == Preferred &&
		len(fileData) >= 1 &&
		fileData[0] != 'd' && // simple test for torrent vs HTML content
		strings.Contains(link.RawQuery, "usetoken=1") {
		html := downloadResponse.Content

		if strings.Contains(html, "You do not have any freeleech tokens left.") ||
			strings.Contains(html, "You do not have enough freeleech tokens") ||
			strings.Contains(html, "This torrent is too large.") ||
			strings.Contains(html, "You cannot use tokens here") {
			// Try to download again without usetoken
			withoutToken, err := url.Parse(port.NewURI(link).RemoveQueryParam("usetoken").FullURI())
			if err != nil {
				return nil, fmt.Errorf("read the download link without its token: %w", err)
			}
			if downloadResponse, err = baseDownload(ix, withoutToken); err != nil {
				return nil, err
			}
		}
	}

	return port.Torrent{Raw: downloadResponse.Body}, nil
}

// baseDownload is base.Download, the framework's fetch of a link.
// SlipStream: the kit has no call for the default fetch, so this sends
// the GET and reads its status as the default does.
func baseDownload[S any](ix *port.Indexer[S], link *url.URL) (*port.IndexerResponse, error) {
	response, err := ix.Fetch(port.Get(link))
	switch {
	case err != nil:
		return nil, err
	case response.StatusCode == http.StatusNotFound:
		return nil, port.ErrReleaseUnavailable
	case response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices:
		return nil, port.IndexerError("the Indexer responded to the download with status %d", response.StatusCode)
	}
	return response, nil
}

// SlipStream: CheckIfLoginNeeded reads no settings, so it can't skip
// the check when a cookie is set. A sign-in then reuses the cookie.
func checkIfLoginNeeded(response *port.IndexerResponse) bool {
	invalidResponses := []string{`"bad credentials"`, `"groupName":"wrong-creds"`}

	return response.HasHTTPRedirect || slices.ContainsFunc(invalidResponses, func(invalid string) bool { return strings.Contains(response.Content, invalid) })
}
