// Package rss reads a torrent site's RSS feed. When a feed is added, Check
// finds the rules that read its items (where the download link is, and
// whether the size and seeders are in the description) or the Refusal
// that keeps it out. Read reads a feed's items by those rules. gofeed
// parses the XML.
package rss

import (
	"bytes"
	"cmp"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/mmcdole/gofeed"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/magnet"
)

// Link is where an item's download link is.
type Link string

const (
	// Enclosure is the enclosure's url.
	Enclosure Link = "enclosure"
	// Torrent is the torrent namespace's magnetURI.
	Torrent Link = "torrent"
	// Item is the item's own link.
	Item Link = "item"
)

// Reading is how a feed's items are read.
type Reading struct {
	Link                   Link `json:"link"`
	SizeFromDescription    bool `json:"sizeFromDescription,omitempty"`
	SeedersFromDescription bool `json:"seedersFromDescription,omitempty"`
	// AllowNoSize keeps an item with no size, which is the Admin's
	// choice rather than something the feed shows.
	AllowNoSize bool `json:"-"`
}

// Refusal is why a feed can't be used.
type Refusal string

const (
	NoItems      Refusal = "no_items"
	NoTitle      Refusal = "no_title"
	NoLink       Refusal = "no_link"
	RepeatedGUID Refusal = "repeated_guid"
	NoSize       Refusal = "no_size"
)

var refusalWords = map[Refusal]string{
	NoItems:      "the feed has no items",
	NoTitle:      "some items have no title",
	NoLink:       "some items have no download link",
	RepeatedGUID: "some items repeat the same GUID",
	NoSize:       "some items have no size",
}

func (r Refusal) Error() string { return refusalWords[r] }

// Entry is one item, read.
type Entry struct {
	// GUID tells items apart: the item's guid, else its download link.
	GUID  string
	Title string
	// Link is the .torrent link, empty when the item offers a magnet link.
	Link   string
	Magnet string
	// InfoHash is the torrent namespace's infoHash, else the magnet link's.
	InfoHash  *bencode.InfoHash
	Size      int64
	Seeders   int
	Published time.Time
	// Details is the item's page on the site.
	Details string
}

// links is where Check looks for the download link, in order.
var links = []Link{Enclosure, Torrent, Item}

// Check finds how body's items are read: the first place every item has
// a download link, and the description for a size or seeders the items
// don't carry as elements. It refuses a feed with no items, an item with
// no title or no download link, a repeated GUID, and, unless allowNoSize,
// an item with no size.
func Check(body []byte, allowNoSize bool) (Reading, int, error) {
	items, err := parse(body)
	if err != nil {
		return Reading{}, 0, err
	}
	if len(items) == 0 {
		return Reading{}, 0, NoItems
	}
	r := Reading{AllowNoSize: allowNoSize}
	for _, l := range links {
		if every(items, func(it *gofeed.Item) bool { return linkOf(it, l) != "" }) {
			r.Link = l
			break
		}
	}
	switch {
	case !every(items, func(it *gofeed.Item) bool { return strings.TrimSpace(it.Title) != "" }):
		return Reading{}, 0, NoTitle
	case r.Link == "":
		return Reading{}, 0, NoLink
	}
	seen := map[string]bool{}
	for _, it := range items {
		guid := guidOf(it, r.Link)
		if seen[guid] {
			return Reading{}, 0, RepeatedGUID
		}
		seen[guid] = true
	}
	r.SizeFromDescription = !every(items, func(it *gofeed.Item) bool { return elementSize(it) > 0 }) &&
		every(items, func(it *gofeed.Item) bool { return elementSize(it) > 0 || describedSize(it) > 0 })
	if !allowNoSize && !every(items, func(it *gofeed.Item) bool { return sizeOf(it, r) > 0 }) {
		return Reading{}, 0, NoSize
	}
	r.SeedersFromDescription = !some(items, func(it *gofeed.Item) bool { return ext(it, "seeds") != "" }) &&
		some(items, func(it *gofeed.Item) bool { return describedSeeders(it) >= 0 })
	return r, len(items), nil
}

// Read reads body's items by r, leaving out an item r can't read: one
// with no title, no download link or, unless r allows it, no size, and a
// later item with the GUID of an earlier one.
func Read(body []byte, r Reading) ([]Entry, error) {
	items, err := parse(body)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var out []Entry
	for _, it := range items {
		e := Entry{
			GUID: guidOf(it, r.Link), Title: strings.TrimSpace(it.Title), Size: sizeOf(it, r), Details: it.Link,
			Seeders: int(number(ext(it, "seeds"))),
		}
		if r.SeedersFromDescription {
			e.Seeders = max(describedSeeders(it), 0)
		}
		if it.PublishedParsed != nil {
			e.Published = it.PublishedParsed.UTC()
		}
		if r.Link == Item {
			e.Details = ""
		}
		link := linkOf(it, r.Link)
		if magnet.Is(link) {
			e.Magnet = link
		} else {
			e.Link = link
		}
		e.InfoHash = infoHashOf(it, e.Magnet)
		if e.Title == "" || link == "" || seen[e.GUID] || (e.Size == 0 && !r.AllowNoSize) {
			continue
		}
		seen[e.GUID] = true
		out = append(out, e)
	}
	return out, nil
}

func parse(body []byte) ([]*gofeed.Item, error) {
	f, err := gofeed.NewParser().Parse(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("read the feed: %w", err)
	}
	return f.Items, nil
}

func linkOf(it *gofeed.Item, l Link) string {
	switch l {
	case Enclosure:
		for _, e := range it.Enclosures {
			if u := strings.TrimSpace(e.URL); u != "" {
				return u
			}
		}
	case Torrent:
		return ext(it, "magnetURI")
	case Item:
		return strings.TrimSpace(it.Link)
	}
	return ""
}

func guidOf(it *gofeed.Item, l Link) string {
	return cmp.Or(strings.TrimSpace(it.GUID), linkOf(it, l))
}

// ext is the value of the torrent namespace's element name.
func ext(it *gofeed.Item, name string) string {
	for _, e := range it.Extensions["torrent"][name] {
		if v := strings.TrimSpace(e.Value); v != "" {
			return v
		}
	}
	return ""
}

func sizeOf(it *gofeed.Item, r Reading) int64 {
	if n := elementSize(it); n > 0 {
		return n
	}
	if r.SizeFromDescription {
		return describedSize(it)
	}
	return 0
}

// elementSize is the size an element gives: the enclosure's length, the
// torrent namespace's contentLength or a size element.
func elementSize(it *gofeed.Item) int64 {
	var sizes []string
	for _, e := range it.Enclosures {
		sizes = append(sizes, e.Length)
	}
	for _, s := range append(sizes, ext(it, "contentLength"), it.Custom["size"]) {
		if n := number(s); n > 0 {
			return n
		}
	}
	return 0
}

var (
	sizeIn    = regexp.MustCompile(`(?i)\b(\d+(?:[.,]\d+)?)\s*([KMGT]i?B|B|bytes)\b`)
	seedersIn = regexp.MustCompile(`(?i)\bseed(?:s|ers?)\b\W{0,3}(\d+)|(\d+)\s*seed(?:s|ers?)\b`)
)

var units = map[string]float64{"B": 1, "BYTES": 1, "KB": 1 << 10, "KIB": 1 << 10, "MB": 1 << 20, "MIB": 1 << 20, "GB": 1 << 30, "GIB": 1 << 30, "TB": 1 << 40, "TIB": 1 << 40}

// describedSize is the first size the description names, such as
// "Size: 1.4 GB", counting a KB as 1024 bytes as the sites do.
func describedSize(it *gofeed.Item) int64 {
	m := sizeIn.FindStringSubmatch(it.Description)
	if m == nil {
		return 0
	}
	n, err := strconv.ParseFloat(strings.ReplaceAll(m[1], ",", "."), 64)
	if err != nil {
		return 0
	}
	return int64(math.Round(n * units[strings.ToUpper(m[2])]))
}

// describedSeeders is the seeders the description names, such as
// "Seeders: 12", -1 for none.
func describedSeeders(it *gofeed.Item) int {
	m := seedersIn.FindStringSubmatch(it.Description)
	if m == nil {
		return -1
	}
	n, err := strconv.Atoi(cmp.Or(m[1], m[2]))
	if err != nil {
		return -1
	}
	return n
}

func infoHashOf(it *gofeed.Item, magnetLink string) *bencode.InfoHash {
	if h, err := bencode.ParseInfoHash(strings.ToLower(ext(it, "infoHash"))); err == nil {
		return &h
	}
	if l, err := magnet.Parse(magnetLink); err == nil {
		return &l.InfoHash
	}
	return nil
}

func number(s string) int64 {
	n, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	if err != nil {
		return 0
	}
	return n
}

func every(items []*gofeed.Item, f func(*gofeed.Item) bool) bool {
	for _, it := range items {
		if !f(it) {
			return false
		}
	}
	return true
}

func some(items []*gofeed.Item, f func(*gofeed.Item) bool) bool {
	for _, it := range items {
		if f(it) {
			return true
		}
	}
	return false
}
