package port

import (
	"errors"
	"fmt"
	"time"

	"github.com/jatassi/slipstream/internal/newznab"
)

// TorrentInfo is Prowlarr's, field for field. The framework converts it
// to the contract's Release: it drops a release with no title, dedupes
// by GUID, reads InfoHash as hex, takes Leechers as Peers less Seeders and
// MinimumSeedTime as seconds. IndexerFlags are only the ones the site
// states; the factor flags come from the factors, in one place.
type TorrentInfo struct {
	GUID                 string
	Title                string
	Size                 int64
	DownloadURL          string
	MagnetURL            string
	InfoURL              string
	InfoHash             string
	PublishDate          time.Time
	Seeders              int
	Peers                int
	Grabs                int
	Files                int
	ImdbID               int
	TmdbID               int
	TvdbID               int
	Categories           []newznab.Category
	DownloadVolumeFactor float64
	UploadVolumeFactor   float64
	MinimumRatio         float64
	MinimumSeedTime      int64
	IndexerFlags         []SiteFlag
}

// SiteFlag is a flag the site states. The factor flags, such as
// freeleech, have no SiteFlag, so a port can't set one.
type SiteFlag string

const (
	Internal  SiteFlag = "internal"
	Scene     SiteFlag = "scene"
	Exclusive SiteFlag = "exclusive"
	Nuked     SiteFlag = "nuked"
	Golden    SiteFlag = "golden"
	Approved  SiteFlag = "approved"
)

// Got is what Download fetched: a Torrent or a Magnet.
type Got interface{ got() }

type (
	Torrent struct{ Raw []byte }
	Magnet  struct{ Link string }
)

func (Torrent) got() {}
func (Magnet) got()  {}

// The errors a port returns, named after the C# exceptions they stand
// for. The framework maps each to the contract's outcome by the port's
// auth family, so a port never chooses a stop cause.

// SiteError is IndexerException: the job fails with the site's words.
type SiteError struct{ Message string }

func (e *SiteError) Error() string { return e.Message }

// AuthError is IndexerAuthException: the site refused the credentials.
// Message is the site's own words, or "".
type AuthError struct{ Message string }

func (e *AuthError) Error() string {
	if e.Message == "" {
		return "the site refused the credentials"
	}
	return "the site refused the credentials: " + e.Message
}

// LimitError is RequestLimitReachedException and
// TooManyRequestsException: the site limited requests, for After when it
// said, else for the Controller's default.
type LimitError struct {
	Message string
	After   time.Duration
}

func (e *LimitError) Error() string {
	if e.Message == "" {
		return "the site limited requests"
	}
	return "the site limited requests: " + e.Message
}

// IndexerError is IndexerException.
func IndexerError(format string, args ...any) error {
	return &SiteError{Message: fmt.Sprintf(format, args...)}
}

// IndexerAuthError is IndexerAuthException. Its cause comes from the auth
// family: account_refused naming Entry.Key for StatelessKey,
// credentials_rejected for a sign-in, cookie_expired for PastedCookie.
func IndexerAuthError(msg string) error { return &AuthError{Message: msg} }

// RequestLimitReached is limited with no retry time.
func RequestLimitReached(msg string) error { return &LimitError{Message: msg} }

// TooManyRequests is limited for after, zero when the site gave none.
func TooManyRequests(after time.Duration) error { return &LimitError{After: after} }

var (
	// ErrReleaseUnavailable is a download link the site no longer serves.
	ErrReleaseUnavailable = errors.New("the site no longer serves the release")
	// ErrTwoFactorCodeNeeded stops a CodeLogin with code_needed.
	ErrTwoFactorCodeNeeded = errors.New("the site needs a two-factor code")
)

// FilterReleasesByQuery is IndexerBase.FilterReleasesByQuery.
func FilterReleasesByQuery(_ []TorrentInfo, _ SearchCriteria) []TorrentInfo {
	panic("port: FilterReleasesByQuery is not implemented")
}
