package imports

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/jatassi/slipstream/internal/imports/internal/db"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/scanning"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/tasks"
)

// Phase is how far a root's unit of Library scan is.
type Phase string

const (
	Listing   Phase = "listing"
	Comparing Phase = "comparing"
	Probing   Phase = "probing"
)

// Line is a root's scan line: its last comparison at its place and, while
// its unit runs, its step.
type Line struct {
	// ComparedAt is zero until a comparison finished at the root's place.
	ComparedAt storage.Time
	Stopped    *scanning.Stopped
	// Summary is what the comparison found, "It found 3 new files and 1
	// gone file.", or why it stopped. Empty until one finished.
	Summary string
	Running *Running
}

type Running struct {
	Run   tasks.RunID
	Phase Phase
	Done  int
	Total *int
}

// Lines is each root's scan line. A root with neither a comparison nor a
// running unit has none.
func (im *Imports) Lines(ctx context.Context) (map[roots.ID]Line, error) {
	var (
		all      []roots.Root
		compared []db.ImportsRoot
	)
	err := im.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		var err error
		if all, err = roots.All(ctx, r); err != nil {
			return err
		}
		compared, err = db.New(r).ListCompared(ctx)
		return err
	})
	if err != nil {
		return nil, err
	}
	run, steps, _ := im.cfg.Steps(TaskID)
	out := map[roots.ID]Line{}
	for _, root := range all {
		var l Line
		for _, c := range compared {
			if c.RootID == root.ID && c.Place == root.Place && !c.ComparedAt.IsZero() {
				l = lineOf(c)
			}
		}
		if st, ok := steps[unitKey(root.ID)]; ok {
			l.Running = &Running{Run: run, Phase: Phase(st.Phase), Done: st.Done, Total: st.Total}
		}
		if l != (Line{}) {
			out[root.ID] = l
		}
	}
	return out, nil
}

func lineOf(c db.ImportsRoot) Line {
	l := Line{ComparedAt: c.ComparedAt}
	if c.StoppedGone != nil {
		l.Stopped = &scanning.Stopped{Gone: int(*c.StoppedGone), Of: int(*c.StoppedOf)}
		l.Summary = stoppedWords(*l.Stopped)
		return l
	}
	l.Summary = foundWords(int(c.Found), int(c.Gone), int(c.Returned))
	return l
}

func foundWords(found, gone, returned int) string {
	var parts []string
	for _, p := range []struct {
		n         int
		one, many string
	}{{found, "new file", "new files"}, {gone, "gone file", "gone files"}, {returned, "returned file", "returned files"}} {
		if p.n > 0 {
			parts = append(parts, count(p.n, p.one, p.many))
		}
	}
	switch len(parts) {
	case 0:
		return "Nothing changed."
	case 1:
		return "It found " + parts[0] + "."
	}
	return "It found " + strings.Join(parts[:len(parts)-1], ", ") + " and " + parts[len(parts)-1] + "."
}

// stoppedWords is what a stopped comparison says, on its root's row and
// in its Health item.
func stoppedWords(s scanning.Stopped) string {
	verb := "are"
	if s.Gone == 1 {
		verb = "is"
	}
	return fmt.Sprintf("Library scan stopped because %s of %s %s gone. Check the disk or share it is on, then scan again.",
		grouped(s.Gone), count(s.Of, "file", "files"), verb)
}

// count is n and its noun: "1 file", "1,204 files".
func count(n int, one, many string) string {
	if n == 1 {
		return "1 " + one
	}
	return grouped(n) + " " + many
}

// grouped is n with its thousands grouped by commas, as English writes it.
func grouped(n int) string {
	s := strconv.Itoa(n)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}

// Comparison is what one comparison of a root found, as its record keeps
// it.
type Comparison struct {
	// Place is the root's place the comparison read.
	Place                        int64
	Found, Gone, Returned, Added int
	Stopped                      *scanning.Stopped
	// Consumed is the count Mark them gone confirmed that the comparison
	// ran with, zero for none.
	Consumed int
}

// record keeps c as root's last comparison in tx. It clears Mark them
// gone's count only while it still holds the one c ran with, so a second
// Mark them gone during the comparison applies to the next.
func record(ctx context.Context, tx *storage.Tx[file], root roots.ID, c Comparison, now storage.Time) error {
	q := db.New(tx)
	p := db.RecordComparisonParams{
		RootID: root, Place: c.Place, ComparedAt: now, Found: int64(c.Found), Gone: int64(c.Gone), Returned: int64(c.Returned), Added: int64(c.Added),
	}
	if c.Stopped != nil {
		p.StoppedGone, p.StoppedOf = new(int64(c.Stopped.Gone)), new(int64(c.Stopped.Of))
	}
	if err := q.RecordComparison(ctx, p); err != nil {
		return err
	}
	if c.Consumed == 0 {
		return nil
	}
	return q.ConsumeConfirmed(ctx, db.ConsumeConfirmedParams{RootID: root, Confirmed: new(int64(c.Consumed))})
}

var ErrNotStopped = errors.New("the root's last library scan didn't stop")

// confirmGone is Mark them gone's record: the count of root's stopped
// comparison, which the next comparison passes up to. ErrNotStopped when
// root's last comparison didn't stop.
func (im *Imports) confirmGone(ctx context.Context, root roots.ID) error {
	return im.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		n, err := db.New(tx).ConfirmGone(ctx, root)
		if err == nil && n == 0 {
			return ErrNotStopped
		}
		return err
	})
}

// guardOf is the stop rule a comparison of root runs with, read in r.
func guardOf(ctx context.Context, r storage.Reader[file], root roots.ID) (scanning.Guard, error) {
	q := db.New(r)
	s, err := q.GetSettings(ctx)
	if err != nil {
		return scanning.Guard{}, err
	}
	g := scanning.Guard{Share: int(StopShare.Value(s.StopShare))}
	c, err := q.GetCompared(ctx, root)
	switch {
	case errors.Is(err, storage.ErrNoRows):
		return g, nil
	case err != nil:
		return scanning.Guard{}, err
	}
	if c.Confirmed != nil {
		g.Confirmed = int(*c.Confirmed)
	}
	return g, nil
}

// skipCause is why a root's unit skips while the root reads s, a State
// other than Ready: Storage's sentences for a root in trouble, and its
// words for one waiting on its Node, named node.
func skipCause(s roots.State, node string) string {
	switch s {
	case roots.Checking:
		return "Waiting for " + node + " to check the folder."
	case roots.Ready, roots.Offline, roots.Missing, roots.Unmounted, roots.Foreign, roots.Unwritable, roots.Unresponsive, roots.WrittenOff:
	}
	sentence, fix, _ := roots.Trouble(s)
	return sentence + " " + fix
}
