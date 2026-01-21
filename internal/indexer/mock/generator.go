package mock

import (
	"fmt"

	"github.com/slipstream/slipstream/internal/indexer/types"
)

// MockMedia represents a movie or series for generating mock releases.
type MockMedia struct {
	Title  string
	Year   int
	TmdbID int
	TvdbID int
	ImdbID int
}

// generateMovieReleases creates mock releases for a movie.
func generateMovieReleases(m MockMedia) []types.ReleaseInfo {
	baseGUID := fmt.Sprintf("https://mockindexer.org/torrent/%d", m.TmdbID)

	return []types.ReleaseInfo{
		{
			GUID:        fmt.Sprintf("%s/1", baseGUID),
			Title:       fmt.Sprintf("%s.%d.2160p.UHD.BluRay.REMUX.HDR.HEVC.TrueHD.7.1.Atmos-MOCK", sanitizeTitle(m.Title), m.Year),
			DownloadURL: fmt.Sprintf("%s/1/download", baseGUID),
			InfoURL:     fmt.Sprintf("%s/1", baseGUID),
			Size:        65_000_000_000, // ~65GB
			Categories:  []int{2000},
			TmdbID:      m.TmdbID,
			ImdbID:      m.ImdbID,
			Quality:     "2160p",
			Source:      "Remux",
			Resolution:  2160,
		},
		{
			GUID:        fmt.Sprintf("%s/2", baseGUID),
			Title:       fmt.Sprintf("%s.%d.2160p.WEB-DL.DDP5.1.DV.HDR.H.265-MOCK", sanitizeTitle(m.Title), m.Year),
			DownloadURL: fmt.Sprintf("%s/2/download", baseGUID),
			InfoURL:     fmt.Sprintf("%s/2", baseGUID),
			Size:        25_000_000_000, // ~25GB
			Categories:  []int{2000},
			TmdbID:      m.TmdbID,
			ImdbID:      m.ImdbID,
			Quality:     "2160p",
			Source:      "WEB-DL",
			Resolution:  2160,
		},
		{
			GUID:        fmt.Sprintf("%s/3", baseGUID),
			Title:       fmt.Sprintf("%s.%d.1080p.BluRay.x264.DTS-HD.MA.5.1-MOCK", sanitizeTitle(m.Title), m.Year),
			DownloadURL: fmt.Sprintf("%s/3/download", baseGUID),
			InfoURL:     fmt.Sprintf("%s/3", baseGUID),
			Size:        12_000_000_000, // ~12GB
			Categories:  []int{2000},
			TmdbID:      m.TmdbID,
			ImdbID:      m.ImdbID,
			Quality:     "1080p",
			Source:      "BluRay",
			Resolution:  1080,
		},
		{
			GUID:        fmt.Sprintf("%s/4", baseGUID),
			Title:       fmt.Sprintf("%s.%d.1080p.WEB-DL.DDP5.1.H.264-MOCK", sanitizeTitle(m.Title), m.Year),
			DownloadURL: fmt.Sprintf("%s/4/download", baseGUID),
			InfoURL:     fmt.Sprintf("%s/4", baseGUID),
			Size:        8_000_000_000, // ~8GB
			Categories:  []int{2000},
			TmdbID:      m.TmdbID,
			ImdbID:      m.ImdbID,
			Quality:     "1080p",
			Source:      "WEB-DL",
			Resolution:  1080,
		},
	}
}

// generateTVReleases creates mock releases for a TV series (season packs for seasons 1-5).
func generateTVReleases(m MockMedia) []types.ReleaseInfo {
	baseGUID := fmt.Sprintf("https://mockindexer.org/torrent/tv/%d", m.TvdbID)

	var releases []types.ReleaseInfo

	// Generate releases for seasons 1-5
	for season := 1; season <= 5; season++ {
		seasonStr := fmt.Sprintf("S%02d", season)
		seasonKey := fmt.Sprintf("s%02d", season)

		releases = append(releases,
			types.ReleaseInfo{
				GUID:        fmt.Sprintf("%s/%s-2160p-remux", baseGUID, seasonKey),
				Title:       fmt.Sprintf("%s.%s.2160p.UHD.BluRay.REMUX.DV.HDR.HEVC.TrueHD.7.1.Atmos-MOCK", sanitizeTitle(m.Title), seasonStr),
				DownloadURL: fmt.Sprintf("%s/%s-2160p-remux/download", baseGUID, seasonKey),
				InfoURL:     fmt.Sprintf("%s/%s-2160p-remux", baseGUID, seasonKey),
				Size:        120_000_000_000, // ~120GB
				Categories:  []int{5000},
				TmdbID:      0,
				TvdbID:      m.TvdbID,
				ImdbID:      m.ImdbID,
				Quality:     "2160p",
				Source:      "Remux",
				Resolution:  2160,
			},
			types.ReleaseInfo{
				GUID:        fmt.Sprintf("%s/%s-2160p-webdl", baseGUID, seasonKey),
				Title:       fmt.Sprintf("%s.%s.2160p.WEB-DL.DDP5.1.DV.HDR.H.265-MOCK", sanitizeTitle(m.Title), seasonStr),
				DownloadURL: fmt.Sprintf("%s/%s-2160p-webdl/download", baseGUID, seasonKey),
				InfoURL:     fmt.Sprintf("%s/%s-2160p-webdl", baseGUID, seasonKey),
				Size:        45_000_000_000, // ~45GB
				Categories:  []int{5000},
				TmdbID:      0,
				TvdbID:      m.TvdbID,
				ImdbID:      m.ImdbID,
				Quality:     "2160p",
				Source:      "WEB-DL",
				Resolution:  2160,
			},
			types.ReleaseInfo{
				GUID:        fmt.Sprintf("%s/%s-1080p-bluray", baseGUID, seasonKey),
				Title:       fmt.Sprintf("%s.%s.1080p.BluRay.x264.DTS-HD.MA.5.1-MOCK", sanitizeTitle(m.Title), seasonStr),
				DownloadURL: fmt.Sprintf("%s/%s-1080p-bluray/download", baseGUID, seasonKey),
				InfoURL:     fmt.Sprintf("%s/%s-1080p-bluray", baseGUID, seasonKey),
				Size:        35_000_000_000, // ~35GB
				Categories:  []int{5000},
				TmdbID:      0,
				TvdbID:      m.TvdbID,
				ImdbID:      m.ImdbID,
				Quality:     "1080p",
				Source:      "BluRay",
				Resolution:  1080,
			},
			types.ReleaseInfo{
				GUID:        fmt.Sprintf("%s/%s-1080p-webdl", baseGUID, seasonKey),
				Title:       fmt.Sprintf("%s.%s.1080p.WEB-DL.DDP5.1.H.264-MOCK", sanitizeTitle(m.Title), seasonStr),
				DownloadURL: fmt.Sprintf("%s/%s-1080p-webdl/download", baseGUID, seasonKey),
				InfoURL:     fmt.Sprintf("%s/%s-1080p-webdl", baseGUID, seasonKey),
				Size:        18_000_000_000, // ~18GB
				Categories:  []int{5000},
				TmdbID:      0,
				TvdbID:      m.TvdbID,
				ImdbID:      m.ImdbID,
				Quality:     "1080p",
				Source:      "WEB-DL",
				Resolution:  1080,
			},
		)
	}

	return releases
}

// sanitizeTitle converts a title to release format (dots instead of spaces, no special chars).
func sanitizeTitle(title string) string {
	result := ""
	for _, r := range title {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			result += string(r)
		case r == ' ', r == '-', r == '_':
			result += "."
		case r == '\'', r == ':':
			// Skip apostrophes and colons
		}
	}
	return result
}
