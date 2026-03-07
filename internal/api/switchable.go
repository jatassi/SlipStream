package api

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"

	"github.com/slipstream/slipstream/internal/auth"
	"github.com/slipstream/slipstream/internal/autosearch"
	"github.com/slipstream/slipstream/internal/availability"
	"github.com/slipstream/slipstream/internal/calendar"
	"github.com/slipstream/slipstream/internal/database/sqlc"
	"github.com/slipstream/slipstream/internal/defaults"
	"github.com/slipstream/slipstream/internal/downloader"
	"github.com/slipstream/slipstream/internal/history"
	importer "github.com/slipstream/slipstream/internal/import"
	"github.com/slipstream/slipstream/internal/indexer"
	"github.com/slipstream/slipstream/internal/indexer/grab"
	"github.com/slipstream/slipstream/internal/indexer/ratelimit"
	"github.com/slipstream/slipstream/internal/indexer/status"
	"github.com/slipstream/slipstream/internal/library/librarymanager"
	"github.com/slipstream/slipstream/internal/library/movies"
	"github.com/slipstream/slipstream/internal/library/quality"
	"github.com/slipstream/slipstream/internal/library/rootfolder"
	"github.com/slipstream/slipstream/internal/library/slots"
	"github.com/slipstream/slipstream/internal/library/tv"
	"github.com/slipstream/slipstream/internal/metadata"
	"github.com/slipstream/slipstream/internal/missing"
	"github.com/slipstream/slipstream/internal/notification"
	"github.com/slipstream/slipstream/internal/portal/admin"
	"github.com/slipstream/slipstream/internal/portal/autoapprove"
	"github.com/slipstream/slipstream/internal/portal/invitations"
	portalnotifs "github.com/slipstream/slipstream/internal/portal/notifications"
	"github.com/slipstream/slipstream/internal/portal/quota"
	"github.com/slipstream/slipstream/internal/portal/requests"
	"github.com/slipstream/slipstream/internal/portal/users"
	"github.com/slipstream/slipstream/internal/preferences"
	"github.com/slipstream/slipstream/internal/prowlarr"
	"github.com/slipstream/slipstream/internal/rsssync"
)

// SwitchableServices tracks all services that need database switching for dev mode toggle.
// Fields tagged with `switchable:"db"` have SetDB(*sql.DB).
// Fields tagged with `switchable:"queries"` have SetDB(*sqlc.Queries).
// The reflection-based UpdateAll and Validate methods eliminate manual service enumeration.
type SwitchableServices struct {
	// Services that accept *sql.DB
	Defaults            *defaults.Service                  `switchable:"db"`
	Calendar            *calendar.Service                  `switchable:"db"`
	Availability        *availability.Service              `switchable:"db"`
	Missing             *missing.Service                   `switchable:"db"`
	History             *history.Service                   `switchable:"db"`
	Preferences         *preferences.Service               `switchable:"db"`
	Movies              *movies.Service                    `switchable:"db"`
	TV                  *tv.Service                        `switchable:"db"`
	Quality             *quality.Service                   `switchable:"db"`
	Slots               *slots.Service                     `switchable:"db"`
	RootFolder          *rootfolder.Service                `switchable:"db"`
	NetworkLogoStore    *metadata.SQLNetworkLogoStore      `switchable:"db"`
	Download            *downloader.Service                `switchable:"db"`
	Indexer             *indexer.Service                   `switchable:"db"`
	Status              *status.Service                    `switchable:"db"`
	RateLimiter         *ratelimit.Limiter                 `switchable:"db"`
	Grab                *grab.Service                      `switchable:"db"`
	Prowlarr            *prowlarr.Service                  `switchable:"db"`
	Autosearch          *autosearch.Service                `switchable:"db"`
	Import              *importer.Service                  `switchable:"db"`
	ImportSettings      *importer.SettingsHandlers         `switchable:"db"`
	LibraryManager      *librarymanager.Service            `switchable:"db"`
	Notification        *notification.Service              `switchable:"db"`
	StatusTracker       *requests.StatusTracker            `switchable:"db"`
	LibraryChecker      *requests.LibraryChecker           `switchable:"db"`
	AdminLibraryChecker *adminRequestLibraryCheckerAdapter `switchable:"db"`
	Watchers            *requests.WatchersService          `switchable:"db"`
	RequestSearcher     *requests.RequestSearcher          `switchable:"db"`

	// Services that accept *sqlc.Queries
	RssSync         *rsssync.Service         `switchable:"queries"`
	RssSyncSettings *rsssync.SettingsHandler `switchable:"queries"`
	Users           *users.Service           `switchable:"queries"`
	Invitations     *invitations.Service     `switchable:"queries"`
	Quota           *quota.Service           `switchable:"queries"`
	Notifications   *portalnotifs.Service    `switchable:"queries"`
	AutoApprove     *autoapprove.Service     `switchable:"queries"`
	Requests        *requests.Service        `switchable:"queries"`
	Auth            *auth.Service
	AdminSettings   *admin.SettingsHandlers `switchable:"queries"`
	Passkey         *auth.PasskeyService    `switchable:"queries,optional"`
}

// UpdateAll switches all registered services to use the given database.
// Services tagged with switchable:"db" receive *sql.DB.
// Services tagged with switchable:"queries" receive *sqlc.Queries.
func (sw *SwitchableServices) UpdateAll(db *sql.DB) {
	queries := sqlc.New(db)
	v := reflect.ValueOf(sw).Elem()
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		tag := t.Field(i).Tag.Get("switchable")
		if tag == "" || field.IsNil() {
			continue
		}
		method := field.MethodByName("SetDB")
		if !method.IsValid() {
			panic(fmt.Sprintf("switchable field %s has no SetDB method", t.Field(i).Name))
		}
		switch {
		case strings.HasPrefix(tag, "db"):
			method.Call([]reflect.Value{reflect.ValueOf(db)})
		case strings.HasPrefix(tag, "queries"):
			method.Call([]reflect.Value{reflect.ValueOf(queries)})
		}
	}
}

// Validate checks that all required switchable fields are assigned.
// Fields tagged with "optional" (e.g., switchable:"queries,optional") are skipped.
func (sw *SwitchableServices) Validate() error {
	v := reflect.ValueOf(sw).Elem()
	t := v.Type()
	var unset []string
	for i := 0; i < v.NumField(); i++ {
		tag := t.Field(i).Tag.Get("switchable")
		if tag == "" {
			continue
		}
		if strings.Contains(tag, "optional") {
			continue
		}
		if v.Field(i).IsNil() {
			unset = append(unset, t.Field(i).Name)
		}
	}
	if len(unset) > 0 {
		return fmt.Errorf("switchable services not registered: %s", strings.Join(unset, ", "))
	}
	return nil
}
