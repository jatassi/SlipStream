package surface_test

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/api/apitest"
	"github.com/jatassi/slipstream/internal/backups"
	"github.com/jatassi/slipstream/internal/calendar"
	"github.com/jatassi/slipstream/internal/conversions"
	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/firstrun"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/health"
	"github.com/jatassi/slipstream/internal/history"
	"github.com/jatassi/slipstream/internal/home"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/imports"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/joincode"
	"github.com/jatassi/slipstream/internal/library"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/mail"
	"github.com/jatassi/slipstream/internal/mediaservers"
	"github.com/jatassi/slipstream/internal/metadata"
	"github.com/jatassi/slipstream/internal/metadata/artwork"
	"github.com/jatassi/slipstream/internal/moves"
	"github.com/jatassi/slipstream/internal/naming"
	"github.com/jatassi/slipstream/internal/node/surface"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/nodesystem"
	"github.com/jatassi/slipstream/internal/notifications"
	"github.com/jatassi/slipstream/internal/pairing"
	"github.com/jatassi/slipstream/internal/reachability"
	"github.com/jatassi/slipstream/internal/reconciling"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/servingapi"
	"github.com/jatassi/slipstream/internal/subtitles"
	"github.com/jatassi/slipstream/internal/supersededapi"
	"github.com/jatassi/slipstream/internal/system"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/tasks"
	"github.com/jatassi/slipstream/internal/trackers"
	"github.com/jatassi/slipstream/internal/transfers"
	"github.com/jatassi/slipstream/internal/works"
)

// worker is a Worker's Node: a pairing record and no library.db.
type worker struct{}

func (worker) Claim(context.Context) (system.Claim, error) { return system.Worker, nil }
func (worker) Identity() (*identity.Service, bool)         { return nil, false }
func (worker) Mail() (*mail.Mail, bool)                    { return nil, false }

func (worker) ClaimController(context.Context, func(*identity.Service) error) error {
	panic("a Worker's closed gate admitted claimController")
}

func (worker) Nodes() (*nodes.Service, bool) { return nil, false }

func (worker) Systems() (*nodesystem.Nodes, bool) { return nil, false }

func (worker) Works() (*works.Works, bool) { return nil, false }

func (worker) MediaServers() (mediaservers.Service, bool) { return nil, false }

func (worker) Reachability() (*reachability.Reachability, bool) { return nil, false }

func (worker) Notifications() (*notifications.Service, bool) { return nil, false }

func (worker) Calendar() (*calendar.Calendar, bool) { return nil, false }
func (worker) Home() (*home.Service, bool)          { return nil, false }

func (worker) History() (*history.Service, bool) { return nil, false }

func (worker) Subtitles() (*subtitles.Subtitles, bool) { return nil, false }

func (worker) Minter() (artwork.Minter, bool) { return artwork.Minter{}, false }

func (worker) Metadata() (*metadata.Metadata, bool) { return nil, false }

func (worker) Indexers() (*indexers.Indexers, bool) { return nil, false }

func (worker) Trackers() (*trackers.Trackers, bool) { return nil, false }

func (worker) Targets() (*targets.Targets, bool) { return nil, false }

func (worker) Roots() (*roots.Roots, bool) { return nil, false }

func (worker) Files() (*files.Files, bool) { return nil, false }

func (worker) Naming() (*naming.Naming, bool) { return nil, false }

func (worker) Imports() (*imports.Imports, bool) { return nil, false }

func (worker) Library() (*library.Library, bool) { return nil, false }

func (worker) Grabs() (*grabs.Grabs, bool) { return nil, false }

func (worker) Tasks() (*tasks.Scheduler, bool) { return nil, false }

func (worker) DownloadClients() (*downloadclients.Clients, bool) { return nil, false }
func (worker) Transfers() (*transfers.Transfers, bool)           { return nil, false }
func (worker) Moves() (*moves.Moves, bool)                       { return nil, false }
func (worker) Conversions() (*conversions.Conversions, bool)     { return nil, false }
func (worker) Backups() (*backups.Backups, bool)                 { return nil, false }

func (worker) Health() (*health.Health, bool) { return nil, false }

func (worker) FirstRun() (*firstrun.FirstRun, bool) { return nil, false }

func (worker) FirstRunOpen(context.Context) (bool, error) {
	panic("a Worker read first setup")
}

func (worker) Worker() (servingapi.Facts, bool) {
	return servingapi.Facts{Name: "seedbox", Fingerprint: "AAAA-BBBB-CCCC-DDDD", Controller: "home"}, true
}

func (worker) Join(context.Context, joincode.Code, func(pairing.Pending)) (pairing.Joined, error) {
	panic("a Worker's closed gate admitted claimWorker")
}

func (worker) Removal() (pairing.Removal, bool) { return pairing.Removal{}, false }

func (worker) Superseded() bool { return false }

func (worker) Fence() (supersededapi.Fence, bool) { return nil, false }

func (worker) Pending() (pairing.Pending, bool) {
	panic("a Worker's closed gate admitted getWorkerClaim")
}

func TestTheAccessWalkPassesOnAWorker(t *testing.T) {
	log := slog.New(slog.DiscardHandler)
	gate, _, err := identity.NewGate(t.Context(), worker{}.Claim, log)
	if err != nil {
		t.Fatal(err)
	}
	r := api.NewRouter(api.Services{Node: worker{}, Gate: gate, Web: fstest.MapFS{}, Log: log})
	hub := live.NewHub(surface.Events)
	t.Cleanup(hub.Close)
	surface.Mount(r, surface.Services{Node: worker{}, Hub: hub})
	if breaches := apitest.CheckWorkerAccess(r.Handler(), surface.Document()); len(breaches) != 0 {
		t.Fatalf("breaches:\n%s", strings.Join(breaches, "\n"))
	}
}

func (worker) OpenIn(context.Context, works.WorkID) ([]mediaservers.OpenIn, error) { return nil, nil }

func (worker) Reconciling() (*reconciling.Service, bool) { return nil, false }

func (worker) ReceiveRestore(context.Context, io.Reader) (string, error) {
	panic("a Worker's closed gate admitted uploadRestore")
}

func (worker) UploadRoom(context.Context) (int64, error) {
	panic("a Worker's closed gate admitted uploadRestore")
}

func (worker) OpenRestore(context.Context, string, string) (backups.Summary, error) {
	panic("a Worker's closed gate admitted openRestore")
}

func (worker) RestoreController(context.Context, string, backups.Confirm) error {
	panic("a Worker's closed gate admitted confirmRestore")
}

func (worker) Retire(context.Context) error {
	panic("a Worker's closed gate admitted retireController")
}

func (worker) StageBackup(context.Context, string) (string, backups.Summary, error) {
	panic("a Worker's closed gate admitted stageBackupRestore")
}
