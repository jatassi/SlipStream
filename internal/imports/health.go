package imports

import (
	"context"
	"fmt"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/health"
	"github.com/jatassi/slipstream/internal/imports/internal/db"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/scanning"
	"github.com/jatassi/slipstream/internal/storage"
)

var HealthAbout = health.About{
	ID: health.LibraryID, Label: "Library", Phrase: "the library",
	Events: []live.Event{files.ChangedEvent, ChangedEvent, roots.ChangedEvent},
}

// HealthSource answers, per root, a Warn item for its gone files, a Fail
// item while its last comparison stopped, a Warn item for the files
// ffprobe can't read, and a Warn item for each fresh import put back in
// its holding folder, each named by the root's words. A root's own state
// stays in Storage, and a scan that fails or skips for another reason in
// Scheduled tasks.
func (im *Imports) HealthSource() health.Source {
	return health.Source{Answer: im.healthAnswer}
}

// healthRead is what Library Health reads of every root.
type healthRead struct {
	all        []roots.Root
	gone       map[roots.ID]files.Tally
	unreadable map[roots.ID]files.Tally
	putBack    []files.PutBack
	compared   []db.ImportsRoot
}

func (im *Imports) healthAnswer(ctx context.Context) (health.Answer, error) {
	words, err := im.cfg.Roots.Words(ctx)
	if err != nil {
		return health.Answer{}, err
	}
	var h healthRead
	err = im.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		var err error
		if h.all, err = roots.All(ctx, r); err != nil {
			return err
		}
		if h.gone, err = files.GoneOf(ctx, r); err != nil {
			return err
		}
		if h.unreadable, err = files.UnreadableOf(ctx, r); err != nil {
			return err
		}
		if h.putBack, err = im.cfg.Files.PutBacks(ctx, r); err != nil {
			return err
		}
		h.compared, err = db.New(r).ListCompared(ctx)
		return err
	})
	if err != nil {
		return health.Answer{}, err
	}
	var a health.Answer
	for _, root := range h.all {
		d, _ := im.cfg.Works.Describe(root.ContentType)
		a.Items = append(a.Items, h.itemsOf(root, words[root.ID], d.Work.Many)...)
	}
	return a, nil
}

// itemsOf is root's items, its subject its words and works what its
// library holds.
func (h healthRead) itemsOf(root roots.Root, subject, works string) []health.Item {
	var out []health.Item
	for _, c := range h.compared {
		if c.RootID == root.ID && c.Place == root.Place && c.StoppedGone != nil {
			out = append(out, stoppedItem(root, subject, scanning.Stopped{Gone: int(*c.StoppedGone), Of: int(*c.StoppedOf)}))
		}
	}
	if t, ok := h.gone[root.ID]; ok {
		out = append(out, goneItem(root, subject, t))
	}
	if t, ok := h.unreadable[root.ID]; ok {
		out = append(out, unreadableItem(root, subject, t))
	}
	for _, p := range h.putBack {
		if p.Root == root.ID {
			out = append(out, putBackItem(subject, works, p))
		}
	}
	return out
}

// putBackItem is the Warn item of p, imported into a library of works:
// "A file imported into Movies went back to its holding folder because
// another file already has its name, Heat (1995).mkv. Rename or remove
// that file." It opens the Work, whose files list it.
func putBackItem(subject, works string, p files.PutBack) health.Item {
	return health.Item{Key: health.KeyOf("put_back", p.File.String()), Level: health.Warn, Says: health.Says{
		Subject: subject,
		Text:    "A file imported into " + works + " went back to its holding folder because another file already has its name, " + p.Name + ". Rename or remove that file.",
		Action:  health.LinkAction("/works/"+p.Work.String(), "Open the Work"),
	}}
}

func stoppedItem(root roots.Root, subject string, s scanning.Stopped) health.Item {
	return health.Item{Key: health.KeyOf("stopped", root.ID.String()), Level: health.Fail, Says: health.Says{
		Subject: subject, Text: stoppedWords(s),
		Action: health.LinkAction("/settings/library-roots#root-"+root.ID.String(), "Open the root"),
	}}
}

func goneItem(root roots.Root, subject string, t files.Tally) health.Item {
	text := count(t.Count, "file", "files") + " are gone. Restore them, or forget them on each Work's page."
	if t.Count == 1 {
		text = "1 file is gone. Restore it, or forget it on its Work's page."
	}
	detail := make([]health.Fact, len(t.First))
	for i, p := range t.First {
		detail[i] = health.Fact{Label: "File", Text: p.Path}
	}
	return health.Item{Key: health.KeyOf("gone", root.ID.String()), Level: health.Warn, Says: health.Says{
		Subject: subject, Text: text, Detail: detail,
		Action: health.LinkAction("/library/"+string(root.ContentType)+"?root="+root.ID.String()+"&gone=1", "Show the Works"),
	}}
}

func unreadableItem(root roots.Root, subject string, t files.Tally) health.Item {
	text := fmt.Sprintf("ffprobe can't read %s. Replace them, then scan again.", count(t.Count, "file", "files"))
	if t.Count == 1 {
		text = "ffprobe can't read 1 file. Replace it, then scan again."
	}
	detail := make([]health.Fact, 0, 2*len(t.First))
	for _, p := range t.First {
		detail = append(detail, health.Fact{Label: "File", Text: p.Path}, health.Fact{Label: "Error", Text: p.Line})
	}
	return health.Item{Key: health.KeyOf("unreadable", root.ID.String()), Level: health.Warn, Says: health.Says{Subject: subject, Text: text, Detail: detail}}
}
