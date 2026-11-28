package conversions

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/jatassi/slipstream/internal/conversions/internal/db"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/transfers"
)

// Purpose is the conversion Transfers' purpose: source copies to scratch
// and outputs returned to their root. A copy yields to an Import's on its
// pair.
func (c *Conversions) Purpose() transfers.Purpose {
	return transfers.Purpose{
		Kind: transfers.Conversion, Verified: c.verified, Failed: c.transferFailed, After: c.transferred, Names: c.jobNames, Yields: true,
	}
}

// job is a conversion Transfer's job: a source copy of File to Node, or
// Conversion's output of one attempt.
type job struct {
	Node       role.NodeID
	File       files.ID
	Conversion ID
	Attempt    int
	Output     bool
}

func parseJob(name string) (job, error) {
	kind, rest, _ := strings.Cut(name, "/")
	a, b, _ := strings.Cut(rest, "/")
	var j job
	switch kind {
	case "src":
		node, err := storage.ParseUUID(a)
		if err != nil {
			return job{}, err
		}
		file, err := storage.ParseUUID(b)
		j.Node, j.File = role.NodeID{UUID: node}, files.ID{UUID: file}
		return j, err
	case "out":
		id, err := storage.ParseUUID(a)
		if err != nil {
			return job{}, err
		}
		n, err := strconv.Atoi(b)
		if err != nil {
			return job{}, fmt.Errorf("read the attempt of %q: %w", name, err)
		}
		j.Conversion, j.Attempt, j.Output = ID{UUID: id}, n, true
		return j, nil
	}
	return job{}, fmt.Errorf("%q isn't a conversion job", name)
}

// verified moves the Conversions a finished copy was for: every copying
// one of that source on that Node is ready, and a returned output is
// checked.
func (c *Conversions) verified(ctx context.Context, tx *storage.Tx[storage.Library], t transfers.Transfer, _ string) error {
	j, err := parseJob(t.Job)
	if err != nil {
		return err
	}
	if j.Output {
		return c.advance(ctx, tx, j.Conversion, Event{On: Returned, Node: t.Source, Attempt: j.Attempt, Fenced: true})
	}
	return c.copying(ctx, tx, j, Copied)
}

// transferFailed is a run fault that costs no proof: the copy or the
// return failed.
func (c *Conversions) transferFailed(ctx context.Context, tx *storage.Tx[storage.Library], t transfers.Transfer, _ transfers.Failure) error {
	j, err := parseJob(t.Job)
	if err != nil {
		return err
	}
	if j.Output {
		return c.advance(ctx, tx, j.Conversion, Event{On: ReturnFailed, Node: t.Source, Attempt: j.Attempt, Fenced: true})
	}
	return c.copying(ctx, tx, j, CopyFailed)
}

func (c *Conversions) copying(ctx context.Context, tx *storage.Tx[storage.Library], j job, on On) error {
	ids, err := db.New(tx).ListCopying(ctx, db.ListCopyingParams{NodeID: j.Node, SourceFileID: j.File})
	if err != nil {
		return fmt.Errorf("read the Conversions copying the source: %w", err)
	}
	for _, id := range ids {
		if err := c.advance(ctx, tx, id, Event{On: on, Node: j.Node}); err != nil {
			return err
		}
	}
	return nil
}

// transferred wakes the reconciler, which tells the Nodes the copies
// readied.
func (c *Conversions) transferred([]string) { c.cfg.Poke.Wake() }

// jobNames names each job by its Work, as the Conversion's row keeps it.
func (c *Conversions) jobNames(ctx context.Context, rd storage.Reader[storage.Library], jobs []string) (map[string]string, error) {
	q := db.New(rd)
	out := map[string]string{}
	for _, name := range jobs {
		j, err := parseJob(name)
		if err != nil {
			continue
		}
		id := j.Conversion
		if !j.Output {
			open, err := q.ListOpenOfSource(ctx, j.File)
			if err != nil {
				return nil, err
			}
			if len(open) == 0 {
				continue
			}
			id = open[0].ID
		}
		row, err := readRow(ctx, q, id)
		if err != nil {
			continue
		}
		out[name] = row.Snapshot.Work
	}
	return out, nil
}
