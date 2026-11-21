package subtitles

import (
	"context"
	"errors"

	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/subtitling"
	"github.com/jatassi/slipstream/internal/works"
)

// Facet is a Content Type's subtitle capability, embedded in
// contenttype.Library. A type without subtitles embeds None.
type Facet interface {
	// Subtitled is the kinds this type's files are searched as, none when
	// it has no subtitles.
	Subtitled() []subtitling.Kind
	// SubtitleSubject is what a file holding items of work is searched as:
	// ids, title, year, and for an episode the Aired season and number of
	// every Item, whatever the Episode order. An Item with neither is
	// ErrNoAiredNumber.
	SubtitleSubject(ctx context.Context, r storage.Reader[storage.Library], work works.WorkID, items []works.ItemID) (subtitling.Subject, error)
}

var ErrNoAiredNumber = errors.New("no aired number for this episode")

// ErrNoSubtitles is SubtitleSubject of a type with no subtitles.
var ErrNoSubtitles = errors.New("this Content Type has no subtitles")

// None is the Facet of a type with no subtitles.
type None struct{}

func (None) Subtitled() []subtitling.Kind { return nil }

func (None) SubtitleSubject(context.Context, storage.Reader[storage.Library], works.WorkID, []works.ItemID) (subtitling.Subject, error) {
	return subtitling.Subject{}, ErrNoSubtitles
}

// kindOfType is the kind t's files are searched as, zero for none.
func kindOfType(f Facet) subtitling.Kind {
	if ks := f.Subtitled(); len(ks) > 0 {
		return ks[0]
	}
	return 0
}
