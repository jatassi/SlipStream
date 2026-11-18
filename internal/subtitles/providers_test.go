package subtitles_test

import (
	"errors"
	"slices"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/subtitles"
	"github.com/jatassi/slipstream/internal/subtitles/subtitlestest"
	"github.com/jatassi/slipstream/internal/subtitling"
)

func order(t *testing.T, w *subtitlestest.World) []subtitling.ProviderID {
	t.Helper()
	ps, err := w.Subs.Providers(t.Context(), identity.Admin{})
	if err != nil {
		t.Fatal(err)
	}
	out := make([]subtitling.ProviderID, len(ps))
	for i, p := range ps {
		out[i] = p.Spec.ID
	}
	return out
}

func TestProvidersReorderAndTurnOffAndOn(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := world(t, nil)
		admin := identity.Admin{}
		if got := order(t, w); !slices.Equal(got, []subtitling.ProviderID{"subdl", "gestdown"}) {
			t.Fatalf("the default order is %v", got)
		}
		if _, err := w.Subs.OrderProviders(t.Context(), admin, []subtitling.ProviderID{"gestdown", "subdl"}); err != nil {
			t.Fatal(err)
		}
		if got := order(t, w); !slices.Equal(got, []subtitling.ProviderID{"gestdown", "subdl"}) {
			t.Fatalf("after the reorder the order is %v", got)
		}
		var mismatch *subtitles.OrderMismatchError
		if _, err := w.Subs.OrderProviders(t.Context(), admin, []subtitling.ProviderID{"gestdown"}); !errors.As(err, &mismatch) {
			t.Fatalf("an order missing SubDL = %v, want OrderMismatchError", err)
		}
		off, err := w.Subs.SetProviderOn(t.Context(), admin, "gestdown", false)
		if err != nil || off.Standing(time.Now()) != subtitles.Off {
			t.Fatalf("Gestdown turned off reads %+v, %v", off, err)
		}
		on, err := w.Subs.SetProviderOn(t.Context(), admin, "gestdown", true)
		if err != nil || on.Standing(time.Now()) != subtitles.Ready {
			t.Fatalf("Gestdown turned on reads %+v, %v", on, err)
		}
		if gained := query(t, w, `SELECT id FROM subtitles_provider WHERE gained_at IS NOT NULL`); !slices.Equal(gained, []string{"gestdown"}) {
			t.Fatalf("providers that gained the power to search: %v, want gestdown", gained)
		}
		if _, err := w.Subs.SetProviderOn(t.Context(), admin, "opensubtitles", true); !errors.Is(err, subtitles.ErrNoProvider) {
			t.Fatalf("an unknown provider = %v", err)
		}
	})
}

func TestAKeyIsCheckedOnAHolderBeforeItIsStored(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := world(t, nil)
		admin := identity.Admin{}
		w.Holders(false)
		if _, err := w.Subs.SetKey(t.Context(), admin, "subdl", "k1"); !errors.Is(err, subtitles.ErrNoHolder) {
			t.Fatalf("a key with no holder = %v, want ErrNoHolder", err)
		}
		w.Holders(true)
		w.Holder = subtitlestest.KeyHolder{Refused: "Not Authorized"}
		_, err := w.Subs.SetKey(t.Context(), admin, "subdl", "k1")
		var refused *subtitles.KeyRefusedError
		if !errors.As(err, &refused) || refused.Provider != "SubDL" || refused.Words != "Not Authorized" {
			t.Fatalf("a refused key = %v, want SubDL's own words", err)
		}
		if keys := query(t, w, `SELECT id FROM subtitles_provider WHERE api_key_sealed IS NOT NULL`); len(keys) != 0 {
			t.Fatalf("a refused key was stored for %v", keys)
		}
		w.Holder = subtitlestest.KeyHolder{Account: "jsmith", Plan: "free"}
		p, err := w.Subs.SetKey(t.Context(), admin, "subdl", "k2")
		if err != nil {
			t.Fatal(err)
		}
		if !p.HasKey || p.Account != "jsmith" || p.Plan != "free" || p.Allowance == nil || p.Allowance.Downloads.Remaining != 19 || p.Standing(time.Now()) != subtitles.Ready {
			t.Fatalf("an accepted key reads %+v", p)
		}
		if _, err := w.Subs.SetKey(t.Context(), admin, "gestdown", "k3"); !errors.Is(err, subtitles.ErrKeyless) {
			t.Fatalf("a key for Gestdown = %v, want ErrKeyless", err)
		}
		p, err = w.Subs.DeleteKey(t.Context(), admin, "subdl")
		if err != nil || p.HasKey || p.Account != "" || p.Standing(time.Now()) != subtitles.NeedsKey {
			t.Fatalf("a deleted key reads %+v, %v", p, err)
		}
	})
}
