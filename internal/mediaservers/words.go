package mediaservers

import (
	"errors"
	"fmt"
	"net/url"

	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/setting"
)

// Words is what a person reads for one of the Service's errors, the
// problem's detail and a route test's words alike. It is false for any
// other error.
func Words(err error) (string, bool) {
	var (
		old      *TooOldError
		other    *OtherServerError
		away     *UnreachedError
		refused  *SignInRefusedError
		local    *LocalhostOnWorkerError
		unlinked *NodeNotLinkedError
		short    *MarginTooShortError
	)
	switch {
	case errors.As(err, &old):
		return fmt.Sprintf("%s %s is older than SlipStream supports. Update it to %s or later, then save again.", old.Label, old.Found, old.Floor), true
	case errors.As(err, &other):
		return fmt.Sprintf("A different %s server responds at %s. Check the address, or confirm the new server under Details.", other.Label, hostOf(other.Address)), true
	case errors.As(err, &away):
		return fmt.Sprintf("%s at %s didn't respond to %s. Check that %s is running and that %s can reach %s.",
			away.Label, away.Address, away.Node, away.Label, away.Node, hostOf(away.Address)), true
	case errors.As(err, &refused):
		return refusedWords(refused), true
	case errors.As(err, &local):
		return fmt.Sprintf("The Controller can't use %s, because localhost on %s is %s itself. Enter %s's network address, or a fallback address the Controller can reach.",
			local.Address, local.Node, local.Node, local.Node), true
	case errors.As(err, &unlinked):
		return fmt.Sprintf("%s doesn't hold Media Server link. Tick the Role on %s, or reach the server from the Controller.", unlinked.Node, unlinked.Node), true
	case errors.As(err, &short):
		return fmt.Sprintf("%q must be longer than %q, which is %s. Lengthen the margin or renew more often.",
			RenewMargin.Label, RenewEvery.Label, setting.Words(short.Cadence)), true
	}
	return "", false
}

func refusedWords(e *SignInRefusedError) string {
	switch e.With {
	case RefusedPlex:
		return "SlipStream's sign-in to " + e.Server + " has expired or was revoked. Sign in with Plex again."
	case RefusedKey:
		return e.Server + " refused the API key. Paste a new key, or sign in with an administrator's name and password."
	case RefusedPassword:
	}
	return e.Server + " refused that name and password. Check them and sign in again."
}

// hostOf is an address's host and port, as the copy names a server:
// 192.168.1.20:8096.
func hostOf(address string) string {
	u, err := url.Parse(address)
	if err != nil || u.Host == "" {
		return address
	}
	return u.Host
}

// failure is a route test's error as the Admin reads it: the server's own
// response, or no response at all from a Node that didn't get one.
func failure(err error, r route) error {
	p, ok := link.ServerProblem(err)
	if !ok {
		return &UnreachedError{Label: r.label, Node: r.name, Address: r.address}
	}
	switch p.Reason {
	case link.ReasonRefused:
		with := RefusedKey
		if r.kind.About().Form == link.SignInPlexPIN {
			with = RefusedPlex
		}
		return &SignInRefusedError{Server: r.server, With: with}
	case link.ReasonOtherServer:
		return &OtherServerError{Label: r.label, Address: r.address}
	case link.ReasonUnreachable, link.ReasonUnsupported, link.ReasonFailed:
	}
	return &UnreachedError{Label: r.label, Node: r.name, Address: r.address}
}
