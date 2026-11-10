package grabs

import "github.com/jatassi/slipstream/internal/identity"

// Actor is who did something: an Admin's account and name, or the API
// key, Account nil and Name empty. The name is kept beside the account,
// since an account can be deleted.
type Actor struct {
	Account *identity.AccountID
	Name    string
}

// Words is the Admin's name, or "the API key".
func (a Actor) Words() string {
	if a.Name == "" {
		return "the API key"
	}
	return a.Name
}
