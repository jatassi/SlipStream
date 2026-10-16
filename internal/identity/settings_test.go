package identity

import (
	"context"
	"testing"

	"github.com/jatassi/slipstream/internal/identity/internal/db"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
)

func TestTheTableRefusesWhatBoundsRefuse(t *testing.T) {
	lib := storagetest.Library(t, Schema())
	fields := map[string]func(*db.SaveSettingsParams) *int64{
		"pinWaitAfter":              func(p *db.SaveSettingsParams) *int64 { return &p.PinWaitAfter },
		"pinWaitSeconds":            func(p *db.SaveSettingsParams) *int64 { return &p.PinWaitSeconds },
		"pinOffAfter":               func(p *db.SaveSettingsParams) *int64 { return &p.PinOffAfter },
		"addressFailures":           func(p *db.SaveSettingsParams) *int64 { return &p.AddressFailures },
		"addressWindowSeconds":      func(p *db.SaveSettingsParams) *int64 { return &p.AddressWindowSeconds },
		"reconfirmWindowSeconds":    func(p *db.SaveSettingsParams) *int64 { return &p.ReconfirmWindowSeconds },
		"sessionLifetimeSeconds":    func(p *db.SaveSettingsParams) *int64 { return &p.SessionLifetimeSeconds },
		"signupLinkLifetimeSeconds": func(p *db.SaveSettingsParams) *int64 { return &p.SignupLinkLifetimeSeconds },
		"resetLinkLifetimeSeconds":  func(p *db.SaveSettingsParams) *int64 { return &p.ResetLinkLifetimeSeconds },
		"emailLinkLifetimeSeconds":  func(p *db.SaveSettingsParams) *int64 { return &p.EmailLinkLifetimeSeconds },
	}
	defaults := func() db.SaveSettingsParams {
		var p db.SaveSettingsParams
		for _, b := range Bounds {
			*fields[b.Field](&p) = b.Default
		}
		return p
	}
	save := func(p db.SaveSettingsParams) error {
		return lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[file]) error {
			return db.New(tx).SaveSettings(ctx, p)
		})
	}
	if len(fields) != len(Bounds) {
		t.Fatalf("%d fields for %d Bounds", len(fields), len(Bounds))
	}
	for _, b := range Bounds {
		for v, ok := range map[int64]bool{b.Floor - 1: false, b.Floor: true, b.Ceiling: true, b.Ceiling + 1: false} {
			p := defaults()
			switch b.Field {
			case "pinWaitAfter":
				p.PinOffAfter = 20
			case "pinOffAfter":
				p.PinWaitAfter = 3
			}
			*fields[b.Field](&p) = v
			if err := save(p); (err == nil) != ok {
				t.Errorf("%s = %d saved with %v, want accepted %v", b.Field, v, err, ok)
			}
		}
	}
	p := defaults()
	p.PinOffAfter = p.PinWaitAfter
	if err := save(p); err == nil {
		t.Error("pinOffAfter equal to pinWaitAfter was saved")
	}
}
