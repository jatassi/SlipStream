package email

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/notification/types"
)

// EncryptionMode defines the TLS encryption strategy
type EncryptionMode string

const (
	EncryptionPreferred EncryptionMode = "preferred" // Try STARTTLS, fall back to plain
	EncryptionAlways    EncryptionMode = "always"    // Require TLS (port 465 or STARTTLS)
	EncryptionNever     EncryptionMode = "never"     // No encryption
)

// Settings contains email-specific configuration
type Settings struct {
	Server     string         `json:"server"`
	Port       int            `json:"port"`
	UseTLS     bool           `json:"useTLS"`
	Encryption EncryptionMode `json:"encryption,omitempty"`
	Username   string         `json:"username,omitempty"`
	Password   string         `json:"password,omitempty"`
	From       string         `json:"from"`
	To         string         `json:"to"`
	CC         string         `json:"cc,omitempty"`
	BCC        string         `json:"bcc,omitempty"`
	UseHTML    bool           `json:"useHtml,omitempty"`
}

// Notifier sends notifications via SMTP email
type Notifier struct {
	name     string
	settings Settings
	logger   *zerolog.Logger
}

// New creates a new email notifier
func New(name string, settings *Settings, logger *zerolog.Logger) *Notifier {
	if settings.Port == 0 {
		settings.Port = 587
	}
	if settings.Encryption == "" {
		if settings.UseTLS {
			settings.Encryption = EncryptionAlways
		} else {
			settings.Encryption = EncryptionPreferred
		}
	}
	subLogger := logger.With().Str("notifier", "email").Str("name", name).Logger()
	return &Notifier{
		name:     name,
		settings: *settings,
		logger:   &subLogger,
	}
}

func (n *Notifier) Type() types.NotifierType {
	return types.NotifierEmail
}

func (n *Notifier) Name() string {
	return n.name
}

func (n *Notifier) Test(ctx context.Context) error {
	return n.sendEmail("SlipStream Test Notification", "This is a test notification from SlipStream.")
}

func (n *Notifier) OnGrab(ctx context.Context, event *types.GrabEvent) error {
	var subject, body string

	if event.Movie != nil {
		subject = fmt.Sprintf("[SlipStream] Movie Grabbed - %s", event.Movie.Title)
		if event.Movie.Year > 0 {
			subject = fmt.Sprintf("[SlipStream] Movie Grabbed - %s (%d)", event.Movie.Title, event.Movie.Year)
		}
		body = fmt.Sprintf("Movie: %s\nQuality: %s\nIndexer: %s\nClient: %s\n\nRelease: %s",
			event.Movie.Title, event.Release.Quality, event.Release.Indexer, event.DownloadClient.Name, event.Release.ReleaseName)
	} else if event.Episode != nil {
		subject = fmt.Sprintf("[SlipStream] %s - %s", event.Episode.FormatEventLabel("Grabbed"), event.Episode.FormatTitle())
		if event.Episode.IsSeasonPack {
			body = fmt.Sprintf("Series: %s\nSeason: %d\nQuality: %s\nIndexer: %s\nClient: %s\n\nRelease: %s",
				event.Episode.SeriesTitle, event.Episode.SeasonNumber,
				event.Release.Quality, event.Release.Indexer, event.DownloadClient.Name, event.Release.ReleaseName)
		} else {
			body = fmt.Sprintf("Series: %s\nSeason: %d Episode: %d\nQuality: %s\nIndexer: %s\nClient: %s\n\nRelease: %s",
				event.Episode.SeriesTitle, event.Episode.SeasonNumber, event.Episode.EpisodeNumber,
				event.Release.Quality, event.Release.Indexer, event.DownloadClient.Name, event.Release.ReleaseName)
		}
	}

	return n.sendEmail(subject, body)
}

func (n *Notifier) OnImport(ctx context.Context, event *types.ImportEvent) error {
	var subject, body string

	if event.Movie != nil {
		subject = fmt.Sprintf("[SlipStream] Movie Downloaded - %s", event.Movie.Title)
		if event.Movie.Year > 0 {
			subject = fmt.Sprintf("[SlipStream] Movie Downloaded - %s (%d)", event.Movie.Title, event.Movie.Year)
		}
		body = fmt.Sprintf("Movie: %s\nQuality: %s\n\nPath: %s", event.Movie.Title, event.Quality, event.DestinationPath)
	} else if event.Episode != nil {
		subject = fmt.Sprintf("[SlipStream] %s - %s", event.Episode.FormatEventLabel("Downloaded"), event.Episode.FormatTitle())
		if event.Episode.IsSeasonPack {
			body = fmt.Sprintf("Series: %s\nSeason: %d\nQuality: %s\n\nPath: %s",
				event.Episode.SeriesTitle, event.Episode.SeasonNumber, event.Quality, event.DestinationPath)
		} else {
			body = fmt.Sprintf("Series: %s\nSeason: %d Episode: %d\nQuality: %s\n\nPath: %s",
				event.Episode.SeriesTitle, event.Episode.SeasonNumber, event.Episode.EpisodeNumber, event.Quality, event.DestinationPath)
		}
	}

	return n.sendEmail(subject, body)
}

func (n *Notifier) OnUpgrade(ctx context.Context, event *types.UpgradeEvent) error {
	var subject, body string

	if event.Movie != nil {
		subject = fmt.Sprintf("[SlipStream] Movie Upgraded - %s", event.Movie.Title)
		if event.Movie.Year > 0 {
			subject = fmt.Sprintf("[SlipStream] Movie Upgraded - %s (%d)", event.Movie.Title, event.Movie.Year)
		}
		body = fmt.Sprintf("Movie: %s\nOld Quality: %s\nNew Quality: %s", event.Movie.Title, event.OldQuality, event.NewQuality)
	} else if event.Episode != nil {
		subject = fmt.Sprintf("[SlipStream] %s - %s", event.Episode.FormatEventLabel("Upgraded"), event.Episode.FormatTitle())
		if event.Episode.IsSeasonPack {
			body = fmt.Sprintf("Series: %s\nSeason: %d\nOld Quality: %s\nNew Quality: %s",
				event.Episode.SeriesTitle, event.Episode.SeasonNumber, event.OldQuality, event.NewQuality)
		} else {
			body = fmt.Sprintf("Series: %s\nSeason: %d Episode: %d\nOld Quality: %s\nNew Quality: %s",
				event.Episode.SeriesTitle, event.Episode.SeasonNumber, event.Episode.EpisodeNumber, event.OldQuality, event.NewQuality)
		}
	}

	return n.sendEmail(subject, body)
}

func (n *Notifier) OnMovieAdded(ctx context.Context, event *types.MovieAddedEvent) error {
	subject := fmt.Sprintf("[SlipStream] Movie Added - %s", event.Movie.Title)
	if event.Movie.Year > 0 {
		subject = fmt.Sprintf("[SlipStream] Movie Added - %s (%d)", event.Movie.Title, event.Movie.Year)
	}

	body := fmt.Sprintf("Movie: %s", event.Movie.Title)
	if event.Movie.Year > 0 {
		body += fmt.Sprintf(" (%d)", event.Movie.Year)
	}
	if event.Movie.Overview != "" {
		body += fmt.Sprintf("\n\n%s", event.Movie.Overview)
	}

	return n.sendEmail(subject, body)
}

func (n *Notifier) OnMovieDeleted(ctx context.Context, event *types.MovieDeletedEvent) error {
	subject := fmt.Sprintf("[SlipStream] Movie Deleted - %s", event.Movie.Title)
	if event.Movie.Year > 0 {
		subject = fmt.Sprintf("[SlipStream] Movie Deleted - %s (%d)", event.Movie.Title, event.Movie.Year)
	}

	body := fmt.Sprintf("Movie: %s", event.Movie.Title)
	if event.DeletedFiles {
		body += "\n\nFiles were also deleted."
	}

	return n.sendEmail(subject, body)
}

func (n *Notifier) OnSeriesAdded(ctx context.Context, event *types.SeriesAddedEvent) error {
	subject := fmt.Sprintf("[SlipStream] Series Added - %s", event.Series.Title)
	if event.Series.Year > 0 {
		subject = fmt.Sprintf("[SlipStream] Series Added - %s (%d)", event.Series.Title, event.Series.Year)
	}

	body := fmt.Sprintf("Series: %s", event.Series.Title)
	if event.Series.Overview != "" {
		body += fmt.Sprintf("\n\n%s", event.Series.Overview)
	}

	return n.sendEmail(subject, body)
}

func (n *Notifier) OnSeriesDeleted(ctx context.Context, event *types.SeriesDeletedEvent) error {
	subject := fmt.Sprintf("[SlipStream] Series Deleted - %s", event.Series.Title)

	body := fmt.Sprintf("Series: %s", event.Series.Title)
	if event.DeletedFiles {
		body += "\n\nFiles were also deleted."
	}

	return n.sendEmail(subject, body)
}

func (n *Notifier) OnHealthIssue(ctx context.Context, event *types.HealthEvent) error {
	subject := fmt.Sprintf("[SlipStream] Health Issue - %s", event.Source)
	body := fmt.Sprintf("Source: %s\nType: %s\n\n%s", event.Source, event.Type, event.Message)

	return n.sendEmail(subject, body)
}

func (n *Notifier) OnHealthRestored(ctx context.Context, event *types.HealthEvent) error {
	subject := fmt.Sprintf("[SlipStream] Health Issue Resolved - %s", event.Source)
	body := fmt.Sprintf("Source: %s\n\n%s", event.Source, event.Message)

	return n.sendEmail(subject, body)
}

func (n *Notifier) OnApplicationUpdate(ctx context.Context, event *types.AppUpdateEvent) error {
	subject := "[SlipStream] Application Updated"
	body := fmt.Sprintf("SlipStream has been updated.\n\nPrevious Version: %s\nNew Version: %s",
		event.PreviousVersion, event.NewVersion)

	return n.sendEmail(subject, body)
}

func (n *Notifier) SendMessage(ctx context.Context, event *types.MessageEvent) error {
	subject := "[SlipStream] " + event.Title
	return n.sendEmail(subject, event.Message)
}

func (n *Notifier) toHTML(plainText string) string {
	escaped := strings.ReplaceAll(plainText, "&", "&amp;")
	escaped = strings.ReplaceAll(escaped, "<", "&lt;")
	escaped = strings.ReplaceAll(escaped, ">", "&gt;")
	escaped = strings.ReplaceAll(escaped, "\n\n", "</p><p>")
	escaped = strings.ReplaceAll(escaped, "\n", "<br>")
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<style>
body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px; }
.header { background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); color: white; padding: 20px; border-radius: 8px 8px 0 0; }
.header h1 { margin: 0; font-size: 24px; }
.content { background: #f9f9f9; padding: 20px; border: 1px solid #ddd; border-top: none; border-radius: 0 0 8px 8px; }
.footer { text-align: center; margin-top: 20px; color: #666; font-size: 12px; }
p { margin: 0 0 10px 0; }
</style>
</head>
<body>
<div class="header"><h1>SlipStream</h1></div>
<div class="content"><p>%s</p></div>
<div class="footer">Sent by SlipStream</div>
</body>
</html>`, escaped)
}

func (n *Notifier) sendEmail(subject, body string) error {
	toAddrs := parseAddresses(n.settings.To)
	ccAddrs := parseAddresses(n.settings.CC)
	bccAddrs := parseAddresses(n.settings.BCC)

	allRecipients := make([]string, 0, len(toAddrs)+len(ccAddrs)+len(bccAddrs))
	allRecipients = append(allRecipients, toAddrs...)
	allRecipients = append(allRecipients, ccAddrs...)
	allRecipients = append(allRecipients, bccAddrs...)

	if len(allRecipients) == 0 {
		return fmt.Errorf("no recipients specified")
	}

	contentType := "text/plain; charset=utf-8"
	emailBody := body
	if n.settings.UseHTML {
		contentType = "text/html; charset=utf-8"
		emailBody = n.toHTML(body)
	}

	var msg strings.Builder
	msg.WriteString(fmt.Sprintf("From: %s\r\n", n.settings.From))
	msg.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(toAddrs, ", ")))
	if len(ccAddrs) > 0 {
		msg.WriteString(fmt.Sprintf("Cc: %s\r\n", strings.Join(ccAddrs, ", ")))
	}
	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	msg.WriteString("MIME-Version: 1.0\r\n")
	msg.WriteString(fmt.Sprintf("Content-Type: %s\r\n", contentType))
	msg.WriteString("\r\n")
	msg.WriteString(emailBody)

	addr := fmt.Sprintf("%s:%d", n.settings.Server, n.settings.Port)

	var auth smtp.Auth
	if n.settings.Username != "" && n.settings.Password != "" {
		auth = smtp.PlainAuth("", n.settings.Username, n.settings.Password, n.settings.Server)
	}

	useTLS := n.settings.Encryption == EncryptionAlways || (n.settings.UseTLS && n.settings.Port == 465)
	if useTLS && n.settings.Port == 465 {
		return n.sendEmailTLS(addr, auth, allRecipients, msg.String())
	}

	return smtp.SendMail(addr, auth, n.settings.From, allRecipients, []byte(msg.String()))
}

func (n *Notifier) sendEmailTLS(addr string, auth smtp.Auth, recipients []string, message string) error {
	client, err := n.dialTLS(addr)
	if err != nil {
		return err
	}
	defer client.Close()

	if err := authenticateAndSetEnvelope(client, auth, n.settings.From, recipients); err != nil {
		return err
	}

	return writeMessageData(client, message)
}

func (n *Notifier) dialTLS(addr string) (*smtp.Client, error) {
	tlsConfig := &tls.Config{
		ServerName: n.settings.Server,
		MinVersion: tls.VersionTLS12,
	}

	dialer := &tls.Dialer{Config: tlsConfig}
	conn, err := dialer.DialContext(context.Background(), "tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	client, err := smtp.NewClient(conn, n.settings.Server)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	return client, nil
}

func authenticateAndSetEnvelope(client *smtp.Client, auth smtp.Auth, from string, recipients []string) error {
	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("failed to authenticate: %w", err)
		}
	}
	if err := client.Mail(from); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}
	for _, rcpt := range recipients {
		if err := client.Rcpt(rcpt); err != nil {
			return fmt.Errorf("failed to set recipient %s: %w", rcpt, err)
		}
	}
	return nil
}

func writeMessageData(client *smtp.Client, message string) error {
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to get data writer: %w", err)
	}
	if _, err := w.Write([]byte(message)); err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("failed to close data writer: %w", err)
	}
	return client.Quit()
}

func parseAddresses(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	addrs := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			addrs = append(addrs, p)
		}
	}
	return addrs
}
