package module

import (
	"database/sql"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/database/sqlc"
	"github.com/slipstream/slipstream/internal/domain/contracts"
	"github.com/slipstream/slipstream/internal/library/quality"
	"github.com/slipstream/slipstream/internal/websocket"
)

// BaseService provides the common fields and initialization shared by all
// module service implementations. Module services embed this struct.
type BaseService struct {
	DB                 *sql.DB
	Queries            *sqlc.Queries
	Hub                *websocket.Hub
	Logger             *zerolog.Logger
	StatusChangeLogger contracts.StatusChangeLogger
	QualityProfiles    *quality.Service
}

// NewBaseService creates a BaseService with standard initialization.
// componentName is used for the logger's "component" field (e.g., "movies", "tv").
func NewBaseService(db *sql.DB, hub *websocket.Hub, logger *zerolog.Logger, qualityService *quality.Service, statusChangeLogger contracts.StatusChangeLogger, componentName string) BaseService {
	subLogger := logger.With().Str("component", componentName).Logger()
	return BaseService{
		DB:                 db,
		Queries:            sqlc.New(db),
		Hub:                hub,
		Logger:             &subLogger,
		QualityProfiles:    qualityService,
		StatusChangeLogger: statusChangeLogger,
	}
}

// Broadcast sends a WebSocket event if the hub is available.
func (bs *BaseService) Broadcast(event string, data map[string]any) {
	if bs.Hub != nil {
		bs.Hub.Broadcast(event, data)
	}
}

// BroadcastEntity sends a WebSocket entity event if the hub is available.
func (bs *BaseService) BroadcastEntity(moduleType, entityType string, entityID int64, action string, payload interface{}) {
	if bs.Hub != nil {
		bs.Hub.BroadcastEntity(moduleType, entityType, entityID, action, payload)
	}
}

// SetDB updates the database connection used by this service.
// This is called when switching between production and development databases.
func (bs *BaseService) SetDB(db *sql.DB) {
	bs.DB = db
	bs.Queries = sqlc.New(db)
}
