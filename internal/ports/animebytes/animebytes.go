// Ported from Prowlarr's src/NzbDrone.Core/Indexers/Definitions/AnimeBytes.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

package animebytes

import (
	"crypto/md5" //nolint:gosec // HashUtil.CalculateMd5 names a Release, it guards nothing
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/jatassi/slipstream/internal/newznab"
	"github.com/jatassi/slipstream/internal/ports/port"
	"github.com/jatassi/slipstream/internal/prowlarr/datetimeutil"
	"github.com/jatassi/slipstream/internal/prowlarr/parseutil"
	"github.com/jatassi/slipstream/internal/prowlarr/serializer"
	"github.com/jatassi/slipstream/internal/prowlarr/stringutil"
)

//go:embed upstream.json
var upstream []byte

// Port is AnimeBytes.
//
// SlipStream: the Music and Book search params are gone, because SlipStream
// has neither mode. FetchPage's three-minute cache of each page is gone:
// SlipStream's own search cache sits in front of every Indexer.
// CheckIfLoginNeeded's false is NoLogin's own behavior.
var Port = port.Define(port.Entry{
	ID:          "animebytes",
	Name:        "AnimeBytes",
	Description: "AnimeBytes (AB) is the largest private torrent tracker that specialises in anime and anime-related content.",
	Language:    "en-US",
	Privacy:     port.Private,
	IndexerURLs: []string{"https://animebytes.tv/"},
	RateLimit:   4 * time.Second,
	Capabilities: port.Capabilities{
		TvSearchParams:    []port.Param{port.Q, port.Season, port.Ep},
		MovieSearchParams: []port.Param{port.Q},
		Categories: port.Categories(
			port.Map("anime[tv_series]", newznab.TVAnime, "TV Series"),
			port.Map("anime[tv_special]", newznab.TVAnime, "TV Special"),
			port.Map("anime[ova]", newznab.TVAnime, "OVA"),
			port.Map("anime[ona]", newznab.TVAnime, "ONA"),
			port.Map("anime[dvd_special]", newznab.TVAnime, "DVD Special"),
			port.Map("anime[bd_special]", newznab.TVAnime, "BD Special"),
			port.Map("anime[movie]", newznab.Movies, "Movie"),
			port.Map("audio", newznab.Audio, "Music"),
			port.Map("gamec[game]", newznab.Console, "Game"),
			port.Map("gamec[game]", newznab.PCGames, "Game"),
			port.Map("gamec[visual_novel]", newznab.Console, "Game Visual Novel"),
			port.Map("gamec[visual_novel]", newznab.PCGames, "Game Visual Novel"),
			port.Map("printedtype[manga]", newznab.BooksComics, "Manga"),
			port.Map("printedtype[oneshot]", newznab.BooksComics, "Oneshot"),
			port.Map("printedtype[anthology]", newznab.BooksComics, "Anthology"),
			port.Map("printedtype[manhwa]", newznab.BooksComics, "Manhwa"),
			port.Map("printedtype[light_novel]", newznab.BooksComics, "Light Novel"),
			port.Map("printedtype[artbook]", newznab.BooksComics, "Artbook"),
		),
	},
	Upstream: port.ReadUpstream(upstream),
}, port.Impl[animeBytesSettings]{
	Login:          port.NoLogin{},
	Movie:          getMovieSearchRequests,
	TV:             getTvSearchRequests,
	Basic:          getBasicSearchRequests,
	Parse:          parseResponse,
	FilterReleases: cleanupReleases,
})

type indexer = port.Indexer[animeBytesSettings]

// SlipStream: Prowlarr's base CleanupReleases fills fields SlipStream fills
// itself, and the Clone is Go's copy. DateTime.UtcNow is the job's clock.
func cleanupReleases(ix *indexer, searchCriteria port.SearchCriteria, releases []port.TorrentInfo) []port.TorrentInfo {
	cleanReleases := releases

	if searchCriteria.IsRssSearch() {
		dayAgo := ix.Clock.Now.AddDate(0, 0, -1)
		cleanReleases = nil
		for index, r := range releases {
			if r.PublishDate.After(dayAgo) || index < 20 {
				cleanReleases = append(cleanReleases, r)
			}
		}
	}

	return cleanReleases
}

// wordClass is .NET's \w, which is Unicode, where RE2's is ASCII.
const wordClass = `\p{L}\p{Mn}\p{Nd}\p{Pc}`

var yearRegex = regexp.MustCompile(`(?:^|[^` + wordClass + `])((?:19|20)\p{Nd}{2})$`)

// SlipStream: the Music and Book overloads are gone. SlipStream has
// neither mode.
func getMovieSearchRequests(ix *indexer, searchCriteria port.SearchCriteria) (port.Chain, error) {
	return getRequestWithSearchType(ix, searchCriteria, "anime", false)
}

func getTvSearchRequests(ix *indexer, searchCriteria port.SearchCriteria) (port.Chain, error) {
	return getRequestWithSearchType(ix, searchCriteria, "anime", true)
}

func getBasicSearchRequests(ix *indexer, searchCriteria port.SearchCriteria) (port.Chain, error) {
	return getRequestWithSearchType(ix, searchCriteria, "anime", false)
}

// SlipStream: tv is C#'s "searchCriteria is TvSearchCriteria", since one
// SearchCriteria type serves every mode.
func getRequestWithSearchType(ix *indexer, searchCriteria port.SearchCriteria, searchType string, tv bool) (port.Chain, error) {
	return port.One(getRequest(ix, searchCriteria, searchType, tv))
}

func getRequest(ix *indexer, searchCriteria port.SearchCriteria, searchType string, tv bool) *port.Request {
	searchURL := "scrape.php"

	term := strings.TrimSpace(searchCriteria.SanitizedSearchTerm())
	searchTerm := cleanSearchTerm(term)

	limit := "20"
	if strings.TrimSpace(searchTerm) != "" {
		limit = "50"
	}

	var parameters stringutil.NameValueCollection
	parameters.Add("username", ix.Settings.Username)
	parameters.Add("torrent_pass", ix.Settings.Passkey)
	parameters.Add("sort", "grouptime")
	parameters.Add("way", "desc")
	parameters.Add("type", searchType)
	parameters.Add("searchstr", searchTerm)
	parameters.Add("limit", limit)

	if ix.Settings.SearchByYear && searchType == "anime" {
		if searchYear, ok := parseYearFromSearchTerm(term); ok && searchYear > 0 {
			parameters.Set("year", strconv.Itoa(searchYear))
		}
	}

	queryCats := ix.Categories.TrackerIDs(searchCriteria.Categories)

	if len(queryCats) > 0 && tv && searchCriteria.Season != nil && *searchCriteria.Season > 0 {
		// Avoid searching for specials if it's a non-zero season search
		queryCats = slices.DeleteFunc(queryCats, func(cat string) bool {
			return cat == "anime[tv_special]" || cat == "anime[ova]" || cat == "anime[dvd_special]" || cat == "anime[bd_special]"
		})
	}

	for _, cat := range queryCats {
		parameters.Set(cat, "1")
	}

	if ix.Settings.FreeleechOnly {
		parameters.Set("freeleech", "1")
	}

	if ix.Settings.ExcludeHentai && searchType == "anime" {
		parameters.Set("hentai", "0")
	}

	searchURL += "?" + parameters.GetQueryString()

	return port.Get(ix.URL(searchURL)).Accept("application/json")
}

var (
	episodeSuffix       = regexp.MustCompile(`[^` + wordClass + `](\p{Nd}x)?\p{Nd}?\p{Nd}$`)
	seasonEpisodeSuffix = regexp.MustCompile(`[^` + wordClass + `](S\p{Nd}\p{Nd}?E)?\p{Nd}?\p{Nd}$`)
	numberSuffix        = regexp.MustCompile(`[^` + wordClass + `]\p{Nd}+$`)
	theMovieSuffix      = regexp.MustCompile(`(?i)(?:^|[^` + wordClass + `])(The Movie)$`)
)

func cleanSearchTerm(term string) string {
	// Tracer does not support searching with episode number so strip it if we have one
	term = episodeSuffix.ReplaceAllString(term, "")
	term = seasonEpisodeSuffix.ReplaceAllString(term, "")
	term = numberSuffix.ReplaceAllString(term, "")

	// SlipStream: the match takes the character before "The Movie" for
	// .NET's \b, so only the group goes.
	term = strings.TrimSpace(term)
	if m := theMovieSuffix.FindStringSubmatchIndex(term); m != nil {
		term = term[:m[2]] + term[m[3]:]
	}

	return strings.TrimSpace(term)
}

func parseYearFromSearchTerm(term string) (int, bool) {
	if strings.TrimSpace(term) == "" {
		return 0, false
	}

	yearMatch := yearRegex.FindStringSubmatch(term)

	if yearMatch == nil {
		return 0, false
	}

	year, err := parseutil.CoerceInt(yearMatch[1])
	return year, err == nil
}

var (
	excludedProperties            = []string{"Freeleech"}
	remuxResolutions              = []string{"1080i", "1080p", "2160p", "4K"}
	commonReleaseGroupsProperties = []string{"Softsubs", "Hardsubs", "RAW", "Translated"}
	excludedFileExtensions        = []string{".mka", ".mds", ".md5", ".nfo", ".sfv", ".ass", ".mks", ".srt", ".ssa", ".sup", ".jpeg", ".jpg", ".png", ".otf", ".ttf"}
	propertiesSeparator           = []string{" | ", " / "}
	simpleSeasonRegex             = regexp.MustCompile(`(?:^|[^` + wordClass + `])Season (\p{Nd}+)(?:[^` + wordClass + `]|$)`)
	episodeRegex                  = regexp.MustCompile(`(?:^|[^` + wordClass + `])Episode (\p{Nd}+)(?:[^` + wordClass + `]|$)`)
	h265Regex                     = regexp.MustCompile(`(?i)H\.?265`)
	h264Regex                     = regexp.MustCompile(`(?i)H\.?264`)
	advancedSeasonRegex           = regexp.MustCompile(`(?i)(?:^|[^` + wordClass + `])(?:(\p{Nd}+)(?:st|nd|rd|th) Season|Season (\p{Nd}+))(?:[^` + wordClass + `]|$)`)
	seasonCharactersRegex         = regexp.MustCompile(`(I{2,})$`)
	containsIgnoreCase            = func(s, v string) bool { return strings.Contains(strings.ToLower(s), strings.ToLower(v)) }
	equalsIgnoreCase              = func(set []string, v string) bool {
		return slices.ContainsFunc(set, func(s string) bool { return strings.EqualFold(s, v) })
	}
	startsWithIgnoreCase = func(s, prefix string) bool {
		return len(s) >= len(prefix) && strings.EqualFold(s[:len(prefix)], prefix)
	}
	tvAnime     = []newznab.Category{newznab.TVAnime}
	movies      = []newznab.Category{newznab.Movies}
	booksComics = []newznab.Category{newznab.BooksComics}
)

// SlipStream: the messages are SlipStream's own, per WRITING.md. Prowlarr
// reads no refusal from AnimeBytes; a 401 is a key it refused, which
// SlipStream names account_refused. Year and Description are gone, since a
// SlipStream Release has neither.
func parseResponse(ix *indexer, indexerResponse *port.IndexerResponse) ([]port.TorrentInfo, error) {
	var releaseInfos []port.TorrentInfo

	if indexerResponse.StatusCode == http.StatusUnauthorized {
		return nil, port.IndexerAuthError("")
	}

	if indexerResponse.StatusCode != http.StatusOK {
		return nil, port.IndexerError("the Indexer responded with status %d", indexerResponse.StatusCode)
	}

	if contentType := indexerResponse.Headers.Get("Content-Type"); !strings.Contains(contentType, "application/json") {
		return nil, port.IndexerError("the Indexer responded with %q content, not JSON", contentType)
	}

	var response animeBytesResponse
	if err := json.Unmarshal(indexerResponse.Body, &response); err != nil {
		return nil, port.IndexerError("read the Indexer's search response: %v", err)
	}

	if strings.TrimSpace(response.Error) != "" {
		return nil, port.IndexerError("the Indexer responded with %q", response.Error)
	}

	if response.Matches == 0 {
		return releaseInfos, nil
	}

	for _, group := range response.Groups {
		categoryName := group.CategoryName
		year := group.Year
		groupName := group.GroupName
		seriesName := group.SeriesName
		mainTitle := html.UnescapeString(group.FullName)

		if strings.TrimSpace(seriesName) != "" {
			mainTitle = seriesName
		}

		var synonyms ignoreCaseSet
		synonyms.add(mainTitle)

		if len(group.Synonyms) > 0 {
			if japaneseTitle, ok := group.Synonyms["Japanese"]; ix.Settings.AddJapaneseTitle && ok && strings.TrimSpace(japaneseTitle) != "" {
				synonyms.add(strings.TrimSpace(japaneseTitle))
			}

			if romajiTitle, ok := group.Synonyms["Romaji"]; ix.Settings.AddRomajiTitle && ok && strings.TrimSpace(romajiTitle) != "" {
				synonyms.add(strings.TrimSpace(romajiTitle))
			}

			if alternativeTitle, ok := group.Synonyms["Alternative"]; ix.Settings.AddAlternativeTitle && ok && strings.TrimSpace(alternativeTitle) != "" {
				for _, t := range strings.Split(alternativeTitle, ",") {
					if t = strings.TrimSpace(t); t != "" {
						synonyms.add(t)
					}
				}
			}
		}

		var categories []newznab.Category

		for _, torrent := range group.Torrents {
			// Skip non-freeleech results when freeleech only is set
			if ix.Settings.FreeleechOnly && torrent.RawDownMultiplier != 0 {
				continue
			}

			torrentID := strconv.FormatInt(int64(torrent.ID), 10)
			link := torrent.Link
			publishDate, err := datetimeutil.ParseExact(torrent.UploadTime, "yyyy-MM-dd HH:mm:ss", datetimeutil.AssumeUniversal, ix.Clock)
			if err != nil {
				return nil, port.IndexerError("read the Indexer's search response: %v", err)
			}
			details := ix.BaseURL.String() + "torrent/" + torrentID + "/group"
			size := int64(torrent.Size)
			snatched := int(torrent.Snatched)
			seeders := int(torrent.Seeders)
			leechers := int(torrent.Leechers)
			fileCount := int(torrent.FileCount)
			peers := seeders + leechers
			rawDownMultiplier := float64(torrent.RawDownMultiplier)
			rawUpMultiplier := float64(torrent.RawUpMultiplier)

			// MST with additional 5 hours per GB
			minimumSeedTime := int64(259200 + int32(size/(1<<30)*18000)) //nolint:gosec // C#'s (int) cast, which wraps

			propertyList := splitProperties(html.UnescapeString(torrent.Property))

			propertyList = slices.DeleteFunc(propertyList, func(p string) bool {
				return slices.ContainsFunc(excludedProperties, func(e string) bool { return containsIgnoreCase(p, e) })
			})
			properties := toHashSet(propertyList)

			if slices.ContainsFunc(properties, func(p string) bool { return strings.HasPrefix(p, "M2TS") }) {
				properties = toHashSet(append(properties, "BR-DISK"))
			}

			isBluRayDisk := slices.ContainsFunc(properties, func(p string) bool {
				return p == "RAW" || strings.HasPrefix(p, "M2TS") || strings.HasPrefix(p, "ISO")
			})

			if ix.Settings.ExcludeRaw && isBluRayDisk {
				continue
			}

			hasRemuxFile := slices.ContainsFunc(torrent.Files, func(f animeBytesFile) bool { return containsIgnoreCase(f.FileName, "Remux") })
			mapped := make([]string, 0, len(properties))
			for _, property := range properties {
				if isBluRayDisk {
					property = replaceWord(h265Regex, property, "HEVC")
					property = replaceWord(h264Regex, property, "AVC")
				}

				if hasRemuxFile && equalsIgnoreCase(remuxResolutions, property) {
					property += " Remux"
				}

				mapped = append(mapped, property)
			}
			properties = toHashSet(mapped)

			var season, episode *int

			releaseInfo := ""
			if ix.Settings.EnableSonarrCompatibility && categoryName == "Anime" {
				releaseInfo = "S01"
			}
			editionTitle := ""
			if torrent.EditionData != nil {
				editionTitle = torrent.EditionData.EditionTitle
			}

			if strings.TrimSpace(editionTitle) != "" {
				releaseInfo = html.UnescapeString(editionTitle)

				if ix.Settings.EnableSonarrCompatibility {
					if m := simpleSeasonRegex.FindStringSubmatch(releaseInfo); m != nil {
						season = coerceInt(m[1])
					}
				}

				if m := episodeRegex.FindStringSubmatch(releaseInfo); m != nil {
					episode = coerceInt(m[1])
				}
			}

			if ix.Settings.EnableSonarrCompatibility && categoryName == "Anime" && season == nil {
				season = parseSeasonFromTitles(synonyms)
			}

			switch {
			case episode != nil && *episode > 0 && season == nil:
				releaseInfo = fmt.Sprintf(" - %02d", *episode)
			case ix.Settings.EnableSonarrCompatibility && season != nil && *season > 0:
				releaseInfo = fmt.Sprintf("S%02d", *season)

				if episode != nil && *episode > 0 {
					releaseInfo += fmt.Sprintf("E%02d - %02d", *episode, *episode)
				}
			}

			releaseInfo = strings.TrimSpace(releaseInfo)

			// Ignore these categories as they'll cause hell with the matcher
			// TV Special, DVD Special, BD Special
			if groupName == "TV Special" || groupName == "DVD Special" || groupName == "BD Special" {
				continue
			}

			if groupName == "TV Series" || groupName == "OVA" || groupName == "ONA" {
				categories = tvAnime
			}

			if groupName == "Movie" || groupName == "Live Action Movie" {
				categories = movies
			}

			switch categoryName {
			case "Manga", "Oneshot", "Anthology", "Manhwa", "Manhua", "Light Novel":
				categories = booksComics
			}

			if categoryName == "Novel" || categoryName == "Artbook" {
				categories = booksComics
			}

			if categoryName == "Game" || categoryName == "Visual Novel" {
				has := func(p string) bool { return slices.Contains(properties, p) }

				if has("PSP") {
					categories = []newznab.Category{newznab.Console, newznab.ConsolePSP}
				}

				if has("PS3") {
					categories = []newznab.Category{newznab.Console, newznab.ConsolePS3}
				}

				if has("PS Vita") {
					categories = []newznab.Category{newznab.Console, newznab.ConsolePSVita}
				}

				if has("3DS") {
					categories = []newznab.Category{newznab.Console, newznab.Console3DS}
				}

				if has("NDS") {
					categories = []newznab.Category{newznab.Console, newznab.ConsoleNDS}
				}

				if has("PSX") || has("PS2") || has("SNES") || has("NES") || has("GBA") || has("Switch") || has("N64") {
					categories = []newznab.Category{newznab.Console, newznab.ConsoleOther}
				}

				if has("PC") {
					categories = []newznab.Category{newznab.PCGames}
				}
			}

			switch categoryName {
			case "Single", "EP", "Album", "Compilation", "Soundtrack", "Remix CD", "PV", "Live Album", "Image CD", "Drama CD", "Vocal CD":
				switch {
				case slices.ContainsFunc(properties, func(p string) bool { return strings.Contains(p, "Lossless") }):
					categories = []newznab.Category{newznab.Audio, newznab.AudioLossless}
				case slices.ContainsFunc(properties, func(p string) bool { return strings.Contains(p, "MP3") }):
					categories = []newznab.Category{newznab.Audio, newznab.AudioMP3}
				default:
					categories = []newznab.Category{newznab.Audio, newznab.AudioOther}
				}
			}

			// We don't actually have a release name >.> so try to create one
			releaseGroup := ""
			for _, p := range slices.Backward(properties) {
				if slices.ContainsFunc(commonReleaseGroupsProperties, func(c string) bool { return startsWithIgnoreCase(p, c) }) && strings.Contains(p, "(") && strings.Contains(p, ")") {
					releaseGroup = p
					break
				}
			}

			if strings.TrimSpace(releaseGroup) != "" {
				start := strings.Index(releaseGroup, "(")
				releaseGroup = "[" + releaseGroup[start+1:strings.Index(releaseGroup, ")")] + "] "
			} else {
				releaseGroup = ""
			}

			var infoString strings.Builder
			for _, p := range properties {
				infoString.WriteString("[" + p + "]")
			}

			newRelease := func(title, guid string) port.TorrentInfo {
				return port.TorrentInfo{
					MinimumRatio:         1,
					MinimumSeedTime:      minimumSeedTime,
					Title:                title,
					InfoURL:              details,
					GUID:                 guid,
					DownloadURL:          link,
					PublishDate:          publishDate,
					Categories:           categories,
					Size:                 size,
					Seeders:              seeders,
					Peers:                peers,
					Grabs:                snatched,
					Files:                fileCount,
					DownloadVolumeFactor: rawDownMultiplier,
					UploadVolumeFactor:   rawUpMultiplier,
				}
			}

			if ix.Settings.UseFilenameForSingleEpisodes {
				files := torrent.Files

				if len(files) > 1 {
					files = slices.DeleteFunc(slices.Clone(files), func(f animeBytesFile) bool {
						return equalsIgnoreCase(excludedFileExtensions, extension(f.FileName))
					})
				}

				if len(files) == 1 {
					fileName := files[0].FileName

					releaseInfos = append(releaseInfos, newRelease(fileName, details+"?nh="+calculateMd5(fileName)))
				}
			}

			yearText := ""
			if year != nil {
				yearText = strconv.FormatInt(int64(*year), 10)
			}
			useYearInTitle := year != nil && *year > 0 && slices.ContainsFunc(torrent.Files, func(f animeBytesFile) bool { return strings.Contains(f.FileName, yearText) })

			for _, title := range synonyms {
				var releaseTitle string
				if groupName == "Movie" || groupName == "Live Action Movie" {
					releaseTitle = releaseGroup + title + " " + yearText + " " + infoString.String()
				} else {
					yearPart := ""
					if useYearInTitle {
						yearPart = " " + yearText
					}
					releaseTitle = releaseGroup + title + yearPart + " " + releaseInfo + " " + infoString.String()
				}

				releaseInfos = append(releaseInfos, newRelease(strings.TrimSpace(releaseTitle), details+"?nh="+calculateMd5(title)))
			}
		}
	}

	slices.SortStableFunc(releaseInfos, func(a, b port.TorrentInfo) int { return b.PublishDate.Compare(a.PublishDate) })

	return releaseInfos, nil
}

func parseSeasonFromTitles(titles []string) *int {
	for _, title := range titles {
		if m := advancedSeasonRegex.FindStringSubmatch(title); m != nil {
			return coerceInt(m[1] + m[2])
		}

		if m := seasonCharactersRegex.FindStringSubmatch(title); m != nil {
			n := len(m[1])
			return &n
		}

		if season, ok := seasonNumber(title); ok {
			return &season
		}
	}

	return nil
}

// seasonNumber is seasonNumberRegex,
// \b(?<!(Part|No\.)[- ._])(?<!\d[/])(?<!\#)(?:S)?(?<season>[2-9])$, which
// RE2 can't express: a digit from 2 to 9 that ends the title, after an
// optional S, at a word start no "Part ", "No. ", digit and "/" or "#"
// comes before.
func seasonNumber(title string) (int, bool) {
	last, size := utf8.DecodeLastRuneInString(title)
	if last < '2' || last > '9' {
		return 0, false
	}
	digitAt := len(title) - size
	for _, at := range []int{digitAt - 1, digitAt} {
		if at < 0 || (at == digitAt-1 && title[at] != 'S') {
			continue
		}
		before := title[:at]
		prev, _ := utf8.DecodeLastRuneInString(before)
		if before != "" && isWord(prev) {
			continue
		}
		if hasPartOrNumberBefore(before) || strings.HasSuffix(before, "#") || digitSlashBefore(before) {
			continue
		}
		return int(last - '0'), true
	}
	return 0, false
}

func hasPartOrNumberBefore(before string) bool {
	if before == "" || !strings.ContainsRune("- ._", rune(before[len(before)-1])) {
		return false
	}
	rest := before[:len(before)-1]
	return strings.HasSuffix(rest, "Part") || strings.HasSuffix(rest, "No.")
}

func digitSlashBefore(before string) bool {
	if !strings.HasSuffix(before, "/") {
		return false
	}
	prev, _ := utf8.DecodeLastRuneInString(before[:len(before)-1])
	return unicode.IsDigit(prev)
}

func isWord(r rune) bool {
	return unicode.IsLetter(r) || unicode.Is(unicode.Mn, r) || unicode.IsDigit(r) || unicode.Is(unicode.Pc, r)
}

// replaceWord is Regex.Replace on \b(re)\b: each match with a word
// boundary on both sides, in .NET's Unicode sense, becomes with.
func replaceWord(re *regexp.Regexp, s, with string) string {
	var b strings.Builder
	last := 0
	for _, m := range re.FindAllStringIndex(s, -1) {
		prev, _ := utf8.DecodeLastRuneInString(s[:m[0]])
		next, _ := utf8.DecodeRuneInString(s[m[1]:])
		if (m[0] > 0 && isWord(prev)) || (m[1] < len(s) && isWord(next)) {
			continue
		}
		b.WriteString(s[last:m[0]])
		b.WriteString(with)
		last = m[1]
	}
	b.WriteString(s[last:])
	return b.String()
}

// splitProperties is Split(PropertiesSeparator, TrimEntries |
// RemoveEmptyEntries): at each place the first separator that starts
// there ends an entry.
func splitProperties(s string) []string {
	var entries []string
	start := 0
	for i := 0; i < len(s); {
		sep := ""
		for _, p := range propertiesSeparator {
			if strings.HasPrefix(s[i:], p) {
				sep = p
				break
			}
		}
		if sep == "" {
			i++
			continue
		}
		entries = append(entries, s[start:i])
		i += len(sep)
		start = i
	}
	entries = append(entries, s[start:])

	var kept []string
	for _, e := range entries {
		if e = strings.TrimSpace(e); e != "" {
			kept = append(kept, e)
		}
	}
	return kept
}

// toHashSet is ToHashSet's ordinal set, in the order its members came.
func toHashSet(s []string) []string {
	var set []string
	for _, v := range s {
		if !slices.Contains(set, v) {
			set = append(set, v)
		}
	}
	return set
}

// ignoreCaseSet is a HashSet under OrdinalIgnoreCase, in the order its
// members came.
type ignoreCaseSet []string

func (s *ignoreCaseSet) add(v string) {
	if !slices.ContainsFunc(*s, func(m string) bool { return strings.EqualFold(m, v) }) {
		*s = append(*s, v)
	}
}

// extension is Path.GetExtension: from the last dot after the last
// separator, "" for none or a trailing dot.
func extension(fileName string) string {
	dot := strings.LastIndexByte(fileName, '.')
	if dot < 0 || dot == len(fileName)-1 || strings.ContainsAny(fileName[dot:], `/\`) {
		return ""
	}
	return fileName[dot:]
}

// calculateMd5 is HashUtil.CalculateMd5: the MD5 of the string's ASCII
// bytes, a "?" for each other character, in upper-case hex.
func calculateMd5(s string) string {
	ascii := make([]byte, 0, len(s))
	for _, r := range s {
		if r > unicode.MaxASCII {
			r = '?'
		}
		ascii = append(ascii, byte(r))
	}
	sum := md5.Sum(ascii) //nolint:gosec // see the import
	return strings.ToUpper(hex.EncodeToString(sum[:]))
}

func coerceInt(s string) *int {
	n, err := parseutil.CoerceInt(s)
	if err != nil {
		return nil
	}
	return &n
}

// SlipStream: Prowlarr's validator is the required flag on the
// credentials. Its passkey length check is gone: SlipStream's settings
// have no length rule. The username is tagged key so a refusal names
// both.
type animeBytesSettings struct {
	Username                     string `port:"username,key" label:"Username" help:"Your AnimeBytes username."`
	port.PasskeyField            `help:"Your AnimeBytes passkey."`
	FreeleechOnly                bool `port:"freeleechOnly" label:"Freeleech only" help:"Search only freeleech Releases."`
	ExcludeRaw                   bool `port:"excludeRaw" label:"Exclude RAW" help:"Leave RAW torrents out of the results."`
	ExcludeHentai                bool `port:"excludeHentai" label:"Exclude hentai" help:"Leave hentai torrents out of the results."`
	SearchByYear                 bool `port:"searchByYear" label:"Search by year" help:"Send a year that ends the search text as its own parameter."`
	EnableSonarrCompatibility    bool `port:"enableSonarrCompatibility" label:"Add seasons to titles" default:"true" help:"Add the season to Release titles so a TV library can match them. Some Releases get the wrong season."`
	UseFilenameForSingleEpisodes bool `port:"useFilenameForSingleEpisodes" label:"Use file names for single episodes" default:"true" help:"Add a Release titled with the file name of a torrent that holds one episode."`
	AddJapaneseTitle             bool `port:"addJapaneseTitle" label:"Add the Japanese title" default:"true" help:"Add a Release under the Japanese title, in kanji, hiragana or katakana."`
	AddRomajiTitle               bool `port:"addRomajiTitle" label:"Add the Romaji title" default:"true" help:"Add a Release under the Romaji title, such as Shingeki no Kyojin for Attack on Titan."`
	AddAlternativeTitle          bool `port:"addAlternativeTitle" label:"Add alternative titles" default:"true" help:"Add a Release under each alternative title, such as AoT for Attack on Titan."`
}

type animeBytesResponse struct {
	Matches serializer.STJInt `json:"Matches"`
	Groups  []animeBytesGroup `json:"Groups"`
	Error   string            `json:"Error"`
}

type animeBytesGroup struct {
	ID           serializer.STJInt   `json:"ID"`
	CategoryName string              `json:"CategoryName"`
	FullName     string              `json:"FullName"`
	GroupName    string              `json:"GroupName"`
	SeriesName   string              `json:"SeriesName"`
	Year         *serializer.STJInt  `json:"Year"`
	Image        string              `json:"Image"`
	Synonyms     map[string]string   `json:"SynonymnsV2"`
	Description  string              `json:"Description"`
	Tags         []string            `json:"Tags"`
	Torrents     []animeBytesTorrent `json:"Torrents"`
}

type animeBytesTorrent struct {
	ID                serializer.STJInt      `json:"ID"`
	EditionData       *animeBytesEditionData `json:"EditionData"`
	RawDownMultiplier serializer.STJFloat    `json:"RawDownMultiplier"`
	RawUpMultiplier   serializer.STJFloat    `json:"RawUpMultiplier"`
	Link              string                 `json:"Link"`
	Property          string                 `json:"Property"`
	Snatched          serializer.STJInt      `json:"Snatched"`
	Seeders           serializer.STJInt      `json:"Seeders"`
	Leechers          serializer.STJInt      `json:"Leechers"`
	Size              serializer.STJInt      `json:"Size"`
	FileCount         serializer.STJInt      `json:"FileCount"`
	Files             []animeBytesFile       `json:"FileList"`
	UploadTime        string                 `json:"UploadTime"`
}

type animeBytesFile struct {
	FileName string            `json:"filename"`
	FileSize serializer.STJInt `json:"size"`
}

type animeBytesEditionData struct {
	EditionTitle string `json:"EditionTitle"`
}
