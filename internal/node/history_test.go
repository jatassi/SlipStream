package node

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"testing/synctest"

	"github.com/jatassi/slipstream/internal/history/entry"
)

type lateFacts struct{}

func (lateFacts) About() entry.About { return entry.About{} }

func TestTheHistoryCatalogIsSealedBeforeTheControllerRunsSoALateRecorderPanics(t *testing.T) {
	env, _ := testEnv(t)
	synctest.Test(t, func(t *testing.T) {
		n, c := startInMemory(t, env)
		if claimed := c.claim(n.Ready().SetupCode, "1234"); claimed.status != http.StatusCreated {
			t.Fatalf("claimController = %d %s", claimed.status, claimed.body)
		}
		synctest.Wait()
		defer func() {
			if r := fmt.Sprint(recover()); !strings.Contains(r, "after the catalog was sealed") {
				t.Fatalf("a Recorder after the Controller started panicked with %q, want the sealed catalog's refusal", r)
			}
		}()
		n.history.Recorder("late", []entry.Entry{entry.Row[lateFacts]{Spec: entry.Spec{Kind: "late", Label: "Late", Group: entry.Library}}})
	})
}
