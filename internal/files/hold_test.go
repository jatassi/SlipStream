package files_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
)

func TestAHoldEditRefillsNothingAndHasNoImpact(t *testing.T) {
	s := shelved(t)
	before := s.snapshot(t, s.svc, s.tree.ID)
	held := s.hd.Spec()
	held.Hold = 2 * time.Hour
	for _, c := range []struct {
		name   string
		change targets.Change
	}{
		{"a Hold", targets.SetHold{ID: s.hd.ID, Hold: 90 * time.Minute}},
		{"a Target edit of only its Hold", targets.UpdateTarget{ID: s.hd.ID, Spec: held}},
	} {
		var touched targets.Touched
		err := s.lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
			var err error
			touched, err = s.targets.Touches(ctx, r, c.change)
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(touched.Works) != 0 {
			t.Errorf("%s touches %v, want no Work", c.name, touched.Works)
		}
		impact := s.change(t, s.svc, c.change, deleting)
		if want := "works=0 ownList=0 unwanted=0 bytes=0 renamed=0 newlyWanted=0 downloading=0"; shown(impact) != want {
			t.Errorf("%s: Impact\n%s\nwant\n%s", c.name, shown(impact), want)
		}
		if after := s.snapshot(t, s.svc, s.tree.ID); after != before {
			t.Errorf("%s changed\n%s\nto\n%s", c.name, before, after)
		}
	}
	got, err := s.targets.GetTarget(t.Context(), admin, s.hd.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Hold != 2*time.Hour {
		t.Errorf("the Target's Hold reads %v, want 2h0m0s", got.Hold)
	}

	for _, hold := range []time.Duration{-time.Minute, 7*24*time.Hour + time.Minute, 90 * time.Second} {
		_, err := s.svc.Change(t.Context(), admin, targets.SetHold{ID: s.hd.ID, Hold: hold}, files.Mode{Unwanted: files.Delete})
		if !errors.Is(err, targets.ErrBadHold) {
			t.Errorf("a Hold of %v: %v, want ErrBadHold", hold, err)
		}
	}
}
