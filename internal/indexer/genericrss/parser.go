package genericrss

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/slipstream/slipstream/internal/indexer/types"
)

// ParseFeed auto-detects the feed format and parses it into ReleaseInfo items.
func ParseFeed(data []byte, indexerID int64, indexerName string) ([]types.TorrentInfo, error) {
	// Try standard RSS/Atom XML first
	if results, err := parseStandardRSS(data, indexerID, indexerName); err == nil && len(results) > 0 {
		return results, nil
	}

	// Try EzRSS (namespace-extended RSS)
	if results, err := parseEzRSS(data, indexerID, indexerName); err == nil && len(results) > 0 {
		return results, nil
	}

	// Try TorrentPotato (JSON)
	if results, err := parseTorrentPotato(data, indexerID, indexerName); err == nil && len(results) > 0 {
		return results, nil
	}

	return nil, fmt.Errorf("unable to parse feed: unrecognized format")
}

// Standard RSS structures

type rssFeed struct {
	XMLName xml.Name   `xml:"rss"`
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Title string    `xml:"title"`
	Items []rssItem `xml:"item"`
}

type rssItem struct {
	Title     string       `xml:"title"`
	Link      string       `xml:"link"`
	GUID      string       `xml:"guid"`
	PubDate   string       `xml:"pubDate"`
	Size      int64        `xml:"size"`
	Enclosure rssEnclosure `xml:"enclosure"`
	Comments  string       `xml:"comments"`
}

type rssEnclosure struct {
	URL    string `xml:"url,attr"`
	Length int64  `xml:"length,attr"`
	Type   string `xml:"type,attr"`
}

func parseStandardRSS(data []byte, indexerID int64, indexerName string) ([]types.TorrentInfo, error) {
	var feed rssFeed
	if err := xml.Unmarshal(data, &feed); err != nil {
		return nil, err
	}

	var results []types.TorrentInfo
	for _, item := range feed.Channel.Items {
		downloadURL := item.Link
		if downloadURL == "" && item.Enclosure.URL != "" {
			downloadURL = item.Enclosure.URL
		}
		if downloadURL == "" {
			continue
		}

		size := item.Size
		if size == 0 && item.Enclosure.Length > 0 {
			size = item.Enclosure.Length
		}

		guid := item.GUID
		if guid == "" {
			guid = downloadURL
		}

		pubDate := parseDate(item.PubDate)

		results = append(results, types.TorrentInfo{
			ReleaseInfo: types.ReleaseInfo{
				GUID:        guid,
				Title:       item.Title,
				DownloadURL: downloadURL,
				InfoURL:     item.Comments,
				Size:        size,
				PublishDate: pubDate,
				IndexerID:   indexerID,
				IndexerName: indexerName,
				Protocol:    inferProtocol(downloadURL, item.Enclosure.Type),
			},
		})
	}

	return results, nil
}

// EzRSS structures (torrent namespace)

type ezrssFeed struct {
	XMLName xml.Name     `xml:"rss"`
	Channel ezrssChannel `xml:"channel"`
}

type ezrssChannel struct {
	Items []ezrssItem `xml:"item"`
}

type ezrssItem struct {
	Title     string          `xml:"title"`
	Link      string          `xml:"link"`
	GUID      string          `xml:"guid"`
	PubDate   string          `xml:"pubDate"`
	Enclosure rssEnclosure    `xml:"enclosure"`
	Torrent   ezrssTorrent    `xml:"torrent"`
	Comments  string          `xml:"comments"`
}

type ezrssTorrent struct {
	InfoHash  string `xml:"infoHash"`
	MagnetURI string `xml:"magnetURI"`
	Seeds     int    `xml:"seeds"`
	Peers     int    `xml:"peers"`
	ContentLength int64 `xml:"contentLength"`
}

func parseEzRSS(data []byte, indexerID int64, indexerName string) ([]types.TorrentInfo, error) {
	var feed ezrssFeed
	if err := xml.Unmarshal(data, &feed); err != nil {
		return nil, err
	}

	var results []types.TorrentInfo
	for _, item := range feed.Channel.Items {
		// EzRSS feeds have the torrent namespace
		if item.Torrent.InfoHash == "" && item.Torrent.MagnetURI == "" {
			continue
		}

		downloadURL := item.Torrent.MagnetURI
		if downloadURL == "" {
			downloadURL = item.Link
		}
		if downloadURL == "" && item.Enclosure.URL != "" {
			downloadURL = item.Enclosure.URL
		}
		if downloadURL == "" {
			continue
		}

		size := item.Torrent.ContentLength
		if size == 0 && item.Enclosure.Length > 0 {
			size = item.Enclosure.Length
		}

		guid := item.GUID
		if guid == "" {
			guid = downloadURL
		}

		results = append(results, types.TorrentInfo{
			ReleaseInfo: types.ReleaseInfo{
				GUID:        guid,
				Title:       item.Title,
				DownloadURL: downloadURL,
				InfoURL:     item.Comments,
				Size:        size,
				PublishDate: parseDate(item.PubDate),
				IndexerID:   indexerID,
				IndexerName: indexerName,
				Protocol:    types.ProtocolTorrent,
			},
			Seeders:  item.Torrent.Seeds,
			Leechers: item.Torrent.Peers,
			InfoHash: item.Torrent.InfoHash,
		})
	}

	return results, nil
}

// TorrentPotato (JSON)

type torrentPotatoResponse struct {
	Results []torrentPotatoItem `json:"results"`
}

type torrentPotatoItem struct {
	ReleaseName string `json:"release_name"`
	TorrentID   string `json:"torrent_id"`
	DownloadURL string `json:"download_url"`
	ImdbID      string `json:"imdb_id"`
	Freeleech   bool   `json:"freeleech"`
	Size        int64  `json:"size"`
	Leechers    int    `json:"leechers"`
	Seeders     int    `json:"seeders"`
}

func parseTorrentPotato(data []byte, indexerID int64, indexerName string) ([]types.TorrentInfo, error) {
	var resp torrentPotatoResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	if len(resp.Results) == 0 {
		return nil, fmt.Errorf("no results")
	}

	var results []types.TorrentInfo
	for _, item := range resp.Results {
		if item.DownloadURL == "" {
			continue
		}

		guid := item.TorrentID
		if guid == "" {
			guid = item.DownloadURL
		}

		imdbID := 0
		if item.ImdbID != "" {
			cleaned := strings.TrimPrefix(item.ImdbID, "tt")
			if v, err := strconv.Atoi(cleaned); err == nil {
				imdbID = v
			}
		}

		dvf := 1.0
		if item.Freeleech {
			dvf = 0
		}

		results = append(results, types.TorrentInfo{
			ReleaseInfo: types.ReleaseInfo{
				GUID:        guid,
				Title:       item.ReleaseName,
				DownloadURL: item.DownloadURL,
				Size:        item.Size,
				IndexerID:   indexerID,
				IndexerName: indexerName,
				Protocol:    types.ProtocolTorrent,
				ImdbID:      imdbID,
			},
			Seeders:              item.Seeders,
			Leechers:             item.Leechers,
			DownloadVolumeFactor: dvf,
		})
	}

	return results, nil
}

// Helpers

func parseDate(s string) time.Time {
	for _, layout := range []string{
		time.RFC1123Z,
		time.RFC1123,
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05",
		"Mon, 02 Jan 2006 15:04:05 -0700",
		"2006-01-02 15:04:05",
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

func inferProtocol(url, enclosureType string) types.Protocol {
	if enclosureType == "application/x-nzb" {
		return types.ProtocolUsenet
	}
	if strings.Contains(url, ".nzb") {
		return types.ProtocolUsenet
	}
	return types.ProtocolTorrent
}
