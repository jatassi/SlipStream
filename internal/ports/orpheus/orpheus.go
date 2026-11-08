// Ported from Prowlarr's src/NzbDrone.Core/Indexers/Definitions/Orpheus.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

// Package orpheus is Orpheus, a Gazelle site with an API key. It reads
// the Gazelle family's API types and Download, as upstream does.
package orpheus

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/newznab"
	"github.com/jatassi/slipstream/internal/ports/internal/gazelle"
	"github.com/jatassi/slipstream/internal/ports/port"
	"github.com/jatassi/slipstream/internal/prowlarr/datetimeutil"
	"github.com/jatassi/slipstream/internal/prowlarr/encoding"
	"github.com/jatassi/slipstream/internal/prowlarr/stringutil"
)

//go:embed upstream.json
var upstream []byte

type ix = port.Indexer[orpheusSettings]

var Port = port.Define(port.Entry{
	ID:           "orpheus",
	Name:         "Orpheus",
	IndexerURLs:  []string{"https://orpheus.network/"},
	Description:  "Orpheus, once called Apollo, is a private Tracker for music.",
	Privacy:      port.Private,
	RateLimit:    3 * time.Second,
	PageSize:     50,
	Capabilities: setCapabilities(),
	Upstream:     port.ReadUpstream(upstream),
}, port.Impl[orpheusSettings]{
	Login:          port.NoLogin{},
	Basic:          getSearchRequests,
	Parse:          parseResponse,
	ModifyRequest:  getDownloadRequest,
	Download:       download,
	FilterReleases: cleanupReleases,
})

// SlipStream: ModifyRequest runs on every request to the Indexer's hosts,
// so the generator's searches get the header here too.
func getDownloadRequest(ix *ix, request *port.Request) {
	request.SetHeader("Authorization", "token "+ix.Settings.APIKey)
}

func cleanupReleases(_ *ix, searchCriteria port.SearchCriteria, cleanReleases []port.TorrentInfo) []port.TorrentInfo {
	if searchCriteria.IsRssSearch() {
		cleanReleases = cleanReleases[:min(50, len(cleanReleases))]
	}

	return cleanReleases
}

// SlipStream: no music or book search, which the framework doesn't run,
// so Basic carries the term.
func setCapabilities() port.Capabilities {
	return port.Capabilities{
		Categories: port.Categories(
			port.Map(1, newznab.Audio, "Music"),
			port.Map(2, newznab.PC, "Applications"),
			port.Map(3, newznab.BooksEBook, "E-Books"),
			port.Map(4, newznab.AudioAudiobook, "Audiobooks"),
			port.Map(5, newznab.Other, "E-Learning Videos"),
			port.Map(6, newznab.Other, "Comedy"),
			port.Map(7, newznab.BooksComics, "Comics"),
		),
	}
}

func download(ix *ix, link *url.URL) (port.Got, error) {
	return gazelle.Download(ix, link, ix.Settings.UseFreeleechToken)
}

func getSearchRequests(ix *ix, searchCriteria port.SearchCriteria) (port.Chain, error) {
	var parameters stringutil.NameValueCollection

	return getPagedRequests(ix, searchCriteria, parameters)
}

func getPagedRequests(ix *ix, searchCriteria port.SearchCriteria, parameters stringutil.NameValueCollection) (port.Chain, error) {
	term := strings.TrimSpace(searchCriteria.SanitizedSearchTerm())

	parameters.Set("action", "browse")
	parameters.Set("order_by", "time")
	parameters.Set("order_way", "desc")

	if strings.TrimSpace(term) != "" {
		parameters.Set("searchstr", term)
	}

	queryCats := ix.Categories.TrackerIDs(searchCriteria.Categories)

	for _, cat := range queryCats {
		parameters.Set("filter_cat["+cat+"]", "1")
	}

	request := port.Get(ix.URL("ajax.php?" + parameters.GetQueryString())).
		Accept("application/json")

	return port.One(request)
}

// SlipStream: the messages are SlipStream's own, per WRITING.md, with
// the site's words quoted.
func parseResponse(ix *ix, indexerResponse *port.IndexerResponse) ([]port.TorrentInfo, error) {
	var torrentInfos []port.TorrentInfo

	if indexerResponse.StatusCode != http.StatusOK {
		var errorResponse gazelle.GazelleErrorResponse
		_ = json.Unmarshal(indexerResponse.Body, &errorResponse)

		// SlipStream: a 401 is a refused API key, which upstream reads as
		// any other status.
		if indexerResponse.StatusCode == http.StatusUnauthorized {
			return nil, port.IndexerAuthError(string(errorResponse.Error))
		}
		if errorResponse.Error != "" {
			return nil, port.IndexerError("the Indexer responded with status %d and the error %q", indexerResponse.StatusCode, errorResponse.Error)
		}
		return nil, port.IndexerError("the Indexer responded with status %d", indexerResponse.StatusCode)
	}

	if contentType := indexerResponse.Headers.Get("Content-Type"); !strings.Contains(contentType, "application/json") {
		return nil, port.IndexerError("the Indexer responded with %q where application/json was expected", contentType)
	}

	var jsonResponse gazelle.GazelleResponse
	if err := json.Unmarshal(indexerResponse.Body, &jsonResponse); err != nil {
		return nil, port.IndexerError("read the Indexer's search response: %v", err)
	}
	if jsonResponse.Status != "success" ||
		strings.TrimSpace(string(jsonResponse.Status)) == "" ||
		jsonResponse.Response == nil {
		return torrentInfos, nil
	}

	for _, result := range jsonResponse.Response.Results {
		if result.Torrents != nil {
			for _, torrent := range result.Torrents {
				// skip releases that cannot be used with freeleech tokens when the option is enabled
				if ix.Settings.UseFreeleechToken == gazelle.Required && !torrent.CanUseToken {
					continue
				}

				id := torrent.TorrentID

				title := getTitle(result, torrent)
				infoURL := getInfoURL(ix, string(result.GroupID), id)
				isFreeLeech := torrent.IsFreeLeech || torrent.IsNeutralLeech || torrent.IsPersonalFreeLeech

				size, err := strconv.ParseInt(string(torrent.Size), 10, 64)
				if err != nil {
					return nil, fmt.Errorf("read a release's size: %w", err)
				}
				seeders, err := strconv.Atoi(string(torrent.Seeders))
				if err != nil {
					return nil, fmt.Errorf("read a release's seeders: %w", err)
				}
				leechers, err := strconv.Atoi(string(torrent.Leechers))
				if err != nil {
					return nil, fmt.Errorf("read a release's leechers: %w", err)
				}
				publishDate, err := datetimeutil.Parse(string(torrent.Time), datetimeutil.AssumeUniversal|datetimeutil.AdjustToUniversal, ix.Clock)
				if err != nil {
					return nil, fmt.Errorf("read a release's date: %w", err)
				}

				// SlipStream: no Artist, Album, Year, Container or Codec, which
				// a Release doesn't carry. IndexerBase adds Scene from
				// TorrentInfo.Scene, which the Go TorrentInfo carries as a flag.
				release := port.TorrentInfo{
					GUID:                 infoURL,
					InfoURL:              infoURL,
					DownloadURL:          getDownloadURL(ix, id, torrent.CanUseToken && !isFreeLeech),
					Title:                encoding.HTMLDecode(title),
					Size:                 size,
					Seeders:              seeders,
					Peers:                leechers + seeders,
					PublishDate:          publishDate,
					Files:                torrent.FileCount,
					Grabs:                torrent.Snatches,
					DownloadVolumeFactor: factor(isFreeLeech),
					UploadVolumeFactor:   factor(torrent.IsNeutralLeech),
				}
				if torrent.Scene {
					release.IndexerFlags = []port.SiteFlag{port.Scene}
				}

				// SlipStream: a category the response leaves out reads as "".
				category := string(torrent.Category)
				if category == "" || strings.Contains(category, "Select Category") {
					release.Categories = ix.Categories.FromTracker("1")
				} else {
					release.Categories = ix.Categories.FromDesc(category)
				}

				torrentInfos = append(torrentInfos, release)
			}
		} else {
			// Non-Audio files are formatted a little differently (1:1 for group and torrents)

			// skip releases that cannot be used with freeleech tokens when the option is enabled
			if ix.Settings.UseFreeleechToken == gazelle.Required && !result.CanUseToken {
				continue
			}

			id := result.TorrentID
			infoURL := getInfoURL(ix, string(result.GroupID), id)
			isFreeLeech := result.IsFreeLeech || result.IsNeutralLeech || result.IsPersonalFreeLeech

			size, err := strconv.ParseInt(string(result.Size), 10, 64)
			if err != nil {
				return nil, fmt.Errorf("read a release's size: %w", err)
			}
			seeders, err := strconv.Atoi(string(result.Seeders))
			if err != nil {
				return nil, fmt.Errorf("read a release's seeders: %w", err)
			}
			leechers, err := strconv.Atoi(string(result.Leechers))
			if err != nil {
				return nil, fmt.Errorf("read a release's leechers: %w", err)
			}
			var publishDate time.Time
			if num, err := strconv.ParseInt(string(result.GroupTime), 10, 64); err == nil {
				publishDate = time.Unix(num, 0).UTC()
			} else if publishDate, err = datetimeutil.FromFuzzyTime(string(result.GroupTime), ix.Clock); err != nil {
				return nil, fmt.Errorf("read a release's date: %w", err)
			}

			release := port.TorrentInfo{
				GUID:                 infoURL,
				InfoURL:              infoURL,
				DownloadURL:          getDownloadURL(ix, id, result.CanUseToken && !isFreeLeech),
				Title:                encoding.HTMLDecode(string(result.GroupName)),
				Size:                 size,
				Seeders:              seeders,
				Peers:                leechers + seeders,
				PublishDate:          publishDate,
				Files:                result.FileCount,
				Grabs:                result.Snatches,
				DownloadVolumeFactor: factor(isFreeLeech),
				UploadVolumeFactor:   factor(result.IsNeutralLeech),
			}

			category := string(result.Category)
			if category == "" || strings.Contains(category, "Select Category") {
				release.Categories = ix.Categories.FromTracker("1")
			} else {
				release.Categories = ix.Categories.FromDesc(category)
			}

			torrentInfos = append(torrentInfos, release)
		}
	}

	// order by date
	slices.SortStableFunc(torrentInfos, func(a, b port.TorrentInfo) int { return b.PublishDate.Compare(a.PublishDate) })
	return torrentInfos, nil
}

// factor is C#'s free ? 0 : 1.
func factor(free bool) float64 {
	if free {
		return 0
	}
	return 1
}

func getTitle(result gazelle.GazelleRelease, torrent gazelle.GazelleTorrent) string {
	title := fmt.Sprintf("%s - %s (%s)", result.Artist, result.GroupName, result.GroupYear)

	if strings.TrimSpace(string(result.ReleaseType)) != "" && result.ReleaseType != "Unknown" {
		title += " [" + string(result.ReleaseType) + "]"
	}

	if strings.TrimSpace(string(torrent.RemasterTitle)) != "" {
		title += " [" + strings.TrimSpace(fmt.Sprintf("%s %s", torrent.RemasterTitle, torrent.RemasterYear)) + "]"
	}

	flags := []string{
		fmt.Sprintf("%s %s", torrent.Format, torrent.Encoding),
		string(torrent.Media),
	}

	if torrent.HasLog {
		flags = append(flags, "Log ("+strconv.Itoa(torrent.LogScore)+"%)")
	}

	if torrent.HasCue {
		flags = append(flags, "Cue")
	}

	return fmt.Sprintf("%s [%s]", title, strings.Join(flags, " / "))
}

func getDownloadURL(ix *ix, torrentID int, canUseToken bool) string {
	uri := port.NewURI(ix.BaseURL).
		CombinePath("/ajax.php").
		AddQueryParam("action", "download").
		AddQueryParam("id", strconv.Itoa(torrentID))

	// Orpheus fails to download if usetoken=0 so we need to only add if we will use one
	if t := ix.Settings.UseFreeleechToken; (t == gazelle.Preferred || t == gazelle.Required) && canUseToken {
		uri = uri.AddQueryParam("usetoken", "1")
	}

	return uri.FullURI()
}

func getInfoURL(ix *ix, groupID string, torrentID int) string {
	uri := port.NewURI(ix.BaseURL).
		CombinePath("/torrents.php").
		AddQueryParam("id", groupID).
		AddQueryParam("torrentid", strconv.Itoa(torrentID))

	return uri.FullURI()
}

// SlipStream: Prowlarr's validator is the required flag on the API key.
// OrpheusFreeleechTokenAction is the Gazelle family's enum, whose values
// and options are the same.
type orpheusSettings struct {
	port.APIKeyField  `help:"Find your API key on the site under Settings > Access Settings."`
	UseFreeleechToken gazelle.GazelleFreeleechTokenAction `port:"useFreeleechToken" label:"Use freeleech tokens" default:"0" help:"With Always, SlipStream lists only Releases that can take a token."`
}
