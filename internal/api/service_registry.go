package api

import (
	"database/sql"

	"github.com/slipstream/slipstream/internal/database/sqlc"
)

// DBSwitchable is implemented by services that accept a *sql.DB for database switching.
type DBSwitchable interface {
	SetDB(db *sql.DB)
}

// QueriesSwitchable is implemented by services that accept *sqlc.Queries for database switching.
type QueriesSwitchable interface {
	SetDB(queries *sqlc.Queries)
}

// ServiceRegistry tracks services for bulk database switching (dev mode toggle).
type ServiceRegistry struct {
	dbServices      []DBSwitchable
	queriesServices []QueriesSwitchable
}

// RegisterDB adds services that accept *sql.DB.
func (r *ServiceRegistry) RegisterDB(svcs ...DBSwitchable) {
	r.dbServices = append(r.dbServices, svcs...)
}

// RegisterQueries adds services that accept *sqlc.Queries.
func (r *ServiceRegistry) RegisterQueries(svcs ...QueriesSwitchable) {
	r.queriesServices = append(r.queriesServices, svcs...)
}

// UpdateAll switches all registered services to use the given database.
func (r *ServiceRegistry) UpdateAll(db *sql.DB) {
	for _, svc := range r.dbServices {
		svc.SetDB(db)
	}
	queries := sqlc.New(db)
	for _, svc := range r.queriesServices {
		svc.SetDB(queries)
	}
}
