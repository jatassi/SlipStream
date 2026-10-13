package indexers

import (
	"encoding/xml"
	"fmt"
	"slices"
	"strconv"
	"strings"
)

// Caps are an Indexer's search modes, keyed by their t= value, each with
// the parameters it takes in the order the Indexer lists them.
type Caps struct {
	Modes map[string][]string `json:"modes"`
}

// Item is one release a search answered.
type Item struct {
	Title string
	Size  int64
	// Link is the .torrent's URL: the enclosure's, else the item's link.
	Link    string
	Seeders int64
	// InfoHash is the v1 info hash in lowercase hex, empty when the
	// Indexer sent none.
	InfoHash string
	// IDs are the Torznab id attributes the item carries, such as tmdbid.
	IDs map[string]string
}

// idAttrs are the torznab:attr names Item.IDs keeps.
var idAttrs = []string{"tmdbid", "imdbid"}

// RefusedError is a Torznab error element: the Indexer understood the
// request and refused it.
type RefusedError struct {
	Code        string
	Description string
}

func (e *RefusedError) Error() string {
	return fmt.Sprintf("the Indexer refused: error %s: %s", e.Code, e.Description)
}

type torznabError struct {
	XMLName     xml.Name
	Code        string `xml:"code,attr"`
	Description string `xml:"description,attr"`
}

// refusal is the RefusedError body holds, nil when body is no Torznab
// error.
func refusal(body []byte) *RefusedError {
	var e torznabError
	if xml.Unmarshal(body, &e) == nil && e.XMLName.Local == "error" {
		return &RefusedError{Code: e.Code, Description: e.Description}
	}
	return nil
}

type capsDocument struct {
	XMLName   xml.Name
	Searching struct {
		Modes []struct {
			XMLName         xml.Name
			Available       string `xml:"available,attr"`
			SupportedParams string `xml:"supportedParams,attr"`
		} `xml:",any"`
	} `xml:"searching"`
}

// ParseCaps reads a t=caps document's available search modes. A mode
// listing no parameters takes q, as Torznab defines.
func ParseCaps(body []byte) (Caps, error) {
	var doc capsDocument
	if err := xml.Unmarshal(body, &doc); err != nil {
		return Caps{}, fmt.Errorf("read the caps: %w", err)
	}
	if doc.XMLName.Local != "caps" {
		return Caps{}, fmt.Errorf("read the caps: the document is <%s>, not <caps>", doc.XMLName.Local)
	}
	c := Caps{Modes: map[string][]string{}}
	for _, m := range doc.Searching.Modes {
		name, ok := mode(m.XMLName.Local)
		if !ok || m.Available != "yes" {
			continue
		}
		params := []string{"q"}
		if m.SupportedParams != "" {
			params = strings.Split(m.SupportedParams, ",")
		}
		c.Modes[name] = params
	}
	return c, nil
}

// mode is the t= value of a caps search element: <x>-search is t=x, but
// for the generic search and tv-search, whose t= is tvsearch.
func mode(element string) (string, bool) {
	switch element {
	case "search":
		return "search", true
	case "tv-search":
		return "tvsearch", true
	}
	name, ok := strings.CutSuffix(element, "-search")
	return name, ok && name != ""
}

type feed struct {
	Items []struct {
		Title     string `xml:"title"`
		Size      int64  `xml:"size"`
		Link      string `xml:"link"`
		Enclosure struct {
			URL string `xml:"url,attr"`
		} `xml:"enclosure"`
		Attrs []struct {
			Name  string `xml:"name,attr"`
			Value string `xml:"value,attr"`
		} `xml:"attr"`
	} `xml:"channel>item"`
}

// ParseItems reads a search's RSS feed.
func ParseItems(body []byte) ([]Item, error) {
	var f feed
	if err := xml.Unmarshal(body, &f); err != nil {
		return nil, fmt.Errorf("read the results: %w", err)
	}
	items := make([]Item, len(f.Items))
	for i, it := range f.Items {
		items[i] = Item{Title: it.Title, Size: it.Size, Link: it.Enclosure.URL, IDs: map[string]string{}}
		if items[i].Link == "" {
			items[i].Link = it.Link
		}
		for _, a := range it.Attrs {
			switch {
			case a.Name == "seeders":
				n, err := strconv.ParseInt(a.Value, 10, 64)
				if err != nil {
					return nil, fmt.Errorf("read the results: item %q: seeders %q is not a number", it.Title, a.Value)
				}
				items[i].Seeders = n
			case a.Name == "infohash":
				items[i].InfoHash = strings.ToLower(a.Value)
			case slices.Contains(idAttrs, a.Name):
				items[i].IDs[a.Name] = a.Value
			}
		}
	}
	return items, nil
}

// Query is the Torznab query for terms under caps: terms' mode with the
// first id parameter caps list for that mode, else a free-text search.
// It chooses by declared capability, so it never retries.
func Query(c Caps, terms Terms) map[string]string {
	for _, p := range c.Modes[terms.Mode] {
		if id, ok := terms.IDs[p]; ok {
			return map[string]string{"t": terms.Mode, p: id}
		}
	}
	return map[string]string{"t": "search", "q": terms.Query}
}
