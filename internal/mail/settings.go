package mail

import (
	"context"
	"crypto/x509"
	"embed"

	"github.com/jatassi/slipstream/internal/mail/internal/db"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage"
)

//go:embed schema/*.sql
var schemaFiles embed.FS

func Schema() storage.Schema[storage.Library] {
	return storage.NewSchema[storage.Library]("mail", schemaFiles)
}

// Settings is the one SMTP setting every email uses. The password is
// write-only: PasswordSet says whether one is saved.
type Settings struct {
	Host        string
	Port        uint16
	Security    Security
	Username    string
	PasswordSet bool
	From        Address
}

// Patch changes the fields it sets. A Password of "" clears it.
type Patch struct {
	Host, Username, FromAddress, FromName *string
	Port                                  *uint16
	Security                              *Security
	Password                              *string
}

type Config struct {
	Library *storage.DB[storage.Library]
	Sealer  *secret.Sealer
	Timings Timings
	// Roots are the CAs a server's certificate must chain to, nil for the
	// system's.
	Roots *x509.CertPool
}

// Mail is the SMTP setting and the Sender built on it. It reads the
// setting at every Send, so a change applies to the next email.
type Mail struct{ cfg Config }

var _ Sender = (*Mail)(nil)

var passwordColumn = secret.NewColumn("mail_settings.password_sealed")

func New(cfg Config) *Mail { return &Mail{cfg: cfg} }

func (m *Mail) Settings(ctx context.Context) (Settings, error) {
	row, err := db.New(m.cfg.Library).GetSettings(ctx)
	if err != nil {
		return Settings{}, err
	}
	return settingsOf(row), nil
}

func (m *Mail) Update(ctx context.Context, p Patch) (Settings, error) {
	var out Settings
	err := m.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		q := db.New(tx)
		row, err := q.GetSettings(ctx)
		if err != nil {
			return err
		}
		set := func(dst *string, v *string) {
			if v != nil {
				*dst = *v
			}
		}
		set(&row.Host, p.Host)
		set(&row.Username, p.Username)
		set(&row.FromAddress, p.FromAddress)
		set(&row.FromName, p.FromName)
		if p.Port != nil {
			row.Port = int64(*p.Port)
		}
		if p.Security != nil {
			row.Security = string(*p.Security)
		}
		if p.Password != nil {
			row.PasswordSealed = secret.Sealed{}
			if *p.Password != "" {
				row.PasswordSealed = m.cfg.Sealer.Seal([]byte(*p.Password), passwordColumn.Single())
			}
		}
		out = settingsOf(row)
		return q.SaveSettings(ctx, db.SaveSettingsParams{
			Host:           row.Host,
			Port:           row.Port,
			Security:       row.Security,
			Username:       row.Username,
			PasswordSealed: row.PasswordSealed,
			FromAddress:    row.FromAddress,
			FromName:       row.FromName,
		})
	})
	return out, err
}

func (m *Mail) Send(ctx context.Context, msg Message) (Outcome, error) {
	if storage.InWrite(ctx) {
		panic("mail: Send inside Write; send after the commit")
	}
	row, err := db.New(m.cfg.Library).GetSettings(ctx)
	if err != nil {
		return Outcome{}, err
	}
	var password []byte
	if row.PasswordSealed.Set() {
		if password, err = m.cfg.Sealer.Open(row.PasswordSealed, passwordColumn.Single()); err != nil {
			return Outcome{}, err
		}
	}
	s := settingsOf(row)
	server := Server{Host: s.Host, Port: s.Port, Security: s.Security, Username: s.Username, Password: string(password), From: s.From}
	return SMTP{Server: server, Timings: m.cfg.Timings, Roots: m.cfg.Roots}.Send(ctx, msg)
}

func settingsOf(row db.MailSetting) Settings {
	return Settings{
		Host:        row.Host,
		Port:        uint16(row.Port), //nolint:gosec // the column's CHECK holds it to 1-65535
		Security:    Security(row.Security),
		Username:    row.Username,
		PasswordSet: row.PasswordSealed.Set(),
		From:        Address{Name: row.FromName, Email: row.FromAddress},
	}
}
