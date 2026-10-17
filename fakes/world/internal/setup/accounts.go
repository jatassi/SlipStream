package setup

import (
	"context"
	"fmt"
	"net/http"
	"net/netip"
	"strings"

	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/world/internal/nodeapi"
	"github.com/jatassi/slipstream/internal/node"
)

// Account is a Scenario Member as the world made it.
type Account struct {
	ID string
	scenario.Member
}

// Accounts makes the Scenario's Members and email setting on its
// controller Node, over c with the Controller's API key, as an Admin and
// each person would. It sets the address links use to the Controller's
// browser URL, then for each Member issues a Signup link, redeems it as
// the link's page does and signs that session out, and makes the Member
// an Admin when the Scenario says so. Only then, with mail, does it save
// smtp as the email setting, so the launch sends no email to the capture.
func Accounts(ctx context.Context, c *http.Client, s scenario.Scenario, ready []node.Ready, apiKey string, smtp netip.AddrPort) ([]Account, error) {
	var base, host, address string
	for i, n := range s.Nodes {
		if n.Setup == scenario.SetupController {
			base, host, address = url(ready[i]), Host(n.Name), BrowserURL(n.Name, ready[i])
		}
	}
	accounts := make([]Account, len(s.Members))
	if len(s.Members) > 0 {
		if err := nodeapi.SetAddress(ctx, c, base, apiKey, address); err != nil {
			return nil, err
		}
	}
	for i, m := range s.Members {
		id, err := redeem(ctx, c, base, apiKey, m)
		if err != nil {
			return nil, err
		}
		if m.Admin {
			if err := nodeapi.MakeAdmin(ctx, c, base, apiKey, id); err != nil {
				return nil, err
			}
		}
		accounts[i] = Account{ID: id, Member: m}
	}
	if s.Mail {
		mail := nodeapi.Mail{Host: smtp.Addr().String(), Port: smtp.Port(), Security: "none", FromAddress: "slipstream@" + host, FromName: "SlipStream"}
		if err := nodeapi.SetMail(ctx, c, base, apiKey, mail); err != nil {
			return nil, err
		}
	}
	return accounts, nil
}

func redeem(ctx context.Context, c *http.Client, base, apiKey string, m scenario.Member) (string, error) {
	link, err := nodeapi.CreateSignupLink(ctx, c, base, apiKey, m.Email)
	if err != nil {
		return "", err
	}
	token, err := linkToken(link)
	if err != nil {
		return "", err
	}
	id, cookie, err := nodeapi.RedeemSignupLink(ctx, c, base, token, m.DisplayName, m.PIN)
	if err != nil {
		return "", err
	}
	return id, nodeapi.SignOut(ctx, c, base, cookie)
}

// linkToken is the token in a link's fragment, where the page reads it.
func linkToken(link string) (string, error) {
	_, token, ok := strings.Cut(link, "#token=")
	if !ok || token == "" {
		return "", fmt.Errorf("signup link %s has no token in its fragment", link)
	}
	return token, nil
}
