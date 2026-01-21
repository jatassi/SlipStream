package mock

import (
	"context"
	"strings"
	"time"

	"github.com/slipstream/slipstream/internal/indexer/types"
)

// Client implements the indexer.Indexer interface for mock testing.
type Client struct {
	indexerDef    *types.IndexerDefinition
	movieReleases map[int][]types.ReleaseInfo
	tvReleases    map[int][]types.ReleaseInfo
}

// Ensure Client implements the Indexer interfaces.
var _ interface {
	Name() string
	Definition() *types.IndexerDefinition
	GetSettings() map[string]string
	Test(ctx context.Context) error
	Search(ctx context.Context, criteria types.SearchCriteria) ([]types.ReleaseInfo, error)
	Download(ctx context.Context, url string) ([]byte, error)
	Capabilities() *types.Capabilities
	SupportsSearch() bool
	SupportsRSS() bool
	SearchTorrents(ctx context.Context, criteria types.SearchCriteria) ([]types.TorrentInfo, error)
} = (*Client)(nil)

// NewClient creates a new mock indexer client.
func NewClient(indexerDef *types.IndexerDefinition) *Client {
	return &Client{
		indexerDef:    indexerDef,
		movieReleases: buildMovieReleasesMap(),
		tvReleases:    buildTVReleasesMap(),
	}
}

func (c *Client) Name() string {
	return c.indexerDef.Name
}

func (c *Client) Definition() *types.IndexerDefinition {
	return c.indexerDef
}

func (c *Client) GetSettings() map[string]string {
	return map[string]string{"mock": "true"}
}

func (c *Client) Test(ctx context.Context) error {
	return nil // Mock indexer always succeeds
}

func (c *Client) Search(ctx context.Context, criteria types.SearchCriteria) ([]types.ReleaseInfo, error) {
	torrents, err := c.SearchTorrents(ctx, criteria)
	if err != nil {
		return nil, err
	}

	releases := make([]types.ReleaseInfo, len(torrents))
	for i, t := range torrents {
		releases[i] = t.ReleaseInfo
	}
	return releases, nil
}

func (c *Client) SearchTorrents(ctx context.Context, criteria types.SearchCriteria) ([]types.TorrentInfo, error) {
	var releases []types.ReleaseInfo

	// Try to find results by ID
	if criteria.TmdbID > 0 {
		if r, ok := c.movieReleases[criteria.TmdbID]; ok {
			releases = append(releases, r...)
		}
	}

	if criteria.TvdbID > 0 {
		if r, ok := c.tvReleases[criteria.TvdbID]; ok {
			releases = append(releases, r...)
		}
	}

	// Fallback to query search if no ID results
	if len(releases) == 0 && criteria.Query != "" {
		releases = c.searchByQuery(criteria.Query)
	}

	// Convert to TorrentInfo with seeder data
	torrents := make([]types.TorrentInfo, len(releases))
	for i, r := range releases {
		// Update indexer info to reflect this mock indexer
		r.IndexerID = c.indexerDef.ID
		r.IndexerName = c.indexerDef.Name
		r.Protocol = types.ProtocolTorrent
		r.PublishDate = time.Now().Add(-time.Duration(i*24) * time.Hour)

		torrents[i] = types.TorrentInfo{
			ReleaseInfo:          r,
			Seeders:              100 + (i * 10),
			Leechers:             5 + i,
			DownloadVolumeFactor: 0, // Freeleech
			UploadVolumeFactor:   1,
		}
	}

	return torrents, nil
}

func (c *Client) searchByQuery(query string) []types.ReleaseInfo {
	query = strings.ToLower(query)
	var results []types.ReleaseInfo

	// Search through all movie results
	for _, releases := range c.movieReleases {
		for _, r := range releases {
			if strings.Contains(strings.ToLower(r.Title), query) {
				results = append(results, r)
			}
		}
	}

	// Search through all TV results
	for _, releases := range c.tvReleases {
		for _, r := range releases {
			if strings.Contains(strings.ToLower(r.Title), query) {
				results = append(results, r)
			}
		}
	}

	return results
}

func (c *Client) Download(ctx context.Context, url string) ([]byte, error) {
	// Return mock torrent data - a minimal valid torrent structure
	return []byte("mock-torrent-data"), nil
}

func (c *Client) Capabilities() *types.Capabilities {
	return &types.Capabilities{
		SupportsMovies:      true,
		SupportsTV:          true,
		SupportsSearch:      true,
		SupportsRSS:         true,
		MaxResultsPerSearch: 100,
		SearchParams:        []string{"q"},
		TvSearchParams:      []string{"q", "tvdbid", "season", "ep"},
		MovieSearchParams:   []string{"q", "tmdbid", "imdbid"},
		Categories: []types.CategoryMapping{
			{ID: 2000, Name: "Movies"},
			{ID: 5000, Name: "TV"},
		},
	}
}

func (c *Client) SupportsSearch() bool {
	return true
}

func (c *Client) SupportsRSS() bool {
	return true
}
