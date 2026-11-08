// Ported from Prowlarr's src/NzbDrone.Core/Indexers/Definitions/MyAnonamouse.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

package myanonamouse

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/jatassi/slipstream/internal/newznab"
	"github.com/jatassi/slipstream/internal/ports/port"
	"github.com/jatassi/slipstream/internal/prowlarr/datetimeutil"
	"github.com/jatassi/slipstream/internal/prowlarr/encoding"
	"github.com/jatassi/slipstream/internal/prowlarr/parseutil"
	"github.com/jatassi/slipstream/internal/prowlarr/stringutil"
)

//go:embed upstream.json
var upstream []byte

type ix = port.Indexer[myAnonamouseSettings]

var Port = port.Define(port.Entry{
	ID:           "myanonamouse",
	Name:         "MyAnonamouse",
	IndexerURLs:  []string{"https://www.myanonamouse.net/"},
	Description:  "MyAnonamouse is a private tracker for ebooks and audiobooks.",
	Privacy:      port.Private,
	PageSize:     100,
	Capabilities: setCapabilities(),
	Upstream:     port.ReadUpstream(upstream),
}, port.Impl[myAnonamouseSettings]{
	// SlipStream: the mam_id cookie is the pasted session, which the site
	// replaces on every response, so GetCookies, Test's cookie reset and
	// the parser's CookiesUpdater calls have nothing to do. The user class
	// HasUserVip caches for an hour is Derive's, read once per Indexer
	// revision on each Node.
	Login:  port.CookieLogin{Rotates: true},
	Basic:  basicSearch,
	Parse:  parseResponse,
	Derive: hasUserVip,
})

// SlipStream: SlipStream has no book search, so the Book search params
// aren't declared and a search is the Basic one.
func setCapabilities() port.Capabilities {
	return port.Capabilities{
		Categories: port.Categories(
			port.Map("13", newznab.AudioAudiobook, "AudioBooks"),
			port.Map("14", newznab.BooksEBook, "E-Books"),
			port.Map("15", newznab.AudioAudiobook, "Musicology"),
			port.Map("16", newznab.AudioAudiobook, "Radio"),
			port.Map("39", newznab.AudioAudiobook, "Audiobooks - Action/Adventure"),
			port.Map("49", newznab.AudioAudiobook, "Audiobooks - Art"),
			port.Map("50", newznab.AudioAudiobook, "Audiobooks - Biographical"),
			port.Map("83", newznab.AudioAudiobook, "Audiobooks - Business"),
			port.Map("51", newznab.AudioAudiobook, "Audiobooks - Computer/Internet"),
			port.Map("97", newznab.AudioAudiobook, "Audiobooks - Crafts"),
			port.Map("40", newznab.AudioAudiobook, "Audiobooks - Crime/Thriller"),
			port.Map("41", newznab.AudioAudiobook, "Audiobooks - Fantasy"),
			port.Map("106", newznab.AudioAudiobook, "Audiobooks - Food"),
			port.Map("42", newznab.AudioAudiobook, "Audiobooks - General Fiction"),
			port.Map("52", newznab.AudioAudiobook, "Audiobooks - General Non-Fic"),
			port.Map("98", newznab.AudioAudiobook, "Audiobooks - Historical Fiction"),
			port.Map("54", newznab.AudioAudiobook, "Audiobooks - History"),
			port.Map("55", newznab.AudioAudiobook, "Audiobooks - Home/Garden"),
			port.Map("43", newznab.AudioAudiobook, "Audiobooks - Horror"),
			port.Map("99", newznab.AudioAudiobook, "Audiobooks - Humor"),
			port.Map("84", newznab.AudioAudiobook, "Audiobooks - Instructional"),
			port.Map("44", newznab.AudioAudiobook, "Audiobooks - Juvenile"),
			port.Map("56", newznab.AudioAudiobook, "Audiobooks - Language"),
			port.Map("45", newznab.AudioAudiobook, "Audiobooks - Literary Classics"),
			port.Map("57", newznab.AudioAudiobook, "Audiobooks - Math/Science/Tech"),
			port.Map("85", newznab.AudioAudiobook, "Audiobooks - Medical"),
			port.Map("87", newznab.AudioAudiobook, "Audiobooks - Mystery"),
			port.Map("119", newznab.AudioAudiobook, "Audiobooks - Nature"),
			port.Map("88", newznab.AudioAudiobook, "Audiobooks - Philosophy"),
			port.Map("58", newznab.AudioAudiobook, "Audiobooks - Pol/Soc/Relig"),
			port.Map("59", newznab.AudioAudiobook, "Audiobooks - Recreation"),
			port.Map("46", newznab.AudioAudiobook, "Audiobooks - Romance"),
			port.Map("47", newznab.AudioAudiobook, "Audiobooks - Science Fiction"),
			port.Map("53", newznab.AudioAudiobook, "Audiobooks - Self-Help"),
			port.Map("89", newznab.AudioAudiobook, "Audiobooks - Travel/Adventure"),
			port.Map("100", newznab.AudioAudiobook, "Audiobooks - True Crime"),
			port.Map("108", newznab.AudioAudiobook, "Audiobooks - Urban Fantasy"),
			port.Map("48", newznab.AudioAudiobook, "Audiobooks - Western"),
			port.Map("111", newznab.AudioAudiobook, "Audiobooks - Young Adult"),
			port.Map("60", newznab.BooksEBook, "Ebooks - Action/Adventure"),
			port.Map("71", newznab.BooksEBook, "Ebooks - Art"),
			port.Map("72", newznab.BooksEBook, "Ebooks - Biographical"),
			port.Map("90", newznab.BooksEBook, "Ebooks - Business"),
			port.Map("61", newznab.BooksComics, "Ebooks - Comics/Graphic novels"),
			port.Map("73", newznab.BooksEBook, "Ebooks - Computer/Internet"),
			port.Map("101", newznab.BooksEBook, "Ebooks - Crafts"),
			port.Map("62", newznab.BooksEBook, "Ebooks - Crime/Thriller"),
			port.Map("63", newznab.BooksEBook, "Ebooks - Fantasy"),
			port.Map("107", newznab.BooksEBook, "Ebooks - Food"),
			port.Map("64", newznab.BooksEBook, "Ebooks - General Fiction"),
			port.Map("74", newznab.BooksEBook, "Ebooks - General Non-Fiction"),
			port.Map("102", newznab.BooksEBook, "Ebooks - Historical Fiction"),
			port.Map("76", newznab.BooksEBook, "Ebooks - History"),
			port.Map("77", newznab.BooksEBook, "Ebooks - Home/Garden"),
			port.Map("65", newznab.BooksEBook, "Ebooks - Horror"),
			port.Map("103", newznab.BooksEBook, "Ebooks - Humor"),
			port.Map("115", newznab.BooksEBook, "Ebooks - Illusion/Magic"),
			port.Map("91", newznab.BooksEBook, "Ebooks - Instructional"),
			port.Map("66", newznab.BooksEBook, "Ebooks - Juvenile"),
			port.Map("78", newznab.BooksEBook, "Ebooks - Language"),
			port.Map("67", newznab.BooksEBook, "Ebooks - Literary Classics"),
			port.Map("79", newznab.BooksMags, "Ebooks - Magazines/Newspapers"),
			port.Map("80", newznab.BooksTechnical, "Ebooks - Math/Science/Tech"),
			port.Map("92", newznab.BooksEBook, "Ebooks - Medical"),
			port.Map("118", newznab.BooksEBook, "Ebooks - Mixed Collections"),
			port.Map("94", newznab.BooksEBook, "Ebooks - Mystery"),
			port.Map("120", newznab.BooksEBook, "Ebooks - Nature"),
			port.Map("95", newznab.BooksEBook, "Ebooks - Philosophy"),
			port.Map("81", newznab.BooksEBook, "Ebooks - Pol/Soc/Relig"),
			port.Map("82", newznab.BooksEBook, "Ebooks - Recreation"),
			port.Map("68", newznab.BooksEBook, "Ebooks - Romance"),
			port.Map("69", newznab.BooksEBook, "Ebooks - Science Fiction"),
			port.Map("75", newznab.BooksEBook, "Ebooks - Self-Help"),
			port.Map("96", newznab.BooksEBook, "Ebooks - Travel/Adventure"),
			port.Map("104", newznab.BooksEBook, "Ebooks - True Crime"),
			port.Map("109", newznab.BooksEBook, "Ebooks - Urban Fantasy"),
			port.Map("70", newznab.BooksEBook, "Ebooks - Western"),
			port.Map("112", newznab.BooksEBook, "Ebooks - Young Adult"),
			port.Map("19", newznab.AudioAudiobook, "Guitar/Bass Tabs"),
			port.Map("20", newznab.AudioAudiobook, "Individual Sheet"),
			port.Map("24", newznab.AudioAudiobook, "Individual Sheet MP3"),
			port.Map("126", newznab.AudioAudiobook, "Instructional Book with Video"),
			port.Map("22", newznab.AudioAudiobook, "Instructional Media - Music"),
			port.Map("113", newznab.AudioAudiobook, "Lick Library - LTP/Jam With"),
			port.Map("114", newznab.AudioAudiobook, "Lick Library - Techniques/QL"),
			port.Map("17", newznab.AudioAudiobook, "Music - Complete Editions"),
			port.Map("26", newznab.AudioAudiobook, "Music Book"),
			port.Map("27", newznab.AudioAudiobook, "Music Book MP3"),
			port.Map("30", newznab.AudioAudiobook, "Sheet Collection"),
			port.Map("31", newznab.AudioAudiobook, "Sheet Collection MP3"),
			port.Map("127", newznab.AudioAudiobook, "Radio -  Comedy"),
			port.Map("130", newznab.AudioAudiobook, "Radio - Drama"),
			port.Map("128", newznab.AudioAudiobook, "Radio - Factual/Documentary"),
			port.Map("132", newznab.AudioAudiobook, "Radio - Reading"),
		),
	}
}

var sanitizeSearchQueryRegex = regexp.MustCompile(`[^\p{L}\p{Mn}\p{Nd}\p{Pc}]+`)

// SlipStream: SearchCriteria has no MinSize or MaxSize, so the size
// parameters aren't sent.
func getPagedRequests(c *ix, q port.SearchCriteria) []*port.Request {
	term := strings.TrimSpace(sanitizeSearchQueryRegex.ReplaceAllString(q.SanitizedSearchTerm(), " "))

	if strings.TrimSpace(q.SearchTerm) != "" && term == "" {
		return nil
	}

	searchType := "all"
	switch c.Settings.SearchType {
	case searchActive:
		searchType = "active"
	case searchFreeleech:
		searchType = "fl"
	case searchFreeleechOrVip:
		searchType = "fl-VIP"
	case searchVip:
		searchType = "VIP"
	case searchNotVip:
		searchType = "nVIP"
	case searchAll:
	}

	var parameters stringutil.NameValueCollection
	parameters.Add("tor[text]", term)
	parameters.Add("tor[searchType]", searchType)
	parameters.Add("tor[srchIn][title]", "true")
	parameters.Add("tor[srchIn][author]", "true")
	parameters.Add("tor[srchIn][narrator]", "true")
	parameters.Add("tor[searchIn]", "torrents")
	parameters.Add("tor[sortType]", "default")
	parameters.Add("tor[startNumber]", strconv.Itoa(q.Offset))
	perpage := q.Limit
	if perpage == 0 {
		perpage = 100
	}
	parameters.Add("perpage", strconv.Itoa(perpage))
	parameters.Add("thumbnails", "1")  // gives links for thumbnail sized versions of their posters
	parameters.Add("description", "1") // include the description

	if c.Settings.SearchInDescription {
		parameters.Set("tor[srchIn][description]", "true")
	}

	if c.Settings.SearchInSeries {
		parameters.Set("tor[srchIn][series]", "true")
	}

	if c.Settings.SearchInFilenames {
		parameters.Set("tor[srchIn][filenames]", "true")
	}

	for index, language := range c.Settings.SearchLanguages {
		parameters.Set(fmt.Sprintf("tor[browse_lang][%d]", index), strconv.Itoa(int(language)))
	}

	var catList []string
	for _, cat := range c.Categories.TrackerIDs(q.Categories) {
		if !slices.Contains(catList, cat) {
			catList = append(catList, cat)
		}
	}

	if len(catList) > 0 {
		for index, category := range catList {
			parameters.Set(fmt.Sprintf("tor[cat][%d]", index), category)
		}
	} else {
		parameters.Set("tor[cat][]", "0")
	}

	searchURL := c.URL("tor/js/loadSearchJSONbasic.php")

	if parameters.Count() > 0 {
		searchURL.RawQuery = parameters.GetQueryString()
	}

	return []*port.Request{port.Get(searchURL).Accept("application/json")}
}

func basicSearch(c *ix, q port.SearchCriteria) (port.Chain, error) {
	var pageableRequests port.Chain

	pageableRequests.Add(getPagedRequests(c, q)...)

	return pageableRequests, nil
}

var vipFreeleechUserClasses = []string{"VIP", "Elite VIP"}

// SlipStream: the messages are SlipStream's own, per WRITING.md. The 403
// is the expired cookie, which the framework words as cookie_expired.
func parseResponse(c *ix, r *port.IndexerResponse) ([]port.TorrentInfo, error) {
	// Throw auth errors here before we try to parse
	if r.StatusCode == http.StatusForbidden {
		return nil, port.IndexerAuthError("")
	}

	// Throw common http errors here before we try to parse
	if r.StatusCode != http.StatusOK {
		return nil, port.IndexerError("the Indexer responded with status %d", r.StatusCode)
	}

	if contentType := r.Headers.Get("Content-Type"); !strings.Contains(contentType, "application/json") {
		return nil, port.IndexerError("the Indexer responded with %q where application/json was expected", contentType)
	}

	var releaseInfos []port.TorrentInfo

	var jsonResponse myAnonamouseResponse
	if err := json.Unmarshal(r.Body, &jsonResponse); err != nil {
		return nil, port.IndexerError("read the Indexer's search response: %v", err)
	}

	errorText := jsonResponse.Error
	if strings.TrimSpace(errorText) != "" && strings.HasPrefix(strings.ToLower(errorText), strings.ToLower("Nothing returned, out of")) {
		return releaseInfos, nil
	}

	if jsonResponse.Data == nil {
		if jsonResponse.Message != "" {
			return nil, port.IndexerError("the Indexer's search response holds no Releases: %s", jsonResponse.Message)
		}
		return nil, port.IndexerError("the Indexer's search response holds no Releases")
	}

	hasUserVip := slices.ContainsFunc(vipFreeleechUserClasses, func(class string) bool { return strings.EqualFold(class, c.Derived["userClass"]) })

	for _, item := range jsonResponse.Data {
		id := int(item.ID)

		// SlipStream: a Release has no description, book title or author,
		// so only the title carries the author.
		title := item.Title

		if strings.TrimSpace(item.AuthorInfo) != "" {
			authors, err := orderedValues(item.AuthorInfo)
			if err != nil {
				return nil, port.IndexerError("read the authors of torrent %d in the Indexer's search response: %v", id, err)
			}
			for i, author := range authors {
				authors[i] = encoding.HTMLDecode(author)
			}
			author := strings.Join(authors[:min(len(authors), 5)], ", ")

			if strings.TrimSpace(author) != "" {
				title += " by " + author
			}
		}

		var flags []string

		if languageCode := item.LanguageCode; languageCode != "" {
			flags = append(flags, languageCode)
		}

		if filetype := item.Filetype; filetype != "" {
			flags = append(flags, strings.ToUpper(filetype))
		}

		if len(flags) > 0 {
			title += " [" + strings.Join(flags, " / ") + "]"
		}

		if item.Vip {
			title += " [VIP]"
		}

		isFreeLeech := bool(item.Free) || bool(item.PersonalFreeLeech) || (hasUserVip && bool(item.FreeVip))

		publishDate, err := datetimeutil.ParseExact(item.Added, "yyyy-MM-dd HH:mm:ss", datetimeutil.AssumeUniversal, c.Clock)
		if err != nil {
			return nil, port.IndexerError("read the date of torrent %d in the Indexer's search response: %v", id, err)
		}

		size, err := parseutil.GetBytes(item.Size)
		if err != nil {
			return nil, port.IndexerError("read the size of torrent %d in the Indexer's search response: %v", id, err)
		}

		downloadVolumeFactor := 1.0
		if isFreeLeech {
			downloadVolumeFactor = 0
		}

		infoURL := fmt.Sprintf("%st/%d", c.BaseURL, id)

		releaseInfos = append(releaseInfos, port.TorrentInfo{
			Title:                title,
			DownloadURL:          getDownloadURL(c, id, !isFreeLeech),
			InfoURL:              infoURL,
			GUID:                 infoURL,
			Categories:           c.Categories.FromTracker(strconv.Itoa(int(item.Category))),
			PublishDate:          publishDate,
			Grabs:                int(item.Grabs),
			Files:                int(item.NumFiles),
			Seeders:              int(item.Seeders),
			Peers:                int(item.Leechers + item.Seeders),
			Size:                 size,
			DownloadVolumeFactor: downloadVolumeFactor,
			UploadVolumeFactor:   1,
			MinimumRatio:         1,
			MinimumSeedTime:      259200, // 72 hours
		})
	}

	return releaseInfos, nil
}

func getDownloadURL(c *ix, torrentID int, canUseToken bool) string {
	u := port.NewURI(c.BaseURL).
		CombinePath("/tor/download.php").
		AddQueryParam("tid", strconv.Itoa(torrentID))

	if c.Settings.UseFreeleechWedge && canUseToken {
		u = u.AddQueryParam("fl", "1")
	}

	return u.FullURI()
}

// hasUserVip reads the account's user class, which the parser checks for
// VIP. A 403 is the expired cookie, as on a search.
func hasUserVip(c *ix) (map[string]string, error) {
	r, err := c.Fetch(port.Get(c.URL("jsonLoad.php")).Accept("application/json"))
	if err != nil {
		return nil, err
	}

	if r.StatusCode == http.StatusForbidden {
		return nil, port.IndexerAuthError("")
	}

	var jsonResponse myAnonamouseUserDataResponse
	if err := json.Unmarshal(r.Body, &jsonResponse); err != nil {
		return nil, port.IndexerError("read the Indexer's account response: %v", err)
	}

	return map[string]string{"userClass": strings.TrimSpace(jsonResponse.UserClass)}, nil
}

// orderedValues are the values of the JSON object s in its order, as
// STJson fills a Dictionary<string, string>.
func orderedValues(s string) ([]string, error) {
	dec := json.NewDecoder(bytes.NewReader([]byte(s)))
	if open, err := dec.Token(); err != nil || open != json.Delim('{') {
		return nil, errors.New("the authors aren't a JSON object")
	}
	var values []string
	for dec.More() {
		if _, err := dec.Token(); err != nil {
			return nil, err //nolint:wrapcheck // the caller names the field
		}
		var v string
		if err := dec.Decode(&v); err != nil {
			return nil, err //nolint:wrapcheck // the caller names the field
		}
		values = append(values, v)
	}
	return values, nil
}

// SlipStream: the mam_id is the pasted session's cookie, so it isn't a
// setting here.
type myAnonamouseSettings struct {
	SearchType          myAnonamouseSearchType       `port:"searchType" label:"Search type" default:"0" help:"MyAnonamouse searches only Releases of this type."`
	SearchInDescription bool                         `port:"searchInDescription" label:"Search in descriptions" help:"Also match the search text in each Release's description."`
	SearchInSeries      bool                         `port:"searchInSeries" label:"Search in series" help:"Also match the search text in series names."`
	SearchInFilenames   bool                         `port:"searchInFilenames" label:"Search in file names" help:"Also match the search text in the names of each Release's files."`
	SearchLanguages     []myAnonamouseSearchLanguage `port:"searchLanguages" label:"Search languages" help:"Search only Releases in these languages. With none chosen, every language is searched."`
	UseFreeleechWedge   bool                         `port:"useFreeleechWedge" label:"Use freeleech wedges" help:"Spend a freeleech wedge on each Grab that isn't freeleech. The wedge makes the torrent freeleech for your account."`
}

type myAnonamouseSearchType int

const (
	searchAll myAnonamouseSearchType = iota
	searchActive
	searchFreeleech
	searchFreeleechOrVip
	searchVip
	searchNotVip
)

// SlipStream: an option has no hint, so a label says what the C# hint
// adds.
func (myAnonamouseSearchType) Options() []port.Option {
	return []port.Option{
		{Value: "0", Label: "All Releases"},
		{Value: "1", Label: "Only active, with a seeder at the last update"},
		{Value: "2", Label: "Freeleech"},
		{Value: "3", Label: "Freeleech or VIP"},
		{Value: "4", Label: "VIP"},
		{Value: "5", Label: "Not VIP"},
	}
}

type myAnonamouseSearchLanguage int

func (myAnonamouseSearchLanguage) Options() []port.Option {
	return []port.Option{
		{Value: "1", Label: "English"},
		{Value: "17", Label: "Afrikaans"},
		{Value: "32", Label: "Arabic"},
		{Value: "35", Label: "Bengali"},
		{Value: "51", Label: "Bosnian"},
		{Value: "18", Label: "Bulgarian"},
		{Value: "6", Label: "Burmese"},
		{Value: "44", Label: "Cantonese"},
		{Value: "19", Label: "Catalan"},
		{Value: "2", Label: "Chinese"},
		{Value: "49", Label: "Croatian"},
		{Value: "20", Label: "Czech"},
		{Value: "21", Label: "Danish"},
		{Value: "22", Label: "Dutch"},
		{Value: "61", Label: "Estonian"},
		{Value: "39", Label: "Farsi"},
		{Value: "23", Label: "Finnish"},
		{Value: "36", Label: "French"},
		{Value: "37", Label: "German"},
		{Value: "26", Label: "Greek"},
		{Value: "59", Label: "Greek, Ancient"},
		{Value: "3", Label: "Gujarati"},
		{Value: "27", Label: "Hebrew"},
		{Value: "8", Label: "Hindi"},
		{Value: "28", Label: "Hungarian"},
		{Value: "63", Label: "Icelandic"},
		{Value: "53", Label: "Indonesian"},
		{Value: "56", Label: "Irish"},
		{Value: "43", Label: "Italian"},
		{Value: "38", Label: "Japanese"},
		{Value: "12", Label: "Javanese"},
		{Value: "5", Label: "Kannada"},
		{Value: "41", Label: "Korean"},
		{Value: "50", Label: "Lithuanian"},
		{Value: "46", Label: "Latin"},
		{Value: "62", Label: "Latvian"},
		{Value: "33", Label: "Malay"},
		{Value: "58", Label: "Malayalam"},
		{Value: "57", Label: "Manx"},
		{Value: "9", Label: "Marathi"},
		{Value: "48", Label: "Norwegian"},
		{Value: "45", Label: "Polish"},
		{Value: "34", Label: "Portuguese"},
		{Value: "52", Label: "Brazilian Portuguese (BP)"},
		{Value: "14", Label: "Punjabi"},
		{Value: "30", Label: "Romanian"},
		{Value: "16", Label: "Russian"},
		{Value: "24", Label: "Scottish Gaelic"},
		{Value: "60", Label: "Sanskrit"},
		{Value: "31", Label: "Serbian"},
		{Value: "54", Label: "Slovenian"},
		{Value: "4", Label: "Spanish"},
		{Value: "55", Label: "Castilian Spanish"},
		{Value: "40", Label: "Swedish"},
		{Value: "29", Label: "Tagalog"},
		{Value: "11", Label: "Tamil"},
		{Value: "10", Label: "Telugu"},
		{Value: "7", Label: "Thai"},
		{Value: "42", Label: "Turkish"},
		{Value: "25", Label: "Ukrainian"},
		{Value: "15", Label: "Urdu"},
		{Value: "13", Label: "Vietnamese"},
		{Value: "47", Label: "Other"},
	}
}

// stjBool is a bool as STJson's BooleanConverter reads it: true, false,
// 1 or 0.
type stjBool bool

func (b *stjBool) UnmarshalJSON(raw []byte) error {
	switch string(raw) {
	case "true", "1":
		*b = true
	case "false", "0":
		*b = false
	default:
		return fmt.Errorf("%s isn't true, false, 1 or 0", raw)
	}
	return nil
}

// stjInt is a number as STJson reads it with AllowReadingFromString: a
// JSON number or a string that holds one.
type stjInt int64

func (n *stjInt) UnmarshalJSON(raw []byte) error {
	v, err := strconv.ParseInt(strings.Trim(string(raw), `"`), 10, 64)
	if err != nil {
		return fmt.Errorf("%s isn't a whole number", raw)
	}
	*n = stjInt(v)
	return nil
}

type myAnonamouseTorrent struct {
	ID                stjInt  `json:"id"`
	Title             string  `json:"title"`
	AuthorInfo        string  `json:"author_info"`
	Description       string  `json:"description"`
	LanguageCode      string  `json:"lang_code"`
	Filetype          string  `json:"filetype"`
	Vip               stjBool `json:"vip"`
	Free              stjBool `json:"free"`
	PersonalFreeLeech stjBool `json:"personal_freeleech"`
	FreeVip           stjBool `json:"fl_vip"`
	Category          stjInt  `json:"category"`
	Added             string  `json:"added"`
	Grabs             stjInt  `json:"times_completed"`
	Seeders           stjInt  `json:"seeders"`
	Leechers          stjInt  `json:"leechers"`
	NumFiles          stjInt  `json:"numfiles"`
	Size              string  `json:"size"`
}

type myAnonamouseResponse struct {
	Error   string                `json:"error"`
	Data    []myAnonamouseTorrent `json:"data"`
	Message string                `json:"message"`
}

type myAnonamouseUserDataResponse struct {
	UserClass string `json:"classname"`
}
