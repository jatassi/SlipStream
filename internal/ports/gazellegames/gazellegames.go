// Ported from Prowlarr's src/NzbDrone.Core/Indexers/Definitions/GazelleGames.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

package gazellegames

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"github.com/jatassi/slipstream/internal/newznab"
	"github.com/jatassi/slipstream/internal/ports/port"
	"github.com/jatassi/slipstream/internal/prowlarr/datetimeutil"
	"github.com/jatassi/slipstream/internal/prowlarr/serializer"
	"github.com/jatassi/slipstream/internal/prowlarr/stringutil"
)

//go:embed upstream.json
var upstream []byte

// Port is GazelleGames.
//
// SlipStream: Prowlarr fetches the passkey in Test and keeps it on the
// settings. SlipStream reads it with Derive, once per revision, before any
// search, test or download.
var Port = port.Define(port.Entry{
	ID:          "gazellegames",
	Name:        "GazelleGames",
	Description: "GazelleGames is a private Tracker for games.",
	Language:    "en-US",
	Privacy:     port.Private,
	IndexerURLs: []string{"https://gazellegames.net/"},
	Capabilities: port.Capabilities{
		Categories: port.Categories(
			// Apple
			port.Map("Mac", newznab.ConsoleOther, "Mac"),
			port.Map("iOS", newznab.PCMobileiOS, "iOS"),
			port.Map("Apple Bandai Pippin", newznab.ConsoleOther, "Apple Bandai Pippin"),
			port.Map("Apple II", newznab.ConsoleOther, "Apple II"),

			// Google
			port.Map("Android", newznab.PCMobileAndroid, "Android"),

			// Microsoft
			port.Map("DOS", newznab.PCGames, "DOS"),
			port.Map("Windows", newznab.PCGames, "Windows"),
			port.Map("Xbox", newznab.ConsoleXBox, "Xbox"),
			port.Map("Xbox 360", newznab.ConsoleXBox360, "Xbox 360"),

			// Nintendo
			port.Map("Game Boy", newznab.ConsoleOther, "Game Boy"),
			port.Map("Game Boy Advance", newznab.ConsoleOther, "Game Boy Advance"),
			port.Map("Game Boy Color", newznab.ConsoleOther, "Game Boy Color"),
			port.Map("NES", newznab.ConsoleOther, "NES"),
			port.Map("Nintendo 64", newznab.ConsoleOther, "Nintendo 64"),
			port.Map("Nintendo 3DS", newznab.ConsoleOther, "Nintendo 3DS"),
			port.Map("New Nintendo 3DS", newznab.ConsoleOther, "New Nintendo 3DS"),
			port.Map("Nintendo DS", newznab.ConsoleNDS, "Nintendo DS"),
			port.Map("Nintendo GameCube", newznab.ConsoleOther, "Nintendo GameCube"),
			port.Map("Pokemon Mini", newznab.ConsoleOther, "Pokemon Mini"),
			port.Map("SNES", newznab.ConsoleOther, "SNES"),
			port.Map("Switch", newznab.ConsoleOther, "Switch"),
			port.Map("Virtual Boy", newznab.ConsoleOther, "Virtual Boy"),
			port.Map("Wii", newznab.ConsoleWii, "Wii"),
			port.Map("Wii U", newznab.ConsoleWiiU, "Wii U"),

			// Sony
			port.Map("PlayStation 1", newznab.ConsoleOther, "PlayStation 1"),
			port.Map("PlayStation 2", newznab.ConsoleOther, "PlayStation 2"),
			port.Map("PlayStation 3", newznab.ConsolePS3, "PlayStation 3"),
			port.Map("PlayStation 4", newznab.ConsolePS4, "PlayStation 4"),
			port.Map("PlayStation Portable", newznab.ConsolePSP, "PlayStation Portable"),
			port.Map("PlayStation Vita", newznab.ConsolePSVita, "PlayStation Vita"),

			// Sega
			port.Map("Dreamcast", newznab.ConsoleOther, "Dreamcast"),
			port.Map("Game Gear", newznab.ConsoleOther, "Game Gear"),
			port.Map("Master System", newznab.ConsoleOther, "Master System"),
			port.Map("Mega Drive", newznab.ConsoleOther, "Mega Drive"),
			port.Map("Pico", newznab.ConsoleOther, "Pico"),
			port.Map("Saturn", newznab.ConsoleOther, "Saturn"),
			port.Map("SG-1000", newznab.ConsoleOther, "SG-1000"),

			// Atari
			port.Map("Atari 2600", newznab.ConsoleOther, "Atari 2600"),
			port.Map("Atari 5200", newznab.ConsoleOther, "Atari 5200"),
			port.Map("Atari 7800", newznab.ConsoleOther, "Atari 7800"),
			port.Map("Atari Jaguar", newznab.ConsoleOther, "Atari Jaguar"),
			port.Map("Atari Lynx", newznab.ConsoleOther, "Atari Lynx"),
			port.Map("Atari ST", newznab.ConsoleOther, "Atari ST"),

			// Amstrad
			port.Map("Amstrad CPC", newznab.ConsoleOther, "Amstrad CPC"),

			// Bandai
			port.Map("Bandai WonderSwan", newznab.ConsoleOther, "Bandai WonderSwan"),
			port.Map("Bandai WonderSwan Color", newznab.ConsoleOther, "Bandai WonderSwan Color"),

			// Commodore
			port.Map("Commodore 64", newznab.ConsoleOther, "Commodore 64"),
			port.Map("Commodore 128", newznab.ConsoleOther, "Commodore 128"),
			port.Map("Commodore Amiga", newznab.ConsoleOther, "Commodore Amiga"),
			port.Map("Amiga CD32", newznab.ConsoleOther, "Amiga CD32"),
			port.Map("Commodore Plus-4", newznab.ConsoleOther, "Commodore Plus-4"),
			port.Map("Commodore VIC-20", newznab.ConsoleOther, "Commodore VIC-20"),

			// NEC
			port.Map("NEC PC-98", newznab.ConsoleOther, "NEC PC-98"),
			port.Map("NEC PC-FX", newznab.ConsoleOther, "NEC PC-FX"),
			port.Map("NEC SuperGrafx", newznab.ConsoleOther, "NEC SuperGrafx"),
			port.Map("NEC TurboGrafx-16", newznab.ConsoleOther, "NEC TurboGrafx-16"),

			// Sinclair
			port.Map("ZX Spectrum", newznab.ConsoleOther, "ZX Spectrum"),

			// Spectravideo
			port.Map("MSX", newznab.ConsoleOther, "MSX"),
			port.Map("MSX 2", newznab.ConsoleOther, "MSX 2"),

			// Tiger
			port.Map("Game.com", newznab.ConsoleOther, "Game.com"),
			port.Map("Gizmondo", newznab.ConsoleOther, "Gizmondo"),

			// VTech
			port.Map("V.Smile", newznab.ConsoleOther, "V.Smile"),
			port.Map("CreatiVision", newznab.ConsoleOther, "CreatiVision"),

			// Tabletop Games
			port.Map("Board Game", newznab.ConsoleOther, "Board Game"),
			port.Map("Card Game", newznab.ConsoleOther, "Card Game"),
			port.Map("Miniature Wargames", newznab.ConsoleOther, "Miniature Wargames"),
			port.Map("Pen and Paper RPG", newznab.ConsoleOther, "Pen and Paper RPG"),

			// Other
			port.Map("3DO", newznab.ConsoleOther, "3DO"),
			port.Map("Casio Loopy", newznab.ConsoleOther, "Casio Loopy"),
			port.Map("Casio PV-1000", newznab.ConsoleOther, "Casio PV-1000"),
			port.Map("Colecovision", newznab.ConsoleOther, "Colecovision"),
			port.Map("Emerson Arcadia 2001", newznab.ConsoleOther, "Emerson Arcadia 2001"),
			port.Map("Entex Adventure Vision", newznab.ConsoleOther, "Entex Adventure Vision"),
			port.Map("Epoch Super Casette Vision", newznab.ConsoleOther, "Epoch Super Casette Vision"), //nolint:misspell // the platform name GazelleGames sends
			port.Map("Fairchild Channel F", newznab.ConsoleOther, "Fairchild Channel F"),
			port.Map("Funtech Super Acan", newznab.ConsoleOther, "Funtech Super Acan"),
			port.Map("GamePark GP32", newznab.ConsoleOther, "GamePark GP32"),
			port.Map("General Computer Vectrex", newznab.ConsoleOther, "General Computer Vectrex"),
			port.Map("Interactive DVD", newznab.ConsoleOther, "Interactive DVD"),
			port.Map("Linux", newznab.ConsoleOther, "Linux"),
			port.Map("Hartung Game Master", newznab.ConsoleOther, "Hartung Game Master"),
			port.Map("Magnavox-Phillips Odyssey", newznab.ConsoleOther, "Magnavox-Phillips Odyssey"),
			port.Map("Mattel Intellivision", newznab.ConsoleOther, "Mattel Intellivision"),
			port.Map("Memotech MTX", newznab.ConsoleOther, "Memotech MTX"),
			port.Map("Miles Gordon Sam Coupe", newznab.ConsoleOther, "Miles Gordon Sam Coupe"),
			port.Map("Nokia N-Gage", newznab.ConsoleOther, "Nokia N-Gage"),
			port.Map("Oculus Quest", newznab.ConsoleOther, "Oculus Quest"),
			port.Map("Ouya", newznab.ConsoleOther, "Ouya"),
			port.Map("Philips Videopac+", newznab.ConsoleOther, "Philips Videopac+"),
			port.Map("Philips CD-i", newznab.ConsoleOther, "Philips CD-i"),
			port.Map("Phone/PDA", newznab.ConsoleOther, "Phone/PDA"),
			port.Map("RCA Studio II", newznab.ConsoleOther, "RCA Studio II"),
			port.Map("Sharp X1", newznab.ConsoleOther, "Sharp X1"),
			port.Map("Sharp X68000", newznab.ConsoleOther, "Sharp X68000"),
			port.Map("SNK Neo Geo", newznab.ConsoleOther, "SNK Neo Geo"),
			port.Map("SNK Neo Geo Pocket", newznab.ConsoleOther, "SNK Neo Geo Pocket"),
			port.Map("Taito Type X", newznab.ConsoleOther, "Taito Type X"),
			port.Map("Tandy Color Computer", newznab.ConsoleOther, "Tandy Color Computer"),
			port.Map("Tangerine Oric", newznab.ConsoleOther, "Tangerine Oric"),
			port.Map("Thomson MO5", newznab.ConsoleOther, "Thomson MO5"),
			port.Map("Watara Supervision", newznab.ConsoleOther, "Watara Supervision"),
			port.Map("Retro - Other", newznab.ConsoleOther, "Retro - Other"),

			// special categories (real categories/not platforms)
			port.Map(1, newznab.PCGames, "Games"),
			port.Map(2, newznab.PC0day, "Applications"),
			port.Map(3, newznab.BooksEBook, "E-Books"),
			port.Map(4, newznab.AudioOther, "OST"),
		),
	},
	Upstream: port.ReadUpstream(upstream),
}, port.Impl[gazelleGamesSettings]{
	Login:  port.NoLogin{},
	Basic:  getBasicSearchRequests,
	Parse:  parseResponse,
	Derive: fetchPasskey,
})

type indexer = port.Indexer[gazelleGamesSettings]

// SlipStream: the message is SlipStream's own, per WRITING.md, and the
// passkey reaches the parser as ix.Derived["passkey"].
func fetchPasskey(ix *indexer) (map[string]string, error) {
	request := port.Get(ix.URL("api.php")).
		Accept("application/json").
		SetHeader("X-API-Key", ix.Settings.APIKey).
		AddQueryParam("request", "quick_user")

	indexResponse, err := ix.Fetch(request)
	if err != nil {
		return nil, err
	}

	var index *gazelleGamesUserResponse
	if err := decodeJSON(indexResponse.Body, &index); err != nil {
		return nil, port.IndexerError("read the Indexer's user response: %v", err)
	}

	if index == nil ||
		strings.TrimSpace(index.Status) == "" ||
		index.Status != "success" ||
		index.Response == nil ||
		strings.TrimSpace(index.Response.PassKey) == "" {
		return nil, port.IndexerAuthError("")
	}

	return map[string]string{"passkey": index.Response.PassKey}, nil
}

// SlipStream: the Movie, Music, TV and Book overloads are gone. Prowlarr's
// capabilities declare no params for those modes, and SlipStream has no
// Music or Book mode.
func getBasicSearchRequests(ix *indexer, searchCriteria port.SearchCriteria) (port.Chain, error) {
	return port.One(getRequest(ix, getBasicSearchParameters(ix, searchCriteria.SanitizedSearchTerm(), searchCriteria)))
}

func getRequest(ix *indexer, parameters stringutil.KeyValuePairs) *port.Request {
	return requestBuilder(ix, "api.php?"+parameters.GetQueryString())
}

func requestBuilder(ix *indexer, resource string) *port.Request {
	return port.Get(ix.URL(resource)).
		Accept("application/json").
		SetHeader("X-API-Key", ix.Settings.APIKey)
}

// SlipStream: MinSize and MaxSize are gone, since a SlipStream search
// carries no size bounds.
func getBasicSearchParameters(ix *indexer, searchTerm string, searchCriteria port.SearchCriteria) stringutil.KeyValuePairs {
	parameters := stringutil.KeyValuePairs{
		{Key: "request", Value: "search"},
		{Key: "search_type", Value: "torrents"},
		{Key: "empty_groups", Value: "filled"},
		{Key: "order_by", Value: "time"},
		{Key: "order_way", Value: "desc"},
	}

	if strings.TrimSpace(searchTerm) != "" {
		name := "searchstr"
		if ix.Settings.SearchGroupNames {
			name = "groupname"
		}
		parameters.Add(name, strings.ReplaceAll(searchTerm, ".", " "))
	}

	if searchCriteria.Categories != nil {
		for _, category := range ix.Categories.TrackerIDs(searchCriteria.Categories) {
			if !isAllDigits(category) {
				parameters.Add("artistcheck[]", category)
			}
		}
	}

	if ix.Settings.FreeleechOnly {
		parameters.Add("freetorrent", "1")
	}

	return parameters
}

func isAllDigits(s string) bool {
	return !strings.ContainsFunc(s, func(r rune) bool { return !unicode.IsDigit(r) })
}

var yearRegex = regexp.MustCompile(`(?:^|[^\p{L}\p{Mn}\p{Nd}\p{Pc}])(?:19|20|21)\p{Nd}{2}(?:[^\p{L}\p{Mn}\p{Nd}\p{Pc}]|$)`)

// SlipStream: the messages are SlipStream's own, per WRITING.md. Prowlarr
// reads no refusal from a search; a 401 is a key GazelleGames refused,
// which SlipStream names account_refused.
func parseResponse(ix *indexer, indexerResponse *port.IndexerResponse) ([]port.TorrentInfo, error) {
	var torrentInfos []port.TorrentInfo

	if indexerResponse.StatusCode == http.StatusUnauthorized {
		return nil, port.IndexerAuthError("")
	}

	if indexerResponse.StatusCode != http.StatusOK {
		return nil, port.IndexerError("the Indexer responded with status %d", indexerResponse.StatusCode)
	}

	if contentType := indexerResponse.Headers.Get("Content-Type"); !strings.Contains(contentType, "application/json") {
		return nil, port.IndexerError("the Indexer responded with %q content, not JSON", contentType)
	}

	var jsonResponse gazelleGamesResponse
	if err := decodeJSON(indexerResponse.Body, &jsonResponse); err != nil {
		return nil, port.IndexerError("read the Indexer's search response: %v", err)
	}

	response, isObject, err := orderedObject(jsonResponse.Response)
	if err != nil {
		return nil, port.IndexerError("read the Indexer's search response: %v", err)
	}

	if jsonResponse.Status != "success" ||
		strings.TrimSpace(string(jsonResponse.Status)) == "" ||
		!isObject {
		return torrentInfos, nil
	}

	for _, group := range response {
		groupID, err := strconv.Atoi(group.Key)
		if err != nil {
			return nil, port.IndexerError("read the Indexer's search response: group %q has no number", group.Key)
		}
		var groupValue gazelleGamesGroup
		if err := decodeJSON(group.Value, &groupValue); err != nil {
			return nil, port.IndexerError("read the Indexer's search response: %v", err)
		}

		groupTorrents, isObject, err := orderedObject(groupValue.Torrents)
		if err != nil {
			return nil, port.IndexerError("read the Indexer's search response: %v", err)
		}
		if !isObject {
			continue
		}

		type keyedTorrent struct {
			Key   int
			Value gazelleGamesTorrent
		}
		var torrents []keyedTorrent
		for _, t := range groupTorrents {
			id, err := strconv.Atoi(t.Key)
			if err != nil {
				return nil, port.IndexerError("read the Indexer's search response: torrent %q has no number", t.Key)
			}
			var value gazelleGamesTorrent
			if err := decodeJSON(t.Value, &value); err != nil {
				return nil, port.IndexerError("read the Indexer's search response: %v", err)
			}
			if strings.EqualFold(string(value.TorrentType), "TORRENT") {
				torrents = append(torrents, keyedTorrent{id, value})
			}
		}

		var categories []newznab.Category
		for _, a := range groupValue.Artists {
			for _, c := range ix.Categories.FromDesc(string(a.Name)) {
				if !slices.Contains(categories, c) {
					categories = append(categories, c)
				}
			}
		}

		for _, torrent := range torrents {
			freeTorrent := tryParseFreeTorrent(torrent.Value.FreeTorrent)
			downloadVolumeFactor := 1.0
			if freeTorrent == freeLeech || freeTorrent == neutral || bool(torrent.Value.LowSeedFL) {
				downloadVolumeFactor = 0
			}

			// Skip non-freeleech results when freeleech only is set
			if ix.Settings.FreeleechOnly && downloadVolumeFactor != 0.0 {
				continue
			}

			torrentID := torrent.Key
			infoURL := getInfoURL(ix, groupID, torrentID)

			if len(categories) == 0 {
				categories = ix.Categories.FromTracker(strconv.Itoa(int(torrent.Value.CategoryID)))
			}

			size, err := strconv.ParseInt(string(torrent.Value.Size), 10, 64)
			if err != nil {
				return nil, port.IndexerError("read the Indexer's search response: size %q isn't a whole number", torrent.Value.Size)
			}

			publishDate, err := datetimeutil.Parse(string(torrent.Value.Time), datetimeutil.AssumeUniversal, ix.Clock)
			if err != nil {
				return nil, port.IndexerError("read the Indexer's search response: %v", err)
			}

			uploadVolumeFactor := 1.0
			if freeTorrent == neutral {
				uploadVolumeFactor = 0
			}

			grabs := 0
			if torrent.Value.Snatched != nil {
				grabs = int(*torrent.Value.Snatched)
			}

			var flags []port.SiteFlag
			if torrent.Value.Scene == 1 {
				flags = append(flags, port.Scene)
			}

			release := port.TorrentInfo{
				GUID:                 infoURL,
				InfoURL:              infoURL,
				DownloadURL:          getDownloadURL(ix, torrentID),
				Title:                getTitle(groupValue, torrent.Value),
				Categories:           categories,
				Files:                int(torrent.Value.FileCount),
				Size:                 size,
				Grabs:                grabs,
				Seeders:              int(torrent.Value.Seeders),
				Peers:                int(torrent.Value.Leechers + torrent.Value.Seeders),
				PublishDate:          publishDate,
				IndexerFlags:         flags,
				DownloadVolumeFactor: downloadVolumeFactor,
				UploadVolumeFactor:   uploadVolumeFactor,
				MinimumSeedTime:      288000, // Minimum of 3 days and 8 hours (80 hours in total)
			}

			torrentInfos = append(torrentInfos, release)
		}
	}

	// order by date
	slices.SortStableFunc(torrentInfos, func(a, b port.TorrentInfo) int { return b.PublishDate.Compare(a.PublishDate) })

	return torrentInfos, nil
}

func getTitle(group gazelleGamesGroup, torrent gazelleGamesTorrent) string {
	title := html.UnescapeString(string(torrent.ReleaseTitle))

	if group.Year != nil && *group.Year > 0 && strings.TrimSpace(title) != "" && !yearRegex.MatchString(title) {
		title += fmt.Sprintf(" (%d)", *group.Year)
	}

	if strings.TrimSpace(string(torrent.RemasterTitle)) != "" {
		title += " [" + strings.TrimSpace(html.UnescapeString(string(torrent.RemasterTitle))+" "+string(torrent.RemasterYear)) + "]"
	}

	flags := []string{strings.TrimSpace(string(torrent.Format + " " + torrent.Encoding))}

	if len(group.Artists) > 0 {
		names := make([]string, 0, len(group.Artists))
		for _, a := range group.Artists {
			names = append(names, string(a.Name))
		}
		flags = append(flags, strings.Join(names, ", "))
	}

	flags = append(flags, string(torrent.Language), string(torrent.Region), string(torrent.Miscellaneous))

	if torrent.Dupable == 1 {
		flags = append(flags, "Trumpable")
	}

	flags = slices.DeleteFunc(flags, func(x string) bool { return strings.TrimSpace(x) == "" })

	if len(flags) > 0 {
		title += " [" + strings.Join(flags, " / ") + "]"
	}

	if strings.TrimSpace(string(torrent.GameDoxType)) != "" {
		title += " [" + strings.TrimSpace(string(torrent.GameDoxType)) + "]"
	}

	return title
}

// AuthKey is required but not checked, just pass in a dummy variable
// to avoid having to track authkey, which is randomly cycled
func getDownloadURL(ix *indexer, torrentID int) string {
	return port.NewURI(ix.BaseURL).
		CombinePath("/torrents.php").
		AddQueryParam("action", "download").
		AddQueryParam("id", strconv.Itoa(torrentID)).
		AddQueryParam("authkey", "prowlarr").
		AddQueryParam("torrent_pass", ix.Derived["passkey"]).
		FullURI()
}

func getInfoURL(ix *indexer, groupID, torrentID int) string {
	return port.NewURI(ix.BaseURL).
		CombinePath("/torrents.php").
		AddQueryParam("id", strconv.Itoa(groupID)).
		AddQueryParam("torrentid", strconv.Itoa(torrentID)).
		FullURI()
}

// SlipStream: Prowlarr's validator is the required flag on the API key.
// Passkey is gone: Derive reads it.
type gazelleGamesSettings struct {
	port.APIKeyField `help:"Find your API key in your GazelleGames settings, under Access Settings. It needs the User and Torrents permissions."`
	SearchGroupNames bool `port:"searchGroupNames" label:"Search group names" help:"Search by group name."`
	FreeleechOnly    bool `port:"freeleechOnly" label:"Freeleech only" help:"Search only freeleech Releases."`
}

type gazelleGamesResponse struct {
	Status   serializer.String `json:"status"`
	Response json.RawMessage   `json:"response"`
}

type gazelleGamesGroup struct {
	Artists  []gazelleGamesArtist `json:"artists"`
	Torrents json.RawMessage      `json:"torrents"`
	Year     *serializer.Int      `json:"year"`
}

type gazelleGamesArtist struct {
	ID   serializer.String `json:"id"`
	Name serializer.String `json:"name"`
}

type gazelleGamesTorrent struct {
	CategoryID    serializer.Int    `json:"categoryId"`
	Format        serializer.String `json:"format"`
	Encoding      serializer.String `json:"encoding"`
	Language      serializer.String `json:"language"`
	Region        serializer.String `json:"region"`
	RemasterYear  serializer.String `json:"remasterYear"`
	RemasterTitle serializer.String `json:"remasterTitle"`
	ReleaseTitle  serializer.String `json:"releaseTitle"`
	Miscellaneous serializer.String `json:"miscellaneous"`
	Scene         serializer.Int    `json:"scene"`
	Dupable       serializer.Int    `json:"dupable"`
	Time          serializer.String `json:"time"`
	TorrentType   serializer.String `json:"torrentType"`
	FileCount     serializer.Int    `json:"fileCount"`
	Size          serializer.String `json:"size"`
	Snatched      *serializer.Int   `json:"snatched"`
	Seeders       serializer.Int    `json:"seeders"`
	Leechers      serializer.Int    `json:"leechers"`
	FreeTorrent   serializer.String `json:"freeTorrent"`
	PersonalFL    serializer.Bool   `json:"personalFL"`
	LowSeedFL     serializer.Bool   `json:"lowSeedFL"`
	GameDoxType   serializer.String `json:"GameDOXType"`
}

type gazelleGamesUserResponse struct {
	Status   string            `json:"status"`
	Response *gazelleGamesUser `json:"response"`
}

type gazelleGamesUser struct {
	PassKey string `json:"passKey"`
}

type gazelleGamesFreeTorrent int

const (
	normal    gazelleGamesFreeTorrent = 0
	freeLeech gazelleGamesFreeTorrent = 1
	neutral   gazelleGamesFreeTorrent = 2
	either    gazelleGamesFreeTorrent = 3
)

var freeTorrentNames = map[string]gazelleGamesFreeTorrent{"normal": normal, "freeleech": freeLeech, "neutral": neutral, "either": either}

// tryParseFreeTorrent is Enum.TryParse(value, true, out freeTorrent): a
// name in any case or a number, or names and numbers joined by commas,
// OR'd. Anything else is Normal.
func tryParseFreeTorrent(value serializer.String) gazelleGamesFreeTorrent {
	var parsed gazelleGamesFreeTorrent
	for part := range strings.SplitSeq(string(value), ",") {
		part = strings.TrimSpace(part)
		if v, ok := freeTorrentNames[strings.ToLower(part)]; ok {
			parsed |= v
			continue
		}
		n, err := strconv.Atoi(part)
		if err != nil {
			return normal
		}
		parsed |= gazelleGamesFreeTorrent(n)
	}
	return parsed
}

var errNotObject = errors.New("not an object")

// keyed is one member of a JSON object.
type keyed struct {
	Key   string
	Value json.RawMessage
}

// orderedObject is a JObject's members in the order the site sent them,
// which Newtonsoft's Dictionary keeps. isObject is false for any other
// value.
func orderedObject(raw json.RawMessage) (members []keyed, isObject bool, err error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || raw[0] != '{' {
		return nil, false, nil
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	if _, err := dec.Token(); err != nil {
		return nil, false, fmt.Errorf("read an object: %w", err)
	}
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return nil, false, fmt.Errorf("read an object: %w", err)
		}
		key, ok := tok.(string)
		if !ok {
			return nil, false, errNotObject
		}
		var value json.RawMessage
		if err := dec.Decode(&value); err != nil {
			return nil, false, fmt.Errorf("read an object: %w", err)
		}
		members = append(members, keyed{key, value})
	}
	return members, true, nil
}

// decodeJSON is Newtonsoft's read: a leading byte order mark is
// whitespace to it.
func decodeJSON(body []byte, v any) error {
	if err := json.Unmarshal(bytes.TrimPrefix(body, []byte("\xef\xbb\xbf")), v); err != nil {
		return fmt.Errorf("read JSON: %w", err)
	}
	return nil
}
