package identity

import (
	"embed"
	"io/fs"
	"time"

	"github.com/jatassi/slipstream/internal/mail"
)

//go:embed templates/*.tmpl
var templateFiles embed.FS

var templates = func() fs.FS {
	sub, err := fs.Sub(templateFiles, "templates")
	if err != nil {
		panic(err)
	}
	return sub
}()

var (
	signupMail       = mail.MustTemplate[linkMailData](templates, "signup")
	resetMail        = mail.MustTemplate[linkMailData](templates, "reset")
	confirmEmailMail = mail.MustTemplate[linkMailData](templates, "confirm-email")
	pinOffMail       = mail.MustTemplate[pinOffData](templates, "pin-off")
	emailChangedMail = mail.MustTemplate[emailChangedData](templates, "email-changed")
)

// linkMailData is what an email carrying a link says. Expires and
// Replaces are already written out in the Controller's zone.
type linkMailData struct {
	AdminName  string
	ServerName string
	Email      string
	URL        string
	Expires    string
	// Replaces is when the link this one withdrew was sent, "" for none.
	Replaces string
}

type pinOffData struct {
	Email string
	Count int
}

type emailChangedData struct{ NewEmail string }

// when is an instant as an email states it, in the Controller's zone.
func when(t time.Time) string { return t.In(time.Local).Format("Monday, January 2, 2006 at 3:04 PM MST") }

// orYourAdmin is who an email says sent a link: the Admin who issued it,
// or "Your Admin" for one the API key or the host command issued.
func orYourAdmin(name string) string {
	if name == "" {
		return "Your Admin"
	}
	return name
}

// post renders a notice and sends it after the response, through the
// Outbox. Nothing waits for its outcome.
func post[D any](s *Service, to mail.Address, t mail.Template[D], data D) error {
	m, err := t.Render(to, data)
	if err != nil {
		return err
	}
	s.outbox.Post(m)
	return nil
}
