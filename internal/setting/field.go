package setting

type About struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	DefaultFrom string `json:"defaultFrom,omitempty" doc:"Where the default comes from, when it is another setting's value"`
	Source      string `json:"source" enum:"database"`
	Applies     string `json:"applies" enum:"live,restart"`
	IsDefault   bool   `json:"isDefault" doc:"The setting has no override; Reset to default leaves it unchanged"`
}

func about(key, label string, isDefault bool) About {
	return About{Key: key, Label: label, Source: "database", Applies: "live", IsDefault: isDefault}
}

// DurationField is a Duration on the wire, in seconds.
type DurationField struct {
	About
	Kind    string `json:"kind" enum:"duration"`
	Value   int64  `json:"value" doc:"Seconds"`
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
	Default int64  `json:"default"`
	Min     int64  `json:"min"`
	Max     int64  `json:"max"`
	Unit    Noun   `json:"unit"`
	Floor   string `json:"floor,omitempty" doc:"Why the minimum protects someone outside the install"`
}

type ChoiceField struct {
	About
	Kind    string         `json:"kind" enum:"choice"`
	Value   string         `json:"value"`
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
	Default Times  `json:"default"`
	Zone    string `json:"zone" doc:"The time zone's name, such as Europe/London"`
}

type Times struct {
	Start string `json:"start" doc:"HH:MM"`
	End   string `json:"end" doc:"HH:MM, before start when the window crosses midnight"`
}
