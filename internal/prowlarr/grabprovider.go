package prowlarr

import (
	"context"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/indexer"
)

// ProwlarrIndexerID is the virtual indexer ID used for Prowlarr releases.
// Since Prowlarr aggregates multiple indexers, we use 0 as a sentinel value.
const ProwlarrIndexerID int64 = 0

// InternalIndexerProvider provides access to internal (SlipStream) indexer clients.
type InternalIndexerProvider interface {
	GetClient(ctx context.Context, id int64) (indexer.Indexer, error)
}

// GrabProvider provides indexer clients for grabbing releases.
// It routes requests to either Prowlarr or the internal indexer service based on the indexer mode.
type GrabProvider struct {
	prowlarrService   *Service
	modeManager       *ModeManager
	internalProvider  InternalIndexerProvider
	logger            zerolog.Logger
}

// NewGrabProvider creates a new grab provider.
func NewGrabProvider(
	prowlarrService *Service,
	modeManager *ModeManager,
	internalProvider InternalIndexerProvider,
	logger zerolog.Logger,
) *GrabProvider {
	return &GrabProvider{
		prowlarrService:  prowlarrService,
		modeManager:      modeManager,
		internalProvider: internalProvider,
		logger:           logger.With().Str("component", "prowlarr-grab-provider").Logger(),
	}
}

// GetClient returns an indexer client for the given indexer ID.
// For Prowlarr mode with ID 0 (or when mode check indicates Prowlarr), returns a Prowlarr client wrapper.
// Otherwise, delegates to the internal indexer provider.
func (p *GrabProvider) GetClient(ctx context.Context, id int64) (indexer.Indexer, error) {
	// Check if this is a Prowlarr release (ID 0) or if we're in Prowlarr mode
	isProwlarrRelease := id == ProwlarrIndexerID

	if isProwlarrRelease {
		p.logger.Debug().Int64("indexerId", id).Msg("Returning Prowlarr client for grab")
		return NewGrabClient(p.prowlarrService, p.logger), nil
	}

	// For non-Prowlarr releases, use the internal provider
	return p.internalProvider.GetClient(ctx, id)
}

// GrabClient wraps the Prowlarr service to implement the indexer.Indexer interface
// for download operations.
type GrabClient struct {
	service *Service
	logger  zerolog.Logger
}

// NewGrabClient creates a new Prowlarr grab client.
func NewGrabClient(service *Service, logger zerolog.Logger) *GrabClient {
	return &GrabClient{
		service: service,
		logger:  logger,
	}
}

// Name returns the indexer name.
func (c *GrabClient) Name() string {
	return "Prowlarr"
}

// Definition returns the indexer definition.
func (c *GrabClient) Definition() *indexer.IndexerDefinition {
	return &indexer.IndexerDefinition{
		ID:             ProwlarrIndexerID,
		Name:           "Prowlarr",
		DefinitionID:   "prowlarr",
		Protocol:       indexer.ProtocolTorrent,
		Privacy:        indexer.PrivacyPrivate,
		SupportsMovies: true,
		SupportsTV:     true,
		SupportsSearch: true,
		SupportsRSS:    false,
		Enabled:        true,
	}
}

// GetSettings returns empty settings.
func (c *GrabClient) GetSettings() map[string]string {
	return make(map[string]string)
}

// Test is not used for grab operations.
func (c *GrabClient) Test(ctx context.Context) error {
	return nil
}

// Search is not used for grab operations but required by the Indexer interface.
func (c *GrabClient) Search(ctx context.Context, criteria indexer.SearchCriteria) ([]indexer.ReleaseInfo, error) {
	return nil, nil
}

// Download downloads a torrent/NZB file through Prowlarr.
func (c *GrabClient) Download(ctx context.Context, url string) ([]byte, error) {
	c.logger.Debug().Str("url", url).Msg("Downloading through Prowlarr")
	return c.service.Download(ctx, url)
}

// Capabilities returns empty capabilities for the Prowlarr client.
func (c *GrabClient) Capabilities() *indexer.Capabilities {
	return &indexer.Capabilities{
		SupportsMovies: true,
		SupportsTV:     true,
		SupportsSearch: true,
		SupportsRSS:    false,
	}
}

// SupportsSearch returns true as Prowlarr supports searching.
func (c *GrabClient) SupportsSearch() bool {
	return true
}

// SupportsRSS returns false as Prowlarr doesn't support RSS through this client.
func (c *GrabClient) SupportsRSS() bool {
	return false
}
