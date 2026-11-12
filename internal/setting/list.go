package setting

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

// ListItem is what a List's entries are, which sets how one is written.
type ListItem string

// Extension is a file name extension, written in lowercase letters and
// digits with no leading dot, such as mkv.
const Extension ListItem = "extension"

type listItem struct {
	noun Noun
	// write returns entry in its stored form, or the sentence refusing it.
	write func(entry string) (string, string)
}

var listItems = map[ListItem]listItem{
	Extension: {noun: Noun{One: "extension", Many: "extensions"}, write: writeExtension},
}

func writeExtension(entry string) (written, refusal string) {
	typed := strings.TrimSpace(entry)
	ext := strings.ToLower(strings.TrimPrefix(typed, "."))
	if ext == "" {
		return "", "An extension can't be blank."
	}
	if strings.IndexFunc(ext, func(r rune) bool { return (r < 'a' || r > 'z') && (r < '0' || r > '9') }) >= 0 {
		return "", `"` + typed + `" isn't an extension. Use letters and digits only, such as mkv.`
	}
	return ext, ""
}

// List is an ordered list of at least Min different entries, each an Item.
type List struct {
	Key, Label string
	Default    []string
	Min        int
	Item       ListItem
}

func (l List) Value(override *[]string) []string {
	if override == nil {
		return l.Default
	}
	return *override
}

// Parse writes each entry in its Item's form and accepts at least Min of
// them with none twice; nil resets.
func (l List) Parse(entries *[]string) (Stored[[]string], error) {
	if entries == nil {
		return Stored[[]string]{}, nil
	}
	item := listItems[l.Item]
	out := make([]string, 0, len(*entries))
	for _, e := range *entries {
		v, refusal := item.write(e)
		switch {
		case refusal != "":
			return Stored[[]string]{}, refuse(l.Key, "%s", refusal)
		case slices.Contains(out, v):
			return Stored[[]string]{}, refuse(l.Key, "%s is in the list twice. Remove one.", v)
		}
		out = append(out, v)
	}
	if len(out) < l.Min {
		return Stored[[]string]{}, refuse(l.Key, "Keep at least %d %s in the list.", l.Min, item.noun.Of(int64(l.Min)))
	}
	return override(out), nil
}

func (l List) ParseJSON(raw json.RawMessage) (Stored[[]string], error) {
	return parseJSON(raw, l.Key, fmt.Sprintf("A list of %s, as strings.", listItems[l.Item].noun.Many), l.Parse)
}

func (l List) Field(override *[]string) ListField {
	return ListField{
		About:   about(l.Key, l.Label, override == nil),
		Kind:    "list",
		Value:   l.Value(override),
		Running: l.Value(override),
		Default: l.Default,
		Min:     l.Min,
		Item:    string(l.Item),
	}
}

// Check reports an unknown Item, or a default Parse would refuse or write
// differently.
func (l List) Check() error {
	if _, ok := listItems[l.Item]; !ok {
		return fmt.Errorf("setting %s: item %q is not a list item", l.Key, l.Item)
	}
	got, err := l.Parse(&l.Default)
	if err != nil {
		return fmt.Errorf("setting %s: default %v is not a list: %w", l.Key, l.Default, err)
	}
	if !slices.Equal(*got.Override(), l.Default) {
		return fmt.Errorf("setting %s: default %v is not written as %v", l.Key, l.Default, *got.Override())
	}
	return nil
}

// ListField is a List on the wire.
type ListField struct {
	About
	Kind    string   `json:"kind" enum:"list"`
	Value   []string `json:"value" nullable:"false"`
	Running []string `json:"running" nullable:"false"`
	Default []string `json:"default" nullable:"false"`
	Min     int      `json:"min" minimum:"0" doc:"The fewest entries the list takes"`
	Item    string   `json:"item" enum:"extension" doc:"What each entry is: an extension is lowercase letters and digits with no leading dot, such as mkv"`
}

// Size is a byte size from Min to Max, set in whole MB.
type Size struct {
	Key, Label        string
	Default, Min, Max int64
}

func (s Size) Value(override *int64) int64 {
	if override == nil {
		return s.Default
	}
	return *override
}

// Parse accepts whole MB within the range; nil resets.
func (s Size) Parse(n *int64) (Stored[int64], error) {
	if n == nil {
		return Stored[int64]{}, nil
	}
	switch {
	case *n < s.Min || *n > s.Max:
		return Stored[int64]{}, refuse(s.Key, "Between %s and %s.", sizeWords(s.Min), sizeWords(s.Max))
	case *n%MB != 0:
		return Stored[int64]{}, refuse(s.Key, "In whole MB.")
	}
	return override(*n), nil
}

func (s Size) ParseJSON(raw json.RawMessage) (Stored[int64], error) {
	return parseJSON(raw, s.Key, "Bytes, as a whole number.", s.Parse)
}

func (s Size) Field(override *int64) SizeField {
	return SizeField{
		About:   about(s.Key, s.Label, override == nil),
		Kind:    "size",
		Value:   s.Value(override),
		Running: s.Value(override),
		Default: s.Default,
		Min:     s.Min,
		Max:     s.Max,
	}
}

func (s Size) Check() error { return checkRange(s.Key, s.Default, s.Min, s.Max, MB) }

// sizeWords writes bytes in GB when whole, else MB.
func sizeWords(n int64) string {
	if n > 0 && n%GB == 0 {
		return fmt.Sprintf("%d GB", n/GB)
	}
	return fmt.Sprintf("%d MB", n/MB)
}

// SizeField is a Size on the wire, in bytes.
type SizeField struct {
	About
	Kind    string `json:"kind" enum:"size"`
	Value   int64  `json:"value" doc:"Bytes"`
	Running int64  `json:"running" doc:"Bytes"`
	Default int64  `json:"default" doc:"Bytes"`
	Min     int64  `json:"min" doc:"Bytes"`
	Max     int64  `json:"max" doc:"Bytes"`
}
