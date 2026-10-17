package passkey_test

import (
	"bytes"
	"errors"
	"slices"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/identity/passkey"
	"github.com/jatassi/slipstream/internal/identity/passkey/passkeytest"
)

func TestDecide(t *testing.T) {
	for _, tc := range []struct {
		host    string
		https   bool
		verdict passkey.Verdict
		id      string
		origin  string
	}{
		{"home.localhost:8484", false, passkey.Verdict{Available: true}, "home.localhost", "http://home.localhost:8484"},
		{"Home.LocalHost:8484", false, passkey.Verdict{Available: true}, "home.localhost", "http://home.localhost:8484"},
		{"localhost:3000", false, passkey.Verdict{Available: true}, "localhost", "http://localhost:3000"},
		{"slipstream.example.com", true, passkey.Verdict{Available: true}, "slipstream.example.com", "https://slipstream.example.com"},
		{"slipstream.example.com:8443", true, passkey.Verdict{Available: true}, "slipstream.example.com", "https://slipstream.example.com:8443"},
		{"slipstream.example.com", false, passkey.Verdict{Reason: passkey.PlainHTTP}, "", ""},
		{"192.168.1.10:8484", true, passkey.Verdict{Reason: passkey.IPAddress}, "", ""},
		{"192.168.1.10", false, passkey.Verdict{Reason: passkey.IPAddress}, "", ""},
		{"[::1]:8484", true, passkey.Verdict{Reason: passkey.IPAddress}, "", ""},
		{"[fd00::10]", true, passkey.Verdict{Reason: passkey.IPAddress}, "", ""},
		{"nas", true, passkey.Verdict{Reason: passkey.NotADomain}, "", ""},
		{"nas.local.", true, passkey.Verdict{Reason: passkey.NotADomain}, "", ""},
	} {
		rp, v := passkey.Decide(tc.host, tc.https)
		if v != tc.verdict || rp.ID() != tc.id || rp.Origin() != tc.origin {
			t.Errorf("Decide(%q, %v) = %q %q %+v, want %q %q %+v", tc.host, tc.https, rp.ID(), rp.Origin(), v, tc.id, tc.origin, tc.verdict)
		}
	}
}

var (
	grace = passkey.User{Handle: bytes.Repeat([]byte{7}, 16), Name: "grace@example.com", DisplayName: "Grace"}
	ada   = passkey.User{Handle: bytes.Repeat([]byte{9}, 16), Name: "ada@example.com", DisplayName: "Ada"}
)

var onePassword = [16]byte{0xba, 0xda, 0x55, 0x66, 0xa7, 0xaa, 0x40, 0x1f, 0xbd, 0x96, 0x45, 0x61, 0x9a, 0x55, 0x12, 0x0d}

func mustRP(t *testing.T, host string, https bool) passkey.RP {
	t.Helper()
	rp, v := passkey.Decide(host, https)
	if !v.Available {
		t.Fatalf("Decide(%q, %v) = %+v", host, https, v)
	}
	return rp
}

func register(t *testing.T, c *passkey.Ceremonies, a *passkeytest.Authenticator, rp passkey.RP, u passkey.User) passkey.Credential {
	t.Helper()
	id, options, err := c.BeginRegistration(rp, u)
	if err != nil {
		t.Fatal(err)
	}
	response, err := a.Register(rp.Origin(), options)
	if err != nil {
		t.Fatal(err)
	}
	cred, err := c.FinishRegistration(rp, id, u, response)
	if err != nil {
		t.Fatalf("FinishRegistration: %v", err)
	}
	return cred
}

func signIn(c *passkey.Ceremonies, a *passkeytest.Authenticator, rp passkey.RP, users ...passkey.User) (passkey.Asserted, error) {
	id, options, err := c.BeginSignIn(rp)
	if err != nil {
		return passkey.Asserted{}, err
	}
	response, err := a.Assert(rp.Origin(), options)
	if err != nil {
		return passkey.Asserted{}, err
	}
	return c.FinishSignIn(rp, id, response, holding(users...))
}

func reconfirm(c *passkey.Ceremonies, a *passkeytest.Authenticator, rp passkey.RP, u passkey.User) (passkey.Asserted, error) {
	id, options, err := c.BeginReconfirm(rp, u)
	if err != nil {
		return passkey.Asserted{}, err
	}
	response, err := a.Assert(rp.Origin(), options)
	if err != nil {
		return passkey.Asserted{}, err
	}
	return c.FinishReconfirm(rp, id, u, response)
}

func TestRegisterThenAssert(t *testing.T) {
	t.Run("http://home.localhost:8484", func(t *testing.T) { registerThenAssert(t, mustRP(t, "home.localhost:8484", false)) })
	t.Run("https://slipstream.example.com", func(t *testing.T) { registerThenAssert(t, mustRP(t, "slipstream.example.com", true)) })
}

func registerThenAssert(t *testing.T, rp passkey.RP) {
	c := passkey.NewCeremonies()
	a := &passkeytest.Authenticator{AAGUID: onePassword}

	cred := register(t, c, a, rp, grace)
	if cred.AAGUID != onePassword || cred.SignCount != 0 || cred.BackupEligible || cred.BackupState ||
		!slices.Equal(cred.Transports, []string{"hybrid", "internal"}) || len(cred.ID) != 16 || len(cred.PublicKey) == 0 {
		t.Fatalf("registered %+v", cred)
	}
	if name, ok := passkey.ProviderName(cred.AAGUID); name != "1Password" || !ok {
		t.Errorf("ProviderName = %q %v, want 1Password", name, ok)
	}

	u := grace
	u.Credentials = []passkey.Credential{cred}
	got, err := signIn(c, a, rp, ada, u)
	if err != nil {
		t.Fatalf("sign-in: %v", err)
	}
	if !bytes.Equal(got.Handle, grace.Handle) || !bytes.Equal(got.Credential.ID, cred.ID) || got.Credential.SignCount != 1 {
		t.Fatalf("sign-in asserted %x count %d", got.Handle, got.Credential.SignCount)
	}

	u.Credentials = []passkey.Credential{got.Credential}
	again, err := reconfirm(c, a, rp, u)
	if err != nil {
		t.Fatalf("re-confirm: %v", err)
	}
	if again.Credential.SignCount != 2 || !slices.Equal(again.Credential.Transports, []string{"hybrid", "internal"}) {
		t.Fatalf("re-confirm asserted %+v", again.Credential)
	}
}

func TestRegistrationExcludesHeldPasskeys(t *testing.T) {
	c := passkey.NewCeremonies()
	a := &passkeytest.Authenticator{}
	rp := mustRP(t, "home.localhost:8484", false)
	u := grace
	u.Credentials = []passkey.Credential{register(t, c, a, rp, grace)}

	_, options, err := c.BeginRegistration(rp, u)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.Register(rp.Origin(), options); !errors.Is(err, passkeytest.ErrExcluded) {
		t.Fatalf("second registration on the same authenticator: %v, want ErrExcluded", err)
	}
	register(t, c, &passkeytest.Authenticator{}, rp, u)
}

func TestSignCount(t *testing.T) {
	c := passkey.NewCeremonies()
	rp := mustRP(t, "home.localhost:8484", false)

	a := &passkeytest.Authenticator{}
	u := grace
	u.Credentials = []passkey.Credential{register(t, c, a, rp, grace)}
	clone := a.Clone()
	for want := uint32(1); want <= 2; want++ {
		got, err := signIn(c, a, rp, u)
		if err != nil || got.Credential.SignCount != want {
			t.Fatalf("sign-in %d: count %d, %v", want, got.Credential.SignCount, err)
		}
		u.Credentials = []passkey.Credential{got.Credential}
	}
	if _, err := signIn(c, clone, rp, u); !errors.Is(err, passkey.ErrFailed) {
		t.Fatalf("a clone whose count is behind: %v, want ErrFailed", err)
	}

	synced := &passkeytest.Authenticator{Synced: true}
	s := ada
	s.Credentials = []passkey.Credential{register(t, c, synced, rp, ada)}
	if cred := s.Credentials[0]; !cred.BackupEligible || !cred.BackupState {
		t.Fatalf("synced registration flags %+v", cred)
	}
	for range 2 {
		got, err := signIn(c, synced, rp, s)
		if err != nil || got.Credential.SignCount != 0 || !got.Credential.BackupState {
			t.Fatalf("synced sign-in: %+v, %v", got.Credential, err)
		}
	}
}

func TestCeremonyRefusals(t *testing.T) {
	rp := mustRP(t, "home.localhost:8484", false)
	other := mustRP(t, "slipstream.example.com", true)
	c := passkey.NewCeremonies()
	a := &passkeytest.Authenticator{}
	u := grace
	u.Credentials = []passkey.Credential{register(t, c, a, rp, grace)}
	unverified := a.Clone()
	unverified.Unverified = true

	replayed, replay := started(t, c, a, rp, rp.Origin())
	if _, err := c.FinishSignIn(rp, replayed, replay, holding(u)); err != nil {
		t.Fatal(err)
	}
	wrongKind, wrongKindResponse := started(t, c, a, rp, rp.Origin())
	elsewhere, elsewhereResponse := started(t, c, a, rp, rp.Origin())
	evil, evilResponse := started(t, c, a, rp, "http://evil.localhost:8484")
	forAda, forAdaOptions, _ := c.BeginRegistration(rp, ada)
	forAdaResponse, _ := (&passkeytest.Authenticator{}).Register(rp.Origin(), forAdaOptions)

	for _, tc := range []struct {
		name string
		err  error
		want error
	}{
		{"replayed", second(c.FinishSignIn(rp, replayed, replay, holding(u))), passkey.ErrCeremony},
		{"wrong kind", second(c.FinishReconfirm(rp, wrongKind, u, wrongKindResponse)), passkey.ErrCeremony},
		{"wrong kind used it up", second(c.FinishSignIn(rp, wrongKind, wrongKindResponse, holding(u))), passkey.ErrCeremony},
		{"another address", second(c.FinishSignIn(other, elsewhere, elsewhereResponse, holding(u))), passkey.ErrCeremony},
		{"another user", second(c.FinishRegistration(rp, forAda, grace, forAdaResponse)), passkey.ErrCeremony},
		{"answered at another origin", second(c.FinishSignIn(rp, evil, evilResponse, holding(u))), passkey.ErrFailed},
		{"user not verified", second(signIn(c, unverified, rp, u)), passkey.ErrFailed},
		{"unknown handle", second(signIn(c, a, rp, ada)), passkey.ErrFailed},
		{"re-confirm without a passkey", third(c.BeginReconfirm(rp, ada)), passkey.ErrNoCredential},
	} {
		if !errors.Is(tc.err, tc.want) {
			t.Errorf("%s: %v, want %v", tc.name, tc.err, tc.want)
		}
	}
}

func TestNoCeremonyWhereNoPasskey(t *testing.T) {
	c := passkey.NewCeremonies()
	for _, host := range []string{"192.168.1.10:8484", "[::1]:8484", "slipstream.example.com"} {
		rp, v := passkey.Decide(host, false)
		if _, _, err := c.BeginSignIn(rp); v.Available || err == nil {
			t.Errorf("at %s: %+v, BeginSignIn %v", host, v, err)
		}
	}
}

func started(t *testing.T, c *passkey.Ceremonies, a *passkeytest.Authenticator, rp passkey.RP, origin string) (id passkey.CeremonyID, response []byte) {
	t.Helper()
	id, options, err := c.BeginSignIn(rp)
	if err != nil {
		t.Fatal(err)
	}
	if response, err = a.Assert(origin, options); err != nil {
		t.Fatal(err)
	}
	return id, response
}

func holding(users ...passkey.User) func([]byte) (passkey.User, bool, error) {
	return func(handle []byte) (passkey.User, bool, error) {
		i := slices.IndexFunc(users, func(u passkey.User) bool { return bytes.Equal(u.Handle, handle) })
		if i < 0 {
			return passkey.User{}, false, nil
		}
		return users[i], true, nil
	}
}

func second[T any](_ T, err error) error { return err }

func third[T, U any](_ T, _ U, err error) error { return err }

func TestCeremonyExpires(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		rp := mustRP(t, "home.localhost:8484", false)
		c := passkey.NewCeremonies()
		a := &passkeytest.Authenticator{}
		u := grace
		u.Credentials = []passkey.Credential{register(t, c, a, rp, grace)}

		early, options, _ := c.BeginSignIn(rp)
		earlyResponse, _ := a.Assert(rp.Origin(), options)
		late, options, _ := c.BeginSignIn(rp)
		lateResponse, _ := a.Assert(rp.Origin(), options)

		time.Sleep(passkey.Lifetime - time.Second)
		if _, err := c.FinishSignIn(rp, early, earlyResponse, holding(u)); err != nil {
			t.Fatalf("finished just inside the lifetime: %v", err)
		}
		time.Sleep(time.Second)
		if _, err := c.FinishSignIn(rp, late, lateResponse, holding(u)); !errors.Is(err, passkey.ErrCeremony) {
			t.Fatalf("finished at the lifetime: %v, want ErrCeremony", err)
		}
	})
}

func TestCeremoniesBounded(t *testing.T) {
	rp := mustRP(t, "home.localhost:8484", false)
	c := passkey.NewCeremonies()
	a := &passkeytest.Authenticator{}
	u := grace
	u.Credentials = []passkey.Credential{register(t, c, a, rp, grace)}

	first, options, _ := c.BeginSignIn(rp)
	firstResponse, _ := a.Assert(rp.Origin(), options)
	second, options, _ := c.BeginSignIn(rp)
	secondResponse, _ := a.Assert(rp.Origin(), options)
	for range 1023 {
		if _, _, err := c.BeginSignIn(rp); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := c.FinishSignIn(rp, first, firstResponse, holding(u)); !errors.Is(err, passkey.ErrCeremony) {
		t.Fatalf("the oldest past the bound: %v, want ErrCeremony", err)
	}
	if _, err := c.FinishSignIn(rp, second, secondResponse, holding(u)); err != nil {
		t.Fatalf("the next oldest: %v", err)
	}
}

func TestProviderNameUnknown(t *testing.T) {
	if name, ok := passkey.ProviderName([16]byte{}); ok {
		t.Fatalf("zero AAGUID named %q", name)
	}
}
