package indexerstest_test

import (
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/indexers/indexerstest"
	"github.com/jatassi/slipstream/internal/searching"
	"github.com/jatassi/slipstream/internal/storage"
)

func TestNodeAnswersEachCallKindAndOffsetByItsOwnScript(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		id := indexers.ID{UUID: storage.NewUUID()}
		var n indexerstest.Node
		n.Answer(id, 0, searching.Answer{Releases: []searching.Release{{Title: "any call"}}})
		n.AnswerCall(id, indexerstest.Searches, 0, searching.Answer{Releases: []searching.Release{{Title: "a search"}}})
		n.AnswerCall(id, indexerstest.Recent(0), 0, searching.Answer{Releases: []searching.Release{{Title: "the newest page"}}})
		n.AnswerCall(id, indexerstest.Recent(100), time.Minute, searching.Answer{Releases: []searching.Release{{Title: "the second page"}}})
		for i, tc := range []struct {
			call searching.Call
			want string
		}{
			{searching.SearchCall{Query: searching.Query{Mode: "search", Text: "orchard"}}, "a search"},
			{searching.RecentCall{}, "the newest page"},
			{searching.RecentCall{Offset: 200}, "any call"},
			{searching.TestCall{}, "any call"},
		} {
			poll, err := n.Run(t.Context(), searching.Job{ID: string(rune('a' + i)), Indexer: id.String(), Call: tc.call})
			if err != nil || poll.Answer == nil || poll.Answer.Releases[0].Title != tc.want {
				t.Fatalf("Run of %T = %+v, %v; want %q", tc.call, poll, err, tc.want)
			}
		}
		late := searching.Job{ID: "late", Indexer: id.String(), Call: searching.RecentCall{Offset: 100}}
		start := time.Now()
		for {
			poll, err := n.Run(t.Context(), late)
			if err != nil {
				t.Fatal(err)
			}
			if poll.Answer != nil {
				if poll.Answer.Releases[0].Title != "the second page" || time.Since(start) != time.Minute {
					t.Fatalf("the second page = %+v after %s, want its own script a minute after the first Run", poll.Answer, time.Since(start))
				}
				return
			}
		}
	})
}
