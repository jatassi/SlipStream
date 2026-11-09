// Ported from Prowlarr's src/NzbDrone.Core/Indexers/Definitions/Toloka.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

package toloka

import (
	_ "embed"
	"slices"
	"strconv"
	"strings"

	"github.com/jatassi/slipstream/internal/newznab"
	"github.com/jatassi/slipstream/internal/ports/port"
	"github.com/jatassi/slipstream/internal/prowlarr/datetimeutil"
	"github.com/jatassi/slipstream/internal/prowlarr/dom"
	"github.com/jatassi/slipstream/internal/prowlarr/parseutil"
	"github.com/jatassi/slipstream/internal/regex"
)

//go:embed upstream.json
var upstream []byte

type ix = port.Indexer[TolokaSettings]

var Port = port.Define(port.Entry{
	ID:           "toloka",
	Name:         "Toloka.to",
	IndexerURLs:  []string{"https://toloka.to/"},
	Description:  "Toloka.to is a semi-private Ukrainian Tracker for movies, TV, music, books, games and software.",
	Language:     "uk-UA",
	Privacy:      port.SemiPrivate,
	Capabilities: SetCapabilities(),
	Upstream:     port.ReadUpstream(upstream),
}, port.Impl[TolokaSettings]{
	Login: port.FormLogin[TolokaSettings]{DoLogin: DoLogin, CheckIfLoginNeeded: CheckIfLoginNeeded},
	Movie: GetSearchRequests,
	TV:    GetTvSearchRequests,
	Basic: GetSearchRequests,
	Parse: ParseResponse,
})

func DoLogin(in *port.SignIn[TolokaSettings]) error {
	loginURL := in.BaseURL.JoinPath("login.php")

	response, err := in.Do(port.Post(loginURL).
		AddFormParameter("username", in.Settings.Username).
		AddFormParameter("password", string(in.Settings.Password)).
		AddFormParameter("autologin", "on").
		AddFormParameter("ssl", "on").
		AddFormParameter("redirect", "").
		AddFormParameter("login", "Вхід").
		SetHeader("Content-Type", "application/x-www-form-urlencoded").
		SetHeader("Referer", loginURL.String()))
	if err != nil {
		return err
	}

	if CheckIfLoginNeeded(response) {
		doc := dom.Parse(response.Content)
		// SlipStream: no fallback message, since the stop sentence names
		// the sign-in fields and Prowlarr's fallback isn't the site's words.
		errorMessage := doc.QuerySelector("table.forumline table span.gen").FirstChild().TextContent()

		return port.IndexerAuthError(strings.TrimSpace(errorMessage))
	}

	return nil
}

func CheckIfLoginNeeded(httpResponse *port.IndexerResponse) bool {
	return !strings.Contains(httpResponse.Content, "logout=true")
}

func SetCapabilities() port.Capabilities {
	// SlipStream: MusicSearchParams and BookSearchParams have no mode in
	// SlipStream, and SupportsRawSearch no meaning. Music and book
	// searches go through Basic.
	return port.Capabilities{
		TvSearchParams:    []port.Param{port.Q, port.Season, port.Ep},
		MovieSearchParams: []port.Param{port.Q},
		Categories: port.Categories(
			port.Map("117", newznab.Movies, "Українське кіно"),
			port.Map("84", newznab.Movies, "|-Мультфільми і казки"),
			port.Map("42", newznab.Movies, "|-Художні фільми"),
			port.Map("124", newznab.TV, "|-Телесеріали"),
			port.Map("125", newznab.TV, "|-Мультсеріали"),
			port.Map("129", newznab.Movies, "|-АртХаус"),
			port.Map("219", newznab.Movies, "|-Аматорське відео"),
			port.Map("118", newznab.Movies, "Українське озвучення"),
			port.Map("16", newznab.Movies, "|-Фільми"),
			port.Map("32", newznab.TV, "|-Телесеріали"),
			port.Map("19", newznab.Movies, "|-Мультфільми"),
			port.Map("44", newznab.TV, "|-Мультсеріали"),
			port.Map("127", newznab.TVAnime, "|-Аніме"),
			port.Map("55", newznab.Movies, "|-АртХаус"),
			port.Map("94", newznab.MoviesOther, "|-Трейлери"),
			port.Map("144", newznab.Movies, "|-Короткометражні"),
			port.Map("190", newznab.Movies, "Українські субтитри"),
			port.Map("70", newznab.Movies, "|-Фільми"),
			port.Map("192", newznab.TV, "|-Телесеріали"),
			port.Map("193", newznab.Movies, "|-Мультфільми"),
			port.Map("195", newznab.TV, "|-Мультсеріали"),
			port.Map("194", newznab.TVAnime, "|-Аніме"),
			port.Map("196", newznab.Movies, "|-АртХаус"),
			port.Map("197", newznab.Movies, "|-Короткометражні"),
			port.Map("225", newznab.TVDocumentary, "Документальні фільми українською"),
			port.Map("21", newznab.TVDocumentary, "|-Українські наукові документальні фільми"),
			port.Map("131", newznab.TVDocumentary, "|-Українські історичні документальні фільми"),
			port.Map("226", newznab.TVDocumentary, "|-BBC"),
			port.Map("227", newznab.TVDocumentary, "|-Discovery"),
			port.Map("228", newznab.TVDocumentary, "|-National Geographic"),
			port.Map("229", newznab.TVDocumentary, "|-History Channel"),
			port.Map("230", newznab.TVDocumentary, "|-Інші іноземні документальні фільми"),
			port.Map("119", newznab.TVOther, "Телепередачі українською"),
			port.Map("18", newznab.TVOther, "|-Музичне відео"),
			port.Map("132", newznab.TVOther, "|-Телевізійні шоу та програми"),
			port.Map("157", newznab.TVSport, "Український спорт"),
			port.Map("235", newznab.TVSport, "|-Олімпіада"),
			port.Map("170", newznab.TVSport, "|-Чемпіонати Європи з футболу"),
			port.Map("162", newznab.TVSport, "|-Чемпіонати світу з футболу"),
			port.Map("166", newznab.TVSport, "|-Чемпіонат та Кубок України з футболу"),
			port.Map("167", newznab.TVSport, "|-Єврокубки"),
			port.Map("168", newznab.TVSport, "|-Збірна України"),
			port.Map("169", newznab.TVSport, "|-Закордонні чемпіонати"),
			port.Map("54", newznab.TVSport, "|-Футбольне відео"),
			port.Map("158", newznab.TVSport, "|-Баскетбол, хоккей, волейбол, гандбол, футзал"),
			port.Map("159", newznab.TVSport, "|-Бокс, реслінг, бойові мистецтва"),
			port.Map("160", newznab.TVSport, "|-Авто, мото"),
			port.Map("161", newznab.TVSport, "|-Інший спорт, активний відпочинок"),
			port.Map("136", newznab.Other, "HD українською"),
			port.Map("96", newznab.MoviesHD, "|-Фільми в HD"),
			port.Map("173", newznab.TVHD, "|-Серіали в HD"),
			port.Map("139", newznab.MoviesHD, "|-Мультфільми в HD"),
			port.Map("174", newznab.TVHD, "|-Мультсеріали в HD"),
			port.Map("140", newznab.TVDocumentary, "|-Документальні фільми в HD"),
			port.Map("120", newznab.MoviesDVD, "DVD українською"),
			port.Map("66", newznab.MoviesDVD, "|-Художні фільми та серіали в DVD"),
			port.Map("137", newznab.MoviesDVD, "|-Мультфільми та мультсеріали в DVD"),
			port.Map("137", newznab.TV, "|-Мультфільми та мультсеріали в DVD"),
			port.Map("138", newznab.MoviesDVD, "|-Документальні фільми в DVD"),
			port.Map("237", newznab.Movies, "Відео для мобільних (iOS, Android, Windows Phone)"),
			port.Map("33", newznab.AudioVideo, "Звукові доріжки та субтитри"),
			port.Map("8", newznab.Audio, "Українська музика (lossy)"),
			port.Map("23", newznab.Audio, "|-Поп, Естрада"),
			port.Map("24", newznab.Audio, "|-Джаз, Блюз"),
			port.Map("43", newznab.Audio, "|-Етно, Фольклор, Народна, Бардівська"),
			port.Map("35", newznab.Audio, "|-Інструментальна, Класична та неокласична"),
			port.Map("37", newznab.Audio, "|-Рок, Метал, Альтернатива, Панк, СКА"),
			port.Map("36", newznab.Audio, "|-Реп, Хіп-хоп, РнБ"),
			port.Map("38", newznab.Audio, "|-Електронна музика"),
			port.Map("56", newznab.Audio, "|-Невидане"),
			port.Map("98", newznab.AudioLossless, "Українська музика (lossless)"),
			port.Map("100", newznab.AudioLossless, "|-Поп, Естрада"),
			port.Map("101", newznab.AudioLossless, "|-Джаз, Блюз"),
			port.Map("102", newznab.AudioLossless, "|-Етно, Фольклор, Народна, Бардівська"),
			port.Map("103", newznab.AudioLossless, "|-Інструментальна, Класична та неокласична"),
			port.Map("104", newznab.AudioLossless, "|-Рок, Метал, Альтернатива, Панк, СКА"),
			port.Map("105", newznab.AudioLossless, "|-Реп, Хіп-хоп, РнБ"),
			port.Map("106", newznab.AudioLossless, "|-Електронна музика"),
			port.Map("11", newznab.Books, "Друкована література"),
			port.Map("134", newznab.Books, "|-Українська художня література (до 1991 р.)"),
			port.Map("177", newznab.Books, "|-Українська художня література (після 1991 р.)"),
			port.Map("178", newznab.Books, "|-Зарубіжна художня література"),
			port.Map("179", newznab.Books, "|-Наукова література (гуманітарні дисципліни)"),
			port.Map("180", newznab.Books, "|-Наукова література (природничі дисципліни)"),
			port.Map("183", newznab.Books, "|-Навчальна та довідкова"),
			port.Map("181", newznab.BooksMags, "|-Періодика"),
			port.Map("182", newznab.Books, "|-Батькам та малятам"),
			port.Map("184", newznab.BooksComics, "|-Графіка (комікси, манґа, BD та інше)"),
			port.Map("185", newznab.AudioAudiobook, "Аудіокниги українською"),
			port.Map("135", newznab.AudioAudiobook, "|-Українська художня література"),
			port.Map("186", newznab.AudioAudiobook, "|-Зарубіжна художня література"),
			port.Map("187", newznab.AudioAudiobook, "|-Історія, біографістика, спогади"),
			port.Map("189", newznab.AudioAudiobook, "|-Сирий матеріал"),
			port.Map("9", newznab.PC, "Windows"),
			port.Map("25", newznab.PC, "|-Windows"),
			port.Map("199", newznab.PC, "|-Офіс"),
			port.Map("200", newznab.PC, "|-Антивіруси та безпека"),
			port.Map("201", newznab.PC, "|-Мультимедія"),
			port.Map("202", newznab.PC, "|-Утиліти, обслуговування, мережа"),
			port.Map("239", newznab.PC, "Linux, Mac OS"),
			port.Map("26", newznab.PC, "|-Linux"),
			port.Map("27", newznab.PCMac, "|-Mac OS"),
			port.Map("240", newznab.PC, "Інші OS"),
			port.Map("211", newznab.PCMobileAndroid, "|-Android"),
			port.Map("122", newznab.PCMobileiOS, "|-iOS"),
			port.Map("40", newznab.PCMobileOther, "|-Інші мобільні платформи"),
			port.Map("241", newznab.Other, "Інше"),
			port.Map("203", newznab.Other, "|-Інфодиски, електронні підручники, відеоуроки"),
			port.Map("12", newznab.Other, "|-Шпалери, фотографії та зображення"),
			port.Map("249", newznab.Other, "|-Веб-скрипти"),
			port.Map("10", newznab.PCGames, "Ігри українською"),
			port.Map("28", newznab.PCGames, "|-PC ігри"),
			port.Map("259", newznab.PCGames, "|-Mac ігри"),
			port.Map("29", newznab.PCGames, "|-Українізації, доповнення, патчі..."),
			port.Map("30", newznab.PCGames, "|-Мобільні та консольні ігри"),
			port.Map("41", newznab.PCMobileiOS, "|-iOS"),
			port.Map("212", newznab.PCMobileAndroid, "|-Android"),
			port.Map("205", newznab.PCGames, "Переклад ігор українською"),
		),
	}
}

func GetSearchRequests(ix *ix, searchCriteria port.SearchCriteria) (port.Chain, error) {
	return port.One(GetPagedRequests(ix, searchCriteria.SanitizedSearchTerm(), searchCriteria.Categories))
}

func GetTvSearchRequests(ix *ix, searchCriteria port.SearchCriteria) (port.Chain, error) {
	term := searchCriteria.SanitizedSearchTerm()

	if searchCriteria.Season != nil && *searchCriteria.Season > 0 {
		term += " Сезон " + strconv.Itoa(*searchCriteria.Season)
	}

	return port.One(GetPagedRequests(ix, term, searchCriteria.Categories))
}

func GetPagedRequests(ix *ix, term string, categories []newznab.ID) *port.Request {
	nm := ""
	if strings.TrimSpace(term) != "" {
		nm = strings.ReplaceAll(term, "-", " ")
	}

	// SlipStream: AddQueryParam writes a space as %20 where
	// GetQueryString writes '+'. The site reads both as a space.
	request := port.Get(ix.URL("tracker.php")).Accept("text/html").
		AddQueryParam("o", "1").
		AddQueryParam("s", "2").
		AddQueryParam("nm", nm)

	if ix.Settings.FreeleechOnly {
		request.AddQueryParam("sds", "1")
	}

	for _, cat := range ix.Categories.TrackerIDs(categories) {
		request.AddQueryParam("f[]", cat)
	}

	return request
}

func ParseResponse(ix *ix, indexerResponse *port.IndexerResponse) ([]port.TorrentInfo, error) {
	var releaseInfos []port.TorrentInfo

	doc := dom.Parse(indexerResponse.Content)

	rows := doc.QuerySelectorAll("table.forumline > tbody > tr[class*=prow]")
	for _, row := range rows {
		downloadURL := row.QuerySelector("td:nth-child(6) > a")

		// Expects moderation
		if downloadURL == nil || !downloadURL.HasAttribute("href") {
			continue
		}

		infoURL := ix.BaseURL.String() + row.QuerySelector("td:nth-child(3) > a").GetAttribute("href")
		title := strings.TrimSpace(row.QuerySelector("td:nth-child(3) > a").TextContent())

		categoryLink := row.QuerySelector("td:nth-child(2) > a").GetAttribute("href")
		cat, err := parseutil.GetArgumentFromQueryString(categoryLink, "f")
		if err != nil {
			return nil, err
		}
		categories := ix.Categories.FromTracker(cat)

		seeders, err := parseutil.CoerceInt(row.QuerySelector("td:nth-child(10) > b").TextContent())
		if err != nil {
			return nil, err
		}
		leechers, err := parseutil.CoerceInt(strings.TrimSpace(row.QuerySelector("td:nth-child(11) > b").TextContent()))
		if err != nil {
			return nil, err
		}

		// 2023-01-21
		added := strings.TrimSpace(row.QuerySelector("td:nth-child(13)").TextContent())

		size, err := parseutil.GetBytes(strings.TrimSpace(row.QuerySelector("td:nth-child(7)").TextContent()))
		if err != nil {
			return nil, err
		}
		grabs, err := parseutil.CoerceInt(row.QuerySelector("td:nth-child(9)").TextContent())
		if err != nil {
			return nil, err
		}
		publishDate, err := datetimeutil.FromFuzzyTime(added, ix.Clock)
		if err != nil {
			return nil, err
		}

		release := port.TorrentInfo{
			GUID:        infoURL,
			InfoURL:     infoURL,
			DownloadURL: ix.BaseURL.String() + downloadURL.GetAttribute("href"),
			Title:       titleParser.Parse(title, categories, ix.Settings.StripCyrillicLetters),
			// SlipStream: a Release has no description, so the site's own
			// title isn't kept beside the parsed one.
			Categories:           categories,
			Seeders:              seeders,
			Peers:                seeders + leechers,
			Size:                 size,
			Grabs:                grabs,
			PublishDate:          publishDate,
			DownloadVolumeFactor: 1,
			UploadVolumeFactor:   1,
			MinimumRatio:         1,
			MinimumSeedTime:      0,
		}

		switch {
		case row.QuerySelector(`img[src="images/gold.gif"], img[src="images/authors.gif"]`) != nil:
			release.DownloadVolumeFactor = 0
		case row.QuerySelector(`img[src="images/silver.gif"]`) != nil:
			release.DownloadVolumeFactor = 0.5
		case row.QuerySelector(`img[src="images/bronze.gif"]`) != nil:
			release.DownloadVolumeFactor = 0.75
		}

		releaseInfos = append(releaseInfos, release)
	}

	return releaseInfos, nil
}

// TolokaTitleParser is Toloka's title cleanup. Its patterns are .NET's,
// so they run on internal/regex, which reads them as .NET does: balancing
// groups, \p{IsCyrillic}, and \w, \W, \s, \d and \b over all of
// Unicode.
type TolokaTitleParser struct{}

var titleParser TolokaTitleParser

// replace is Regex.Replace. A fixed pattern has no match limit, so it errs
// only on a replacement it can't read, which a test would meet.
func replace(re *regex.Regex, input, replacement string) string {
	out, err := re.Replace(regex.TextOf(input), replacement)
	if err != nil {
		panic(err)
	}
	return out
}

// matches is Regex.Matches: each match's index, in runes, and value.
func matches(re *regex.Regex, input string) []regex.Match {
	out, err := re.FindAll(regex.TextOf(input))
	if err != nil {
		panic(err)
	}
	return out
}

var (
	FindTagsInTitlesRegexList = []*regex.Regex{
		regex.MustCompile(`\((?>\((?<c>)|[^()]+|\)(?<-c>))*(?(c)(?!))\)`, regex.Options{}),
		regex.MustCompile(`\[(?>\[(?<c>)|[^\[\]]+|\](?<-c>))*(?(c)(?!))\]`, regex.Options{}),
	}

	tvTitleCommaRegex     = regex.MustCompile(`\s(\d+),(\d+)`, regex.Options{})
	tvTitleCyrillicXRegex = regex.MustCompile(`([\s-])Х+([\)\]])`, regex.Options{IgnoreCase: true})

	tvTitleMultipleSeasonsRegex = regex.MustCompile(`(?:Сезон|Seasons?)\s*[:]*\s+(\d+-\d+)`, regex.Options{IgnoreCase: true})

	tvTitleUkrSeasonEpisodeOfRegex = regex.MustCompile(`Сезон\s*[:]*\s+(\d+).+(?:Серії|Серія|Серій|Епізоди?)+\s*[:]*\s+(\d+(?:-\d+)?)\s*з\s*([\w?])`, regex.Options{IgnoreCase: true})
	tvTitleUkrSeasonEpisodeRegex   = regex.MustCompile(`Сезон\s*[:]*\s+(\d+).+(?:Серії|Серія|Серій|Епізоди?)+\s*[:]*\s+(\d+(?:-\d+)?)`, regex.Options{IgnoreCase: true})
	tvTitleUkrSeasonRegex          = regex.MustCompile(`Сезон\s*[:]*\s+(\d+)`, regex.Options{IgnoreCase: true})
	tvTitleUkrEpisodeOfRegex       = regex.MustCompile(`(?:Серії|Серія|Серій|Епізоди?)+\s*[:]*\s+(\d+(?:-\d+)?)\s*з\s*([\w?])`, regex.Options{IgnoreCase: true})
	tvTitleUkrEpisodeRegex         = regex.MustCompile(`(?:Серії|Серія|Серій|Епізоди?)+\s*[:]*\s+(\d+(?:-\d+)?)`, regex.Options{IgnoreCase: true})

	tvTitleEngSeasonEpisodeOfRegex = regex.MustCompile(`Season\s*[:]*\s+(\d+).+(?:Episodes?)+\s*[:]*\s+(\d+(?:-\d+)?)\s*of\s*([\w?])`, regex.Options{IgnoreCase: true})
	tvTitleEngSeasonEpisodeRegex   = regex.MustCompile(`Season\s*[:]*\s+(\d+).+(?:Episodes?)+\s*[:]*\s+(\d+(?:-\d+)?)`, regex.Options{IgnoreCase: true})
	tvTitleEngSeasonRegex          = regex.MustCompile(`Season\s*[:]*\s+(\d+(?:-\d+)?)`, regex.Options{IgnoreCase: true})
	tvTitleEngEpisodeOfRegex       = regex.MustCompile(`(?:Episodes?)+\s*[:]*\s+(\d+(?:-\d+)?)\s*of\s*([\w?])`, regex.Options{IgnoreCase: true})
	tvTitleEngEpisodeRegex         = regex.MustCompile(`(?:Episodes?)+\s*[:]+\s*[:]*\s+(\d+(?:-\d+)?)`, regex.Options{IgnoreCase: true})

	stripCyrillicRegex = regex.MustCompile(`(\([\p{IsCyrillic}\W]+\))|(^[\p{IsCyrillic}\W\d]+\/ )|([\p{IsCyrillic} \-]+,+)|([\p{IsCyrillic}]+)`, regex.Options{IgnoreCase: true})

	dashRegex          = regex.MustCompile(`\p{Pd}`, regex.Options{IgnoreCase: true})
	ripRegex           = regex.MustCompile(`\b-Rip\b`, regex.Options{IgnoreCase: true})
	hdtvRipRegex       = regex.MustCompile(`\bHDTVRip\b`, regex.Options{IgnoreCase: true})
	webDLRipRegex      = regex.MustCompile(`\bWEB-DLRip\b`, regex.Options{IgnoreCase: true})
	webDLRipShortRegex = regex.MustCompile(`\bWEBDLRip\b`, regex.Options{IgnoreCase: true})
	webDLRegex         = regex.MustCompile(`\bWEBDL\b`, regex.Options{IgnoreCase: true})
	openSlashRegex     = regex.MustCompile(`\(\s*\/\s*`, regex.Options{})
	closeSlashRegex    = regex.MustCompile(`\s*\/\s*\)`, regex.Options{})
	emptyTagRegex      = regex.MustCompile(`[\[\(]\s*[\)\]]`, regex.Options{})
	spacesRegex        = regex.MustCompile(`\s+`, regex.Options{})
)

const trimChars = ` &,.!?+-_|/\:;ʼ` + "`"

func (TolokaTitleParser) Parse(title string, categories []newznab.Category, stripCyrillicLetters bool) string {
	// https://www.fileformat.info/info/unicode/category/Pd/list.htm
	title = replace(dashRegex, title, "-")

	if IsAnyTvCategory(categories) {
		title = replace(tvTitleCommaRegex, title, " $1-$2")
		title = replace(tvTitleCyrillicXRegex, title, "$1XX$2")

		// special case for multiple seasons
		title = replace(tvTitleMultipleSeasonsRegex, title, "S$1")

		title = replace(tvTitleUkrSeasonEpisodeOfRegex, title, "S$1E$2 of $3")
		title = replace(tvTitleUkrSeasonEpisodeRegex, title, "S$1E$2")
		title = replace(tvTitleUkrSeasonRegex, title, "S$1")
		title = replace(tvTitleUkrEpisodeOfRegex, title, "E$1 of $2")
		title = replace(tvTitleUkrEpisodeRegex, title, "E$1")

		title = replace(tvTitleEngSeasonEpisodeOfRegex, title, "S$1E$2 of $3")
		title = replace(tvTitleEngSeasonEpisodeRegex, title, "S$1E$2")
		title = replace(tvTitleEngSeasonRegex, title, "S$1")
		title = replace(tvTitleEngEpisodeOfRegex, title, "E$1 of $2")
		title = replace(tvTitleEngEpisodeRegex, title, "E$1")
	}

	if stripCyrillicLetters {
		title = strings.Trim(replace(stripCyrillicRegex, title, ""), " -")
	}

	title = replace(ripRegex, title, "Rip")
	title = replace(hdtvRipRegex, title, "HDTV")
	title = replace(webDLRipRegex, title, "WEB-DL")
	title = replace(webDLRipShortRegex, title, "WEB-DL")
	title = replace(webDLRegex, title, "WEB-DL")

	title = MoveFirstTagsToEndOfReleaseTitle(title)

	title = replace(openSlashRegex, title, "(")
	title = replace(closeSlashRegex, title, ")")

	title = replace(emptyTagRegex, title, "")

	title = strings.Trim(title, trimChars)

	// replace multiple spaces with a single space
	title = replace(spacesRegex, title, " ")

	return strings.TrimSpace(title)
}

// IsAnyTvCategory is category.Contains(TV) or one of TV.SubCategories.
func IsAnyTvCategory(category []newznab.Category) bool {
	return slices.ContainsFunc(category, func(c newznab.Category) bool {
		return c.ID == newznab.TV.ID || (c.ID < newznab.CustomBase && c.Parent == newznab.TV.ID)
	})
}

func MoveFirstTagsToEndOfReleaseTitle(input string) string {
	output := strings.Trim(input, trimChars)
	for _, findTagsRegex := range FindTagsInTitlesRegexList {
		expectedIndex := 0
		for _, match := range matches(findTagsRegex, output) {
			if match.Index() > expectedIndex {
				substring := string([]rune(output)[expectedIndex:match.Index()])
				if strings.TrimSpace(substring) != "" {
					break
				}
				expectedIndex = match.Index()
			}

			tag := match.Value()
			output = strings.TrimSpace(strings.Replace(output, tag, "", 1) + " " + tag)
			expectedIndex += match.Length()
		}
	}

	return strings.TrimSpace(output)
}

type TolokaSettings struct {
	port.UserPass
	// SlipStream: labels and help reworded per WRITING.md.
	FreeleechOnly        bool `port:"freeleechOnly" label:"Freeleech only" help:"Search only freeleech Releases."`
	StripCyrillicLetters bool `port:"stripCyrillicLetters" label:"Strip Cyrillic letters" default:"true" help:"Remove Cyrillic letters from each Release title."`
}
