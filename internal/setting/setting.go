// Package setting describes an Admin setting once: its key, label,
// default, range, step and floor. An owner stores a nullable override in
// its own row, where null means the default, accepts only a Stored value
// that Parse built, and reports the setting on the wire through Field.
package setting

import (
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"
)

// Stored is a value Parse accepted, in the form its owner keeps: an
// override, or nothing, which resets the setting to its default. The zero
// Stored resets.
type Stored[T any] struct {
	v   T
	set bool
}

// Override is the value to store, nil to store null.
func (s Stored[T]) Override() *T {
	if !s.set {
		return nil
	}
	v := s.v
	return &v
}

func override[T any](v T) Stored[T] { return Stored[T]{v: v, set: true} }

// RangeError refuses a value its setting doesn't accept. Sentence is what
// the setting's control shows.
type RangeError struct {
	Key      string
	Sentence string
}

func (e *RangeError) Error() string { return e.Key + ": " + e.Sentence }

func refuse(key, format string, args ...any) error {
	return &RangeError{Key: key, Sentence: fmt.Sprintf(format, args...)}
}

// Duration is a length of time, stored as whole seconds and set in whole
// Steps between Min and Max. Floor, when set, is why Min protects someone
// outside the install.
type Duration struct {
	Key, Label              string
	Default, Min, Max, Step time.Duration
	Floor                   string
}

// Value is the setting's value under override, a stored count of seconds.
func (d Duration) Value(override *int64) time.Duration {
	if override == nil {
		return d.Default
	}
	return time.Duration(*override) * time.Second
}

// Parse accepts seconds within the range and on the step; nil resets.
func (d Duration) Parse(seconds *int64) (Stored[int64], error) {
	if seconds == nil {
		return Stored[int64]{}, nil
	}
	v := time.Duration(*seconds) * time.Second
	switch {
	case v < d.Min && d.Floor != "":
		return Stored[int64]{}, refuse(d.Key, "No faster than %s. %s", Words(d.Min), d.Floor)
	case v < d.Min || v > d.Max:
		return Stored[int64]{}, refuse(d.Key, "Between %s and %s.", Words(d.Min), Words(d.Max))
	case v%d.Step != 0:
		return Stored[int64]{}, refuse(d.Key, "In whole %s.", stepWords[d.Step])
	}
	return override(*seconds), nil
}

func (d Duration) Field(override *int64) DurationField {
	return DurationField{
		About:   about(d.Key, d.Label, override == nil),
		Kind:    "duration",
		Value:   seconds(d.Value(override)),
		Running: seconds(d.Value(override)),
		Default: seconds(d.Default),
		Min:     seconds(d.Min),
		Max:     seconds(d.Max),
		Step:    seconds(d.Step),
		Floor:   d.Floor,
	}
}

// Check reports a declaration whose range, step or default can't work.
func (d Duration) Check() error {
	if _, ok := stepWords[d.Step]; !ok {
		return fmt.Errorf("setting %s: step %s is not a second, minute, hour or day", d.Key, d.Step)
	}
	return checkRange(d.Key, d.Default, d.Min, d.Max, d.Step)
}

const day = 24 * time.Hour

var stepWords = map[time.Duration]string{time.Second: "seconds", time.Minute: "minutes", time.Hour: "hours", day: "days"}

func seconds(d time.Duration) int64 { return int64(d / time.Second) }

// Words writes d in its largest whole unit: "1 day", "8 h", "10 min", "5 s".
func Words(d time.Duration) string {
	switch {
	case d == day:
		return "1 day"
	case d > 0 && d%day == 0:
		return fmt.Sprintf("%d days", d/day)
	case d > 0 && d%time.Hour == 0:
		return fmt.Sprintf("%d h", d/time.Hour)
	case d > 0 && d%time.Minute == 0:
		return fmt.Sprintf("%d min", d/time.Minute)
	}
	return fmt.Sprintf("%d s", d/time.Second)
}

type Noun struct {
	One  string `json:"one"`
	Many string `json:"many"`
}

func (u Noun) Of(n int64) string {
	if n == 1 {
		return u.One
	}
	return u.Many
}

type Count struct {
	Key, Label        string
	Default, Min, Max int64
	Unit              Noun
	Zero              string
	Floor             string
}

func (c Count) Value(override *int64) int64 {
	if override == nil {
		return c.Default
	}
	return *override
}

// Parse accepts a number within the range; nil resets.
func (c Count) Parse(n *int64) (Stored[int64], error) {
	if n == nil {
		return Stored[int64]{}, nil
	}
	switch {
	case *n < c.Min && c.Floor != "":
		return Stored[int64]{}, refuse(c.Key, "At least %s. %s", c.words(c.Min), c.Floor)
	case *n < c.Min || *n > c.Max:
		return Stored[int64]{}, refuse(c.Key, "Between %d and %s.", c.Min, c.words(c.Max))
	}
	return override(*n), nil
}

func (c Count) words(n int64) string {
	if c.Unit.Many == "%" {
		return fmt.Sprintf("%d%%", n)
	}
	return fmt.Sprintf("%d %s", n, c.Unit.Of(n))
}

func (c Count) Field(override *int64) CountField {
	return CountField{
		About:   about(c.Key, c.Label, override == nil),
		Kind:    "count",
		Value:   c.Value(override),
		Running: c.Value(override),
		Default: c.Default,
		Min:     c.Min,
		Max:     c.Max,
		Unit:    c.Unit,
		Zero:    c.Zero,
		Floor:   c.Floor,
	}
}

func (c Count) Check() error { return checkRange(c.Key, c.Default, c.Min, c.Max, 1) }

func checkRange[T int64 | time.Duration](key string, def, lo, hi, step T) error {
	switch {
	case lo > hi:
		return fmt.Errorf("setting %s: minimum %v is above maximum %v", key, lo, hi)
	case def < lo || def > hi:
		return fmt.Errorf("setting %s: default %v is outside %v to %v", key, def, lo, hi)
	case def%step != 0 || lo%step != 0 || hi%step != 0:
		return fmt.Errorf("setting %s: default and range are not whole steps of %v", key, step)
	}
	return nil
}

type Option[T ~string] struct {
	Value T
	Label string
}

// listedChoices is the most options a refusal still names one by one.
const listedChoices = 6

// Choice is one of a closed set of Options. DefaultFrom names where the
// default comes from when it is another setting's value.
type Choice[T ~string] struct {
	Key, Label  string
	Default     T
	DefaultFrom string
	Options     []Option[T]
}

func (c Choice[T]) Value(override *T) T {
	if override == nil {
		return c.Default
	}
	return *override
}

// Parse accepts one of the Options' values; nil resets.
func (c Choice[T]) Parse(v *string) (Stored[T], error) {
	if v == nil {
		return Stored[T]{}, nil
	}
	i := slices.IndexFunc(c.Options, func(o Option[T]) bool { return string(o.Value) == *v })
	if i < 0 && len(c.Options) > listedChoices {
		return Stored[T]{}, refuse(c.Key, "%s is not one of the %d choices for %s.", *v, len(c.Options), c.Label)
	}
	if i < 0 {
		labels := make([]string, len(c.Options))
		for i, o := range c.Options {
			labels[i] = o.Label
		}
		return Stored[T]{}, refuse(c.Key, "Choose %s.", strings.Join(labels, " or "))
	}
	return override(c.Options[i].Value), nil
}

func (c Choice[T]) Field(override *T) ChoiceField {
	f := ChoiceField{
		About:   about(c.Key, c.Label, override == nil),
		Kind:    "choice",
		Value:   string(c.Value(override)),
		Running: string(c.Value(override)),
		Default: string(c.Default),
		Options: make([]ChoiceOption, len(c.Options)),
	}
	f.DefaultFrom = c.DefaultFrom
	for i, o := range c.Options {
		f.Options[i] = ChoiceOption{Value: string(o.Value), Label: o.Label}
	}
	return f
}

func (c Choice[T]) Check() error {
	if !slices.ContainsFunc(c.Options, func(o Option[T]) bool { return o.Value == c.Default }) {
		return fmt.Errorf("setting %s: default %q is not an option", c.Key, c.Default)
	}
	return nil
}

// Span is a daily stretch of time in minutes after local midnight. An End
// before Start crosses midnight.
type Span struct {
	Start, End int
}

const minutesPerDay = 24 * 60

func (s Span) times() Times { return Times{Start: clock(s.Start), End: clock(s.End)} }

func clock(minutes int) string { return fmt.Sprintf("%02d:%02d", minutes/60, minutes%60) }

// Window is a non-empty daily Span.
type Window struct {
	Key, Label string
	Default    Span
}

func (w Window) Value(override *Span) Span {
	if override == nil {
		return w.Default
	}
	return *override
}

// Parse accepts two different "HH:MM" times; nil resets.
func (w Window) Parse(t *Times) (Stored[Span], error) {
	if t == nil {
		return Stored[Span]{}, nil
	}
	start, okStart := parseClock(t.Start)
	end, okEnd := parseClock(t.End)
	if !okStart || !okEnd {
		return Stored[Span]{}, refuse(w.Key, "Times are 00:00 to 23:59.")
	}
	if start == end {
		return Stored[Span]{}, refuse(w.Key, "The window needs an end after its start.")
	}
	return override(Span{Start: start, End: end}), nil
}

func parseClock(s string) (int, bool) {
	t, err := time.Parse("15:04", s)
	if err != nil || len(s) != len("15:04") {
		return 0, false
	}
	return t.Hour()*60 + t.Minute(), true
}

// Field reports the window in zone, the name of the time zone it is kept in.
func (w Window) Field(override *Span, zone string) WindowField {
	return WindowField{
		About:   about(w.Key, w.Label, override == nil),
		Kind:    "window",
		Value:   w.Value(override).times(),
		Running: w.Value(override).times(),
		Default: w.Default.times(),
		Zone:    zone,
	}
}

func (w Window) Check() error {
	d := w.Default
	if d.Start < 0 || d.Start >= minutesPerDay || d.End < 0 || d.End >= minutesPerDay || d.Start == d.End {
		return fmt.Errorf("setting %s: default %+v is not a non-empty daily span", w.Key, d)
	}
	return nil
}

// Port is a TCP port, 1 to 65535.
type Port struct {
	Key, Label string
	Default    uint16
}

func (p Port) Value(override *uint16) uint16 {
	if override == nil {
		return p.Default
	}
	return *override
}

// Parse accepts a port number; nil resets.
func (p Port) Parse(n *int64) (Stored[uint16], error) {
	if n == nil {
		return Stored[uint16]{}, nil
	}
	if *n < 1 || *n > 65535 {
		return Stored[uint16]{}, refuse(p.Key, "Between 1 and 65535.")
	}
	return override(uint16(*n)), nil
}

func (p Port) Field(override *uint16) PortField {
	return PortField{
		About:   about(p.Key, p.Label, override == nil),
		Kind:    "port",
		Value:   int64(p.Value(override)),
		Running: int64(p.Value(override)),
		Default: int64(p.Default),
	}
}

func (p Port) Check() error {
	if p.Default == 0 {
		return fmt.Errorf("setting %s: default 0 is not a port", p.Key)
	}
	return nil
}

// Switch is on or off.
type Switch struct {
	Key, Label string
	Default    bool
}

func (s Switch) Value(override *bool) bool {
	if override == nil {
		return s.Default
	}
	return *override
}

func (s Switch) Field(override *bool) SwitchField {
	return SwitchField{
		About:   about(s.Key, s.Label, override == nil),
		Kind:    "switch",
		Value:   s.Value(override),
		Running: s.Value(override),
		Default: s.Default,
	}
}

// Unit is what an Amount counts.
type Unit string

const (
	BytesUnit   Unit = "bytes"
	PercentUnit Unit = "percent"
)

// GB and TB are the 1024-based units the web's byte words use, so a level
// and the Storage bar read the same.
const (
	GB int64 = 1 << 30
	TB int64 = 1 << 40
)

const maxSpace = 1024 * TB

// Amount is free space as bytes, or as a whole percent of a volume.
type Amount struct {
	N    int64 `json:"amount" minimum:"0"`
	Unit Unit  `json:"unit" enum:"bytes,percent"`
}

// Bytes is the amount on a volume of total bytes. A percent of a volume
// with no total is 0, so it never judges a broken figure.
func (a Amount) Bytes(total int64) int64 {
	if a.Unit == PercentUnit {
		return total * a.N / 100
	}
	return a.N
}

// Words writes a in its largest unit: "100 GB", "1.5 TB", "10%".
func (a Amount) Words() string {
	if a.Unit == PercentUnit {
		return fmt.Sprintf("%d%%", a.N)
	}
	return byteWords(a.N)
}

func byteWords(n int64) string {
	unit, name := GB, "GB"
	if n >= TB {
		unit, name = TB, "TB"
	}
	return strconv.FormatFloat(math.Round(float64(n)/float64(unit)*10)/10, 'f', -1, 64) + " " + name
}

// Space is free space set as an amount or a percentage. DefaultFrom names
// where the default comes from when it is another setting's value.
type Space struct {
	Key, Label  string
	Default     Amount
	DefaultFrom string
}

func (s Space) Value(override *Amount) Amount {
	if override == nil {
		return s.Default
	}
	return *override
}

// Parse accepts bytes 0 to 1 PB or a whole percent 0 to 100; nil resets.
func (s Space) Parse(v *Amount) (Stored[Amount], error) {
	if v == nil {
		return Stored[Amount]{}, nil
	}
	switch {
	case v.Unit != BytesUnit && v.Unit != PercentUnit:
		return Stored[Amount]{}, refuse(s.Key, "Choose bytes or percent.")
	case v.Unit == BytesUnit && (v.N < 0 || v.N > maxSpace):
		return Stored[Amount]{}, refuse(s.Key, "Between 0 GB and %s.", byteWords(maxSpace))
	case v.Unit == PercentUnit && (v.N < 0 || v.N > 100):
		return Stored[Amount]{}, refuse(s.Key, "Between 0 and 100%%.")
	}
	return override(*v), nil
}

func (s Space) Field(override *Amount) SpaceField {
	f := SpaceField{
		About:    about(s.Key, s.Label, override == nil),
		Kind:     "space",
		Value:    s.Value(override),
		Running:  s.Value(override),
		Default:  s.Default,
		MaxBytes: maxSpace,
	}
	f.DefaultFrom = s.DefaultFrom
	return f
}

func (s Space) Check() error {
	if _, err := s.Parse(&s.Default); err != nil {
		return fmt.Errorf("setting %s: default %+v is not a space: %w", s.Key, s.Default, err)
	}
	return nil
}
