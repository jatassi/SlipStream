package subtitles

import (
	"context"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/naming"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/subtitles/internal/db"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// TestTheHotReadsReadThroughTheirIndexes pins each read the drains, the
// Keeper, the change Write and the Library filters run to its index, with
// no table scan and no sort.
func TestTheHotReadsReadThroughTheirIndexes(t *testing.T) {
	lib := storagetest.Library(t, identity.Schema(), nodes.Schema(), works.Schema(), targets.Schema(), roots.Schema(), downloadclients.Schema(), grabs.Schema(), indexers.Schema(), files.Schema(), Schema(), naming.Schema())
	ids := []files.ID{{UUID: storage.NewUUID()}, {UUID: storage.NewUUID()}}
	profile := targets.ProfileID{UUID: storage.NewUUID()}
	for _, s := range []struct {
		name    string
		indexes []string
		read    func(ctx context.Context, q *db.Queries) error
	}{
		{"NextQueued", []string{"subtitles_want_queue"}, func(ctx context.Context, q *db.Queries) error {
			_, err := q.NextQueued(ctx, db.NextQueuedParams{Kind: "episode", N: 10})
			return err
		}},
		{"WaitingCount", []string{"subtitles_want_queue"}, func(ctx context.Context, q *db.Queries) error {
			_, err := q.WaitingCount(ctx, "feature")
			return err
		}},
		{"PastDue", []string{"subtitles_want_due"}, func(ctx context.Context, q *db.Queries) error {
			_, err := q.PastDue(ctx, db.PastDueParams{Now: storage.Time{}, N: 10})
			return err
		}},
		{"LibraryMarks", []string{"COVERING INDEX subtitles_want_unmet"}, func(ctx context.Context, q *db.Queries) error {
			_, err := q.ListLibraryMarks(ctx)
			return err
		}},
		{"MissingCount", []string{"subtitles_want_unmet"}, func(ctx context.Context, q *db.Queries) error {
			_, err := q.MissingCount(ctx)
			return err
		}},
		{"ImpactGained", []string{"targets_wants_profile_id", "files_file_work_id"}, func(ctx context.Context, q *db.Queries) error {
			_, err := q.ImpactGained(ctx, db.ImpactGainedParams{ProfileID: profile, Key: ",en,", Base: ",en,", Language: "en", NoAudio: 1})
			return err
		}},
		{"ImpactDropped", []string{"subtitles_sidecar_language"}, func(ctx context.Context, q *db.Queries) error {
			_, err := q.ImpactDropped(ctx, db.ImpactDroppedParams{Language: "en", ProfileID: profile})
			return err
		}},
		{"ListDropped", []string{"subtitles_sidecar_language"}, func(ctx context.Context, q *db.Queries) error {
			_, err := q.ListDropped(ctx, db.ListDroppedParams{Language: "en", ProfileID: profile})
			return err
		}},
		{"ListSurveyDue", []string{"subtitles_file_survey_due"}, func(ctx context.Context, q *db.Queries) error {
			_, err := q.ListSurveyDue(ctx, 10)
			return err
		}},
		{"ListPendingFiles", []string{"subtitles_sidecar_pending"}, func(ctx context.Context, q *db.Queries) error {
			_, err := q.ListPendingFiles(ctx, 10)
			return err
		}},
		{"ListParkedFiles", []string{"subtitles_parked_from"}, func(ctx context.Context, q *db.Queries) error {
			_, err := q.ListParkedFiles(ctx, 10)
			return err
		}},
		{"ListRemovalFiles", []string{"subtitles_removal_next"}, func(ctx context.Context, q *db.Queries) error {
			_, err := q.ListRemovalFiles(ctx, db.ListRemovalFilesParams{Now: storage.Time{}, N: 10})
			return err
		}},
		{"ListRemovalsDue", []string{"subtitles_removal_file"}, func(ctx context.Context, q *db.Queries) error {
			_, err := q.ListRemovalsDue(ctx, db.ListRemovalsDueParams{FileID: ids[0], Now: storage.Time{}})
			return err
		}},
		{"ListIntentFiles", nil, func(ctx context.Context, q *db.Queries) error {
			_, err := q.ListIntentFiles(ctx, 10)
			return err
		}},
		{"ListWantsIn", nil, func(ctx context.Context, q *db.Queries) error {
			_, err := q.ListWantsIn(ctx, ids)
			return err
		}},
		{"ListSidecarsIn", nil, func(ctx context.Context, q *db.Queries) error {
			_, err := q.ListSidecarsIn(ctx, ids)
			return err
		}},
		{"ListIntentsIn", nil, func(ctx context.Context, q *db.Queries) error {
			_, err := q.ListIntentsIn(ctx, ids)
			return err
		}},
		{"ListIgnoredIn", nil, func(ctx context.Context, q *db.Queries) error {
			_, err := q.ListIgnoredIn(ctx, ids)
			return err
		}},
		{"ListFilesIn", nil, func(ctx context.Context, q *db.Queries) error {
			_, err := q.ListFilesIn(ctx, ids)
			return err
		}},
	} {
		err := lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
			plans := storagetest.Plan(t, r, func(r storage.Reader[storage.Library]) error { return s.read(ctx, db.New(r)) })
			for _, p := range plans {
				for _, index := range s.indexes {
					if !strings.Contains(strings.Join(p.Lines, "\n"), index) {
						t.Errorf("%s doesn't read through %s: %q", s.name, index, p.Lines)
					}
				}
				for _, l := range p.Lines {
					if strings.HasPrefix(l, "SCAN") && !strings.Contains(l, "INDEX") || strings.Contains(l, "USE TEMP B-TREE") {
						t.Errorf("%s plans %q", s.name, p.Lines)
					}
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}
