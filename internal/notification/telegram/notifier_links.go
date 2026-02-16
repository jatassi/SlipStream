package telegram

import (
	"fmt"
	"strings"
)

type linkEntry struct {
	linkType MetadataLink
	url      string
}

func (n *Notifier) buildLinks(entries []linkEntry) string {
	if !n.settings.IncludeLinks {
		return ""
	}

	var links []string
	for _, e := range entries {
		if n.hasLink(e.linkType) {
			links = append(links, e.url)
		}
	}
	if len(links) == 0 {
		return ""
	}
	return strings.Join(links, " | ") + "\n"
}

func movieLinkEntries(tmdbID int64, imdbID string, traktID int64) []linkEntry {
	var entries []linkEntry
	if tmdbID > 0 {
		entries = append(entries, linkEntry{MetadataLinkTMDb, fmt.Sprintf("<a href=\"https://www.themoviedb.org/movie/%d\">TMDb</a>", tmdbID)})
	}
	if imdbID != "" {
		entries = append(entries, linkEntry{MetadataLinkIMDb, fmt.Sprintf("<a href=\"https://www.imdb.com/title/%s\">IMDb</a>", imdbID)})
	}
	if traktID > 0 {
		entries = append(entries, linkEntry{MetadataLinkTrakt, fmt.Sprintf("<a href=\"https://trakt.tv/movies/%d\">Trakt</a>", traktID)})
	}
	return entries
}

func seriesLinkEntries(tmdbID int64, imdbID string, tvdbID, traktID int64) []linkEntry {
	var entries []linkEntry
	if tmdbID > 0 {
		entries = append(entries, linkEntry{MetadataLinkTMDb, fmt.Sprintf("<a href=\"https://www.themoviedb.org/tv/%d\">TMDb</a>", tmdbID)})
	}
	if imdbID != "" {
		entries = append(entries, linkEntry{MetadataLinkIMDb, fmt.Sprintf("<a href=\"https://www.imdb.com/title/%s\">IMDb</a>", imdbID)})
	}
	if tvdbID > 0 {
		entries = append(entries, linkEntry{MetadataLinkTVDb, fmt.Sprintf("<a href=\"https://thetvdb.com/series/%d\">TVDb</a>", tvdbID)})
	}
	if traktID > 0 {
		entries = append(entries, linkEntry{MetadataLinkTrakt, fmt.Sprintf("<a href=\"https://trakt.tv/shows/%d\">Trakt</a>", traktID)})
	}
	return entries
}
