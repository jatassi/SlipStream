// Package imports owns the Import: a set of video files from one source
// brought into one Work. A library scan's root Imports hold what it finds
// and can't place, which wait in Pending imports for the Admin's actions.
// It also owns each root's last comparison, the Library Health category
// and the scan's settings.
package imports

import (
	"context"
	"embed"
	"log/slog"
	"math/rand/v2"
	"strings"
	"sync"
	"time"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/library"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/scanning"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/tasks"
	"github.com/jatassi/slipstream/internal/transfers"
	"github.com/jatassi/slipstream/internal/works"
)

type file = storage.Library

//go:embed schema/*.sql
var schemaFiles embed.FS

func Schema() storage.Schema[file] { return storage.NewSchema[file]("imports", schemaFiles) }

const TaskID tasks.TaskID = "library-scan"

type Changed struct {
	RootIDs []string `json:"rootIds" nullable:"false"`
}

func changedOf(ids ...roots.ID) Changed {
	out := Changed{RootIDs: make([]string, len(ids))}
	for i, id := range ids {
		out.RootIDs[i] = id.String()
	}
	return out
}

// ChangedEvent names the roots whose pending imports changed.
var ChangedEvent = live.AdminEvent[Changed]("imports.changed")

type Publisher[T any] interface {
	Publish(ctx context.Context, v T)
}

type Config struct {
	Library *storage.DB[file]
	Remote  storing.Remote
	Roots   *roots.Roots
	Files   *files.Files
	Works   *works.Works
	Targets *targets.Targets
	Facets  map[works.TypeID]scanning.Facet
	// Add is library.Add. The library is built after imports, which is its
	// Leaver.
	Add func(ctx context.Context, by identity.Admin, a library.Adding) (library.Added, error)
	// Start and Steps are the Scheduler's, which opens after imports.
	Start        func(ctx context.Context, st tasks.Start) (tasks.Started, error)
	Steps        func(task tasks.TaskID) (run tasks.RunID, steps map[tasks.UnitKey]tasks.Step, open bool)
	Hub          *live.Hub
	Changed      Publisher[Changed]
	FilesChanged Publisher[files.Changed]
	// Probes reads a file's media record where its bytes are. Transfers
	// sends a file to another Node's root, and is asked what live
	// Transfers still send to a volume.
	Probes    storing.Probes
	Transfers *transfers.Transfers
	// Grabs is told how each download's Import goes, through Follow.
	Grabs   *grabs.Grabs
	Timings Timings
	Rand    *rand.Rand
	// BeforeStep sees each root's unit before each phase, and BeforeImport
	// each Import file's IO before it runs, in tests.
	BeforeStep   func(ctx context.Context, root roots.ID, phase Phase)
	BeforeImport func(ctx context.Context, at Step)
	Log          *slog.Logger
}

// Timings are the reconciler's, the grabs backoff: an import wait retries
// after RetryMin, doubling to RetryMax, and a pass runs at least every
// RetryMax.
type Timings struct {
	RetryMin, RetryMax time.Duration
}

type Imports struct {
	cfg    Config
	probes *slots
	wake   chan struct{}

	mu sync.Mutex
	// hooked is what each job's Transfers hooks changed, published once
	// their Write commits.
	hooked map[string]after
}

// New builds the pipeline and registers its Transfers purpose.
func New(cfg Config) *Imports {
	im := &Imports{cfg: cfg, probes: newSlots(), wake: make(chan struct{}, 1), hooked: map[string]after{}}
	cfg.Transfers.Register(im.purpose())
	return im
}

// Touch has the reconciler pass soon.
func (im *Imports) Touch() {
	select {
	case im.wake <- struct{}{}:
	default:
	}
}

func unitKey(root roots.ID) tasks.UnitKey { return tasks.UnitKey("root:" + root.String()) }

// folderOf is a path's Work folder: its first folder, or the path itself
// when it lies loose at the root's top.
func folderOf(path string) (folder string, loose bool) {
	if folder, _, ok := strings.Cut(path, "/"); ok {
		return folder, false
	}
	return path, true
}

// inside is a path inside its Work folder, as a Placer reads it: a loose
// file's own name.
func inside(path string) string {
	if _, rest, ok := strings.Cut(path, "/"); ok {
		return rest
	}
	return path
}
