package moves

import (
	"context"
	"fmt"

	"github.com/jatassi/slipstream/internal/history/entry"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// MovedFacts is a Work moved: the Work, its roots in words, and its
// files.
type MovedFacts struct {
	Work  entry.Work `json:"work"`
	From  string     `json:"from"`
	To    string     `json:"to"`
	Files int        `json:"files"`
	Bytes int64      `json:"bytes"`
}

// FailedFacts is a Move that failed: its subject, the phase it failed in
// and why.
type FailedFacts struct {
	Work  entry.Work  `json:"work"`
	Phase Phase       `json:"phase"`
	Code  FailureCode `json:"code"`
	Words string      `json:"words"`
}

func (f MovedFacts) About() entry.About {
	return entry.About{Type: f.Work.Type, Works: []works.WorkID{f.Work.ID}}
}

func (f FailedFacts) About() entry.About {
	return entry.About{Type: f.Work.Type, Works: []works.WorkID{f.Work.ID}}
}

var (
	movedRow = entry.Row[MovedFacts]{
		Spec:   entry.Spec{Kind: "work_moved", Label: "Work moved", Group: entry.Moves, OnByDefault: true},
		Render: renderMoved,
	}
	failedRow = entry.Row[FailedFacts]{
		Spec:   entry.Spec{Kind: "move_failed", Label: "Move failed", Group: entry.Moves, OnByDefault: true, BellByDefault: true},
		Render: renderFailed,
	}
)

// Entries is the kinds moves records.
func Entries() []entry.Entry { return []entry.Entry{movedRow, failedRow} }

func renderMoved(f MovedFacts, _ entry.Context) entry.Message {
	return entry.Message{
		Title: "Moved " + f.Work.Label + " to the " + f.To,
		Lines: []string{fmt.Sprintf("%s moved from the %s.", filesWords(f.Files), f.From)},
		Link:  entry.WorkLink(f.Work.ID), Poster: f.Work.Poster, Tone: entry.Good,
	}
}

func renderFailed(f FailedFacts, _ entry.Context) entry.Message {
	return entry.Message{
		Title: "Couldn't move " + f.Work.Label, Lines: []string{f.Words},
		Link: entry.WorkLink(f.Work.ID), Poster: f.Work.Poster, Tone: entry.Fail,
	}
}

func filesWords(n int) string {
	if n == 1 {
		return "1 file"
	}
	return fmt.Sprintf("%d files", n)
}

// recordMoved records u's work_moved in tx, the switch's Write.
func (m *Moves) recordMoved(ctx context.Context, tx *storage.Tx[file], u Unit) error {
	w, err := entry.WorkIn(ctx, tx, m.cfg.Works, u.Subject.Work)
	if err != nil {
		return err
	}
	words, err := m.cfg.Roots.WordsIn(ctx, tx)
	if err != nil {
		return err
	}
	f := MovedFacts{Work: w, From: words[u.Subject.From.Root], To: words[u.Subject.To.Root]}
	for _, file := range u.Files {
		if file.Kind == KindLibrary {
			f.Files++
		}
		f.Bytes += file.Size
	}
	return movedRow.Record(ctx, tx, m.cfg.Recorder, f)
}

// recordFailed records u's move_failed in tx, the Write that fails it.
func (m *Moves) recordFailed(ctx context.Context, tx *storage.Tx[file], u Unit, s Failed) error {
	w, err := entry.WorkIn(ctx, tx, m.cfg.Works, u.Subject.Work)
	if err != nil {
		return err
	}
	return failedRow.Record(ctx, tx, m.cfg.Recorder, FailedFacts{Work: w, Phase: s.At, Code: s.Code, Words: failureWords(s.Code)})
}

// failureWords is a failure's sentence: what happened, then what to do.
func failureWords(c FailureCode) string {
	switch c {
	case SourceGone:
		return "Its files are gone from their root. Scan the library, then try again."
	case DestinationGone:
		return "The destination root is gone. Pick another root, then try again."
	case DestinationCheckFailed:
		return "Its files in the destination root didn't match the copies SlipStream placed. Try again."
	case SwitchRefused:
		return "Its files changed while it moved. Try again."
	case DestinationUnreachable:
		return "SlipStream couldn't reach the destination root. Check its Node, then try again."
	case Damaged:
		return "A copy didn't match its source. Try again."
	case SourceWrittenOff:
		return "An Admin had SlipStream stop waiting for the Node holding its files. Bring the Node back, then try again."
	case SourceFailed:
	}
	return "SlipStream couldn't read its files. Check their Node, then try again."
}
