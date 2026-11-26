package link

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
)

// Users is a server's own user accounts, on a kind whose About.Access is
// users. Every call converges: deleting a deleted user and
// disabling a disabled one succeed.
type Users interface {
	// CreateUser makes a user named name with a random password made where
	// the call runs, and returns its id.
	CreateUser(ctx context.Context, name string) (string, error)
	// Restrict posts the user's whole policy back with IsAdministrator
	// false, IsDisabled false, EnableAllFolders false and EnabledFolders the
	// libraries, and every other field as the server had it. Only
	// EnabledFolders hides a library: BlockedMediaFolders hides nothing on
	// either product (online run 2026-11-26).
	Restrict(ctx context.Context, user string, libraries []string) error
	SetPassword(ctx context.Context, user string, pw Password) error
	// Rename gives the user name: *NameRefusedError when another user has
	// it in any case, or on the server's 400 about it.
	Rename(ctx context.Context, user, name string) error
	Disable(ctx context.Context, user string) error
	DeleteUser(ctx context.Context, user string) error
	ListUsers(ctx context.Context) ([]User, error)
}

type User struct {
	ID, Name      string
	Administrator bool
	Disabled      bool
}

// Password is a Member's password for one server's app. It never appears
// in a log, an error or a Write: String, GoString and LogValue print
// [password].
type Password struct{ s string }

var errEmptyPassword = errors.New("the password is empty")

func NewPassword(s string) (Password, error) {
	if s == "" {
		return Password{}, errEmptyPassword
	}
	return Password{s: s}, nil
}

func (p Password) Reveal() string { return p.s }

func (Password) String() string { return "[password]" }

func (Password) GoString() string { return "[password]" }

func (Password) LogValue() slog.Value { return slog.StringValue("[password]") }

// NameRefusedError is another user holding a user's name, or the server's
// 400 about it: the one server reply a Member reads.
type NameRefusedError struct {
	// Taken is whether another user has the name, which the server's users
	// listing showed.
	Taken bool
	// Reply is the server's 400, cut to 200 bytes, when not Taken.
	Reply string
}

func (e *NameRefusedError) Error() string {
	if e.Taken {
		return "the Media Server has a user with this username"
	}
	return "the Media Server refused the username: " + e.Reply
}

// Gone is whether err is the server's 404, which a kind reports as
// ReasonUnsupported with its status.
func Gone(err error) bool {
	p, ok := ServerProblem(err)
	return ok && p.Status == http.StatusNotFound
}
