// Ported from Prowlarr's src/NzbDrone.Core/Indexers/Definitions/Libble.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

package libble

import (
	_ "embed"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/jatassi/slipstream/internal/newznab"
	"github.com/jatassi/slipstream/internal/ports/port"
	"github.com/jatassi/slipstream/internal/prowlarr/datetimeutil"
	"github.com/jatassi/slipstream/internal/prowlarr/dom"
	"github.com/jatassi/slipstream/internal/prowlarr/parseutil"
	"github.com/jatassi/slipstream/internal/prowlarr/stringutil"
)

//go:embed upstream.json
var upstream []byte

type ix = port.Indexer[LibbleSettings]

var Port = port.Define(port.Entry{
	ID:          "libble",
	Name:        "Libble",
	IndexerURLs: []string{"https://libble.me/"},
	// SlipStream: SlipStream's own sentence, per WRITING.md.
	Description:  "Libble is a private Tracker for music.",
	Language:     "en-US",
	Privacy:      port.Private,
	PageSize:     50,
	Capabilities: SetCapabilities(),
	Upstream:     port.ReadUpstream(upstream),
}, port.Impl[LibbleSettings]{
	Login: port.CodeLogin[LibbleSettings]{DoLogin: DoLogin, CheckIfLoginNeeded: CheckIfLoginNeeded},
	Basic: func(ix *ix, searchCriteria port.SearchCriteria) (port.Chain, error) {
		var parameters stringutil.NameValueCollection

		return port.One(GetPagedRequests(ix, searchCriteria, &parameters))
	},
	Parse: ParseResponse,
})

func DoLogin(in *port.SignIn[LibbleSettings]) error {
	loginURL := in.BaseURL.JoinPath("login.php")

	// SlipStream: the code is the authenticator key's or the one typed for a
	// Test, never a stored one. Upstream's TwoFactorAuthCode isn't ported.
	code, hasCode := in.TwoFactorCode()

	// SlipStream: a sign-in's requests carry only the cookies it earns, so
	// Cookies = null needs no line, and the framework keeps the session it
	// earns in place of UpdateCookies.
	response, err := in.Do(port.Post(loginURL).
		AddFormParameter("username", in.Settings.Username).
		AddFormParameter("password", string(in.Settings.Password)).
		AddFormParameter("code", code).
		AddFormParameter("keeplogged", "1").
		AddFormParameter("login", "Login").
		SetHeader("Content-Type", "application/x-www-form-urlencoded").
		SetHeader("Referer", loginURL.String()))
	if err != nil {
		return err
	}

	if CheckIfLoginNeeded(response) {
		// SlipStream: no fallback message, since the stop sentence names
		// the sign-in fields and Prowlarr's fallback isn't the site's words.
		errorMessage := strings.TrimSpace(dom.Parse(response.Content).QuerySelector("#loginform > .warning").TextContent())

		// SlipStream: an account with two-factor sign-in that signed in
		// with no code stops for one.
		if !hasCode && codeWords.MatchString(errorMessage) {
			return port.ErrTwoFactorCodeNeeded
		}

		return port.IndexerAuthError(errorMessage)
	}

	return nil
}

// SlipStream: no Libble page was recorded, so the words its warning uses
// to ask for the two-factor code are a guess. A capture of a refused
// sign-in replaces it.
var codeWords = regexp.MustCompile(`(?i)\b2fa\b|two-factor|two factor`)

func CheckIfLoginNeeded(httpResponse *port.IndexerResponse) bool {
	return !strings.Contains(httpResponse.Content, "logout.php")
}

func SetCapabilities() port.Capabilities {
	// SlipStream: MusicSearchParams have no mode in SlipStream, so music
	// searches go through Basic, and the music search generator isn't
	// ported.
	return port.Capabilities{
		Categories: port.Categories(
			port.Map(1, newznab.Audio, "Music"),
			port.Map(2, newznab.Audio, "Libble Mixtapes"),
			port.Map(7, newznab.AudioVideo, "Music Videos"),
		),
	}
}

func GetPagedRequests(ix *ix, searchCriteria port.SearchCriteria, parameters *stringutil.NameValueCollection) *port.Request {
	term := strings.TrimSpace(searchCriteria.SanitizedSearchTerm())

	parameters.Set("action", "advanced")
	parameters.Set("order_by", "time")
	parameters.Set("order_way", "desc")

	if term != "" {
		parameters.Set("searchstr", term)
	}

	for _, cat := range ix.Categories.TrackerIDs(searchCriteria.Categories) {
		parameters.Set("filter_cat["+cat+"]", "1")
	}

	if searchCriteria.Limit > 0 && searchCriteria.Offset > 0 {
		page := searchCriteria.Offset/searchCriteria.Limit + 1
		parameters.Set("page", strconv.Itoa(page))
	}

	searchURL := ix.URL("torrents.php")
	searchURL.RawQuery = parameters.GetQueryString()

	return port.Get(searchURL).Accept("text/html")
}

var ReleaseYearRegex = regexp.MustCompile(`\[(\d{4})\]$`)

func ParseResponse(ix *ix, indexerResponse *port.IndexerResponse) ([]port.TorrentInfo, error) {
	var releaseInfos []port.TorrentInfo

	doc := dom.Parse(indexerResponse.Content)

	groups := doc.QuerySelectorAll(`table#torrent_table > tbody > tr.group:has(strong > a[href*="torrents.php?id="])`)
	for _, group := range groups {
		albumLinkNode := group.QuerySelector(`strong > a[href*="torrents.php?id="]`)
		groupID, err := parseutil.GetArgumentFromQueryString(albumLinkNode.GetAttribute("href"), "id")
		if err != nil {
			return nil, err
		}

		artistsNodes := group.QuerySelectorAll(`strong > a[href*="artist.php?id="]`)

		releaseArtist := "Various Artists"
		if len(artistsNodes) > 0 {
			artists := make([]string, 0, len(artistsNodes))
			for _, artist := range artistsNodes {
				artists = append(artists, strings.TrimSpace(artist.TextContent()))
			}
			releaseArtist = strings.Join(artists, ", ")
		}

		releaseAlbumName := strings.TrimSpace(group.QuerySelector(`strong > a[href*="torrents.php?id="]`).TextContent())

		title := strings.TrimSpace(group.QuerySelector("td:nth-child(4) > strong").TextContent())
		releaseAlbumYear := ReleaseYearRegex.FindString(title)

		// SlipStream: a Release has no description, genres, artist, album
		// or poster, so div.tags and .thumbnail aren't read.

		rows := doc.QuerySelectorAll(fmt.Sprintf(`table#torrent_table > tbody > tr.group_torrent.groupid_%s:has(a[href*="torrents.php?id="])`, groupID))
		for _, row := range rows {
			detailsNode := row.QuerySelector(`a[href^="torrents.php?id="]`)

			infoURL := ix.BaseURL.String() + strings.TrimSpace(detailsNode.GetAttribute("href"))
			downloadLink := ix.BaseURL.String() + strings.TrimSpace(row.QuerySelector(`a[href^="torrents.php?action=download&id="]`).GetAttribute("href"))

			releaseTags := strings.Trim(detailsNode.FirstChild().TextContent(), " /")
			seeders, err := parseutil.CoerceInt(row.QuerySelector("td:nth-child(6)").TextContent())
			if err != nil {
				return nil, err
			}
			size, err := parseutil.GetBytes(strings.TrimSpace(row.QuerySelector("td:nth-child(4)").TextContent()))
			if err != nil {
				return nil, err
			}
			files, err := parseutil.CoerceInt(row.QuerySelector("td:nth-child(2)").TextContent())
			if err != nil {
				return nil, err
			}
			grabs, err := parseutil.CoerceInt(row.QuerySelector("td:nth-child(5)").TextContent())
			if err != nil {
				return nil, err
			}
			leechers, err := parseutil.CoerceInt(row.QuerySelector("td:nth-child(7)").TextContent())
			if err != nil {
				return nil, err
			}

			release := port.TorrentInfo{
				GUID:                 infoURL,
				InfoURL:              infoURL,
				DownloadURL:          downloadLink,
				Title:                strings.Trim(fmt.Sprintf("%s - %s %s %s", releaseArtist, releaseAlbumName, releaseAlbumYear, releaseTags), " -"),
				Categories:           ParseCategories(group),
				Size:                 size,
				Files:                files,
				Grabs:                grabs,
				Seeders:              seeders,
				Peers:                seeders + leechers,
				DownloadVolumeFactor: 1,
				UploadVolumeFactor:   1,
				MinimumRatio:         1,
				MinimumSeedTime:      259200, // 72 hours,
			}

			dateAdded := strings.TrimSpace(row.QuerySelector("td:nth-child(3) > span[title]").GetAttribute("title"))
			publishDate, ok := datetimeutil.TryParseExact(dateAdded, "MMM dd yyyy, HH:mm", datetimeutil.AssumeUniversal, ix.Clock)
			if !ok {
				if publishDate, err = datetimeutil.FromTimeAgo(strings.TrimSpace(row.QuerySelector("td:nth-child(3)").TextContent()), ix.Clock); err != nil {
					return nil, err
				}
			}
			release.PublishDate = publishDate

			switch strings.Trim(strings.ToLower(row.QuerySelector(`a[href^="torrents.php?id="] strong`).TextContent()), " !") {
			case "neutral":
				release.DownloadVolumeFactor = 0
				release.UploadVolumeFactor = 0
			case "freeleech":
				release.DownloadVolumeFactor = 0
				release.UploadVolumeFactor = 1
			}

			releaseInfos = append(releaseInfos, release)
		}
	}

	return releaseInfos, nil
}

var categoryClass = regexp.MustCompile(`\bcats_(.*?)\b`)

func ParseCategories(group *dom.Node) []newznab.Category {
	cat := strings.TrimSpace(group.QuerySelector("td.cats_col div.cat_icon").GetAttribute("class"))

	if matchCategory := categoryClass.FindStringSubmatch(cat); matchCategory != nil {
		cat = strings.TrimSpace(matchCategory[1])
	}

	switch cat {
	case "music", "libblemixtapes":
		return []newznab.Category{newznab.Audio}
	case "musicvideos":
		return []newznab.Category{newznab.AudioVideo}
	default:
		return []newznab.Category{newznab.Other}
	}
}

type LibbleSettings struct {
	port.UserPass
	// SlipStream: TwoFactorAuthCode isn't ported, because a code stored in
	// the settings fails at the next sign-in. CodeLogin adds the
	// authenticator key.
}
