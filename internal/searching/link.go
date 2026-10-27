package searching

import (
	"fmt"
	"io"
	"log/slog"

	"github.com/jatassi/slipstream/internal/magnet"
)

// Link is a download link. It may carry a passkey, so it never prints:
// fmt and slog show [link]. The Node's transport and the wire read it, and
// so does the store that keeps it, through Reveal.
type Link struct{ raw string }

func LinkOf(raw string) Link { return Link{raw: raw} }

func (l Link) IsZero() bool { return l.raw == "" }

func (l Link) IsMagnet() bool { return magnet.Is(l.raw) }

// Magnet reads l as a magnet link: a *magnet.InvalidError when it isn't
// one.
func (l Link) Magnet() (magnet.Link, error) { return magnet.Parse(l.raw) }

// Reveal is l in clear, for the store that keeps it.
func (l Link) Reveal() string { return l.raw }

func (Link) LogValue() slog.Value { return slog.StringValue("[link]") }

func (Link) Format(f fmt.State, _ rune) { _, _ = io.WriteString(f, "[link]") }
