package clients

import (
	"errors"
	"strings"
	"unicode"
)

// Label is a label that fits its kind. Only ParseLabel makes one, so a
// Label is never empty.
type Label struct{ s string }

const DefaultLabel = "slipstream"

// LabelRule is which labels a kind's client takes.
type LabelRule uint8

const (
	// AnyLabel is qBittorrent's category and rTorrent's d.custom1.
	AnyLabel LabelRule = iota + 1
	// NoComma is Transmission, which splits labels at a comma.
	NoComma
	// LowerWord is Deluge's Label plugin: a-z, 0-9, _ and - only.
	LowerWord
)

var (
	ErrLabelEmpty   = errors.New("the label is empty")
	ErrLabelControl = errors.New("the label has a control character")
	ErrLabelComma   = errors.New("the label has a comma")
	ErrLabelWord    = errors.New("the label has a character other than a-z, 0-9, _ and -")
)

// ParseLabel is raw as kind's label.
func ParseLabel(raw string, kind Kind) (Label, error) {
	switch {
	case raw == "":
		return Label{}, ErrLabelEmpty
	case strings.ContainsFunc(raw, unicode.IsControl):
		return Label{}, ErrLabelControl
	}
	switch kind.Facts().Label {
	case AnyLabel:
	case NoComma:
		if strings.Contains(raw, ",") {
			return Label{}, ErrLabelComma
		}
	case LowerWord:
		if strings.ContainsFunc(raw, func(r rune) bool { return !lowerWord(r) }) {
			return Label{}, ErrLabelWord
		}
	}
	return Label{raw}, nil
}

func lowerWord(r rune) bool {
	return r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '_' || r == '-'
}

func (l Label) String() string { return l.s }
