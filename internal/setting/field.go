package setting

// About is what every kind of field carries. A database setting is never
// locked, applies live and its running value is its value. A config.yaml
// setting reports the layer that set it, the variable that locks it, and
// whether the saved value waits for a restart.
type About struct {
	Key            string `json:"key"`
	Label          string `json:"label"`
	DefaultFrom    string `json:"defaultFrom,omitempty" doc:"Where the default comes from, when it is another setting's value"`
	Source         string `json:"source" enum:"database,default,config.yaml,.env,env"`
	Applies        string `json:"applies" enum:"live,restart"`
	IsDefault      bool   `json:"isDefault" doc:"The setting has no override; Reset to default leaves it unchanged"`
	LockedBy       string `json:"lockedBy,omitempty" doc:"The environment variable that sets the value; the control is read-only"`
	PendingRestart bool   `json:"pendingRestart" doc:"The saved value differs from the running one until the Node restarts"`
}

func about(key, label string, isDefault bool) About {
	return About{Key: key, Label: label, Source: "database", Applies: "live", IsDefault: isDefault}
}

// Field is any kind's wire form. Only this package declares kinds.
type Field interface {
	about() About
}

func (a About) about() About { return a }

// Kinds is every kind of Field, once each.
var Kinds = []Field{DurationField{}, CountField{}, ChoiceField{}, WindowField{}, PortField{}, SwitchField{}}

// DurationField is a Duration on the wire, in seconds.
type DurationField struct {
	About
	Kind    string `json:"kind" enum:"duration"`
	Value   int64  `json:"value" doc:"Seconds"`
	Running int64  `json:"running" doc:"Seconds"`
	Default int64  `json:"default" doc:"Seconds"`
	Min     int64  `json:"min" doc:"Seconds"`
	Max     int64  `json:"max" doc:"Seconds"`
	Step    int64  `json:"step" doc:"Seconds: a second, minute, hour or day"`
	Floor   string `json:"floor,omitempty" doc:"Why the minimum protects someone outside the install"`
}

type CountField struct {
	About
	Kind    string `json:"kind" enum:"count"`
	Value   int64  `json:"value"`
	Running int64  `json:"running"`
	Default int64  `json:"default"`
	Min     int64  `json:"min"`
	Max     int64  `json:"max"`
	Unit    string `json:"unit"`
	Zero    string `json:"zero,omitempty" doc:"The words for 0, such as No limit"`
	Floor   string `json:"floor,omitempty" doc:"Why the minimum protects someone outside the install"`
}

type ChoiceField struct {
	About
	Kind    string         `json:"kind" enum:"choice"`
	Value   string         `json:"value"`
	Running string         `json:"running"`
	Default string         `json:"default"`
	Options []ChoiceOption `json:"options" nullable:"false"`
}

type ChoiceOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// WindowField is a Window in the time zone Zone names.
type WindowField struct {
	About
	Kind    string `json:"kind" enum:"window"`
	Value   Times  `json:"value"`
	Running Times  `json:"running"`
	Default Times  `json:"default"`
	Zone    string `json:"zone" doc:"The time zone's name, such as Europe/London"`
}

// Times is a window's start and end as "HH:MM".
type Times struct {
	Start string `json:"start" doc:"HH:MM"`
	End   string `json:"end" doc:"HH:MM, before start when the window crosses midnight"`
}

// PortField is a Port on the wire.
type PortField struct {
	About
	Kind    string `json:"kind" enum:"port"`
	Value   int64  `json:"value"`
	Running int64  `json:"running"`
	Default int64  `json:"default"`
}

// SwitchField is a Switch on the wire.
type SwitchField struct {
	About
	Kind    string `json:"kind" enum:"switch"`
	Value   bool   `json:"value"`
	Running bool   `json:"running"`
	Default bool   `json:"default"`
}
