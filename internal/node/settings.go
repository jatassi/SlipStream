package node

import (
	"errors"
	"io/fs"
	"net/netip"
	"strconv"

	"github.com/jatassi/slipstream/internal/contenttype"
	"github.com/jatassi/slipstream/internal/install"
	"github.com/jatassi/slipstream/internal/node/internal/config"
	"github.com/jatassi/slipstream/internal/node/internal/logs"
	"github.com/jatassi/slipstream/internal/nodesystem"
	"github.com/jatassi/slipstream/internal/setting"
)

// nodeSettings is the Node settings table: one row per config.yaml key an
// Admin edits, and the only keys the API, the config control op and
// slipstream config set can write. A row's default is Defaults' value,
// never restated here. A row applies live when it has an applier, and on
// restart otherwise.
var nodeSettings = []settingRow{
	portRow{desc: setting.Port{Key: "http.listen", Label: "Port"}, rowOf: rowOf[netip.AddrPort]{at: func(c *Config) *netip.AddrPort { return &c.HTTP.Listen }}},
	countRow{
		desc:  setting.Count{Key: "http.ports_to_try", Label: "Ports to try", Min: 1, Max: 100, Unit: setting.Noun{One: "port", Many: "ports"}},
		rowOf: rowOf[int]{at: func(c *Config) *int { return &c.HTTP.PortsToTry }},
	},
	choiceRow{
		desc:  setting.Choice[logs.Level]{Key: "log.level", Label: "Level", Options: logLevels},
		rowOf: rowOf[logs.Level]{at: func(c *Config) *logs.Level { return &c.Log.Level }, applies: applyLog},
	},
	countRow{
		desc:  setting.Count{Key: "log.max_size_mb", Label: "Rotate at", Min: 1, Max: 1000, Unit: setting.Noun{One: "MB", Many: "MB"}},
		rowOf: rowOf[int]{at: func(c *Config) *int { return &c.Log.MaxSizeMB }, applies: applyLog},
	},
	countRow{
		desc:  setting.Count{Key: "log.max_files", Label: "Files kept", Min: 0, Max: 100, Unit: setting.Noun{One: "file", Many: "files"}, Zero: "No limit"},
		rowOf: rowOf[int]{at: func(c *Config) *int { return &c.Log.MaxFiles }, applies: applyLog},
	},
	countRow{
		desc:  setting.Count{Key: "log.max_age_days", Label: "Days kept", Min: 0, Max: 3650, Unit: setting.Noun{One: "day", Many: "days"}, Zero: "No limit"},
		rowOf: rowOf[int]{at: func(c *Config) *int { return &c.Log.MaxAgeDays }, applies: applyLog},
	},
	switchRow{
		desc:  setting.Switch{Key: "log.compress", Label: "Compress rotated files"},
		rowOf: rowOf[bool]{at: func(c *Config) *bool { return &c.Log.Compress }, applies: applyLog},
	},
	countRow{
		desc:  setting.Count{Key: "log.viewer_lines", Label: "Lines kept for the viewer", Min: 100, Max: 50000, Unit: setting.Noun{One: "line", Many: "lines"}},
		rowOf: rowOf[int]{at: func(c *Config) *int { return &c.Log.ViewerLines }, applies: applyLog},
	},
}

var logLevels = []setting.Option[logs.Level]{
	{Value: logs.Debug, Label: "Debug"},
	{Value: logs.Info, Label: "Info"},
	{Value: logs.Warn, Label: "Warn"},
	{Value: logs.Error, Label: "Error"},
}

// liveApply applies a saved section to the running Node. Rows of one
// section share it, so a change to several of them applies it once.
type liveApply struct {
	apply func(n *Node, saved Config) error
}

var applyLog = &liveApply{apply: func(n *Node, saved Config) error { return n.logs.Apply(saved.Log) }}

func settingRowOf(key string) (settingRow, bool) {
	for _, r := range nodeSettings {
		if r.key() == key {
			return r, true
		}
	}
	return nil, false
}

// settingRow is one row of the table. Each kind of setting implements it.
type settingRow interface {
	key() string
	live() bool
	// field is the row's wire form in v.
	field(v view) setting.Field
	// text is v as config.yaml spells it, nil for a reset, or a
	// *setting.RangeError. saved is the Config the change applies to.
	text(v nodesystem.Value, saved *Config) (*string, error)
	// check is the row's range on a loaded Config, which CheckConfig runs
	// so a hand edit is held to it too.
	check(c *Config) error
	same(a, b *Config) bool
	// adopt sets running's value to saved's.
	adopt(running, saved *Config)
	// applier is how a live row applies, nil for a restart row.
	applier() *liveApply
}

// view is what a row's field is drawn from: the saved and running Configs,
// the defaults, each key's source and the port the web listener bound, 0
// before it binds.
type view struct {
	saved, running, defaults *Config
	keys                     map[string]Key
	boundHTTP                uint16
}

func keysByName(keys []Key) map[string]Key {
	m := make(map[string]Key, len(keys))
	for _, k := range keys {
		m[k.Key] = k
	}
	return m
}

// fields is every row's field in v, in table order.
func (v view) fields() []setting.Field {
	out := make([]setting.Field, len(nodeSettings))
	for i, r := range nodeSettings {
		out[i] = r.field(v)
	}
	return out
}

// rowOf is what every kind of row shares: where its value sits in a
// Config and, for a live row, how the Node applies a saved change.
type rowOf[T comparable] struct {
	at      func(*Config) *T
	applies *liveApply
}

func (r rowOf[T]) live() bool                   { return r.applies != nil }
func (r rowOf[T]) same(a, b *Config) bool       { return *r.at(a) == *r.at(b) }
func (r rowOf[T]) adopt(running, saved *Config) { *r.at(running) = *r.at(saved) }
func (r rowOf[T]) applier() *liveApply          { return r.applies }

func (r rowOf[T]) about(v view, a setting.About) setting.About {
	k := v.keys[a.Key]
	a.Source = string(k.Source)
	a.IsDefault = k.Source == config.SourceDefault
	a.Applies = "restart"
	if r.live() {
		a.Applies = "live"
	}
	if k.Source == config.SourceDotEnv || k.Source == config.SourceEnv {
		a.LockedBy = k.Env
	}
	a.PendingRestart = !r.same(v.saved, v.running)
	return a
}

// refused is a RangeError's sentence alone, for a config problem, which
// names the key already.
func refused(err error) error {
	if r, ok := errors.AsType[*setting.RangeError](err); ok {
		return errors.New(r.Sentence)
	}
	return err
}

func wrongForm(key, sentence string) error { return &setting.RangeError{Key: key, Sentence: sentence} }

// portRow edits http.listen's port. The API sends the port and keeps the
// saved address; the CLI sends a whole address.
type portRow struct {
	rowOf[netip.AddrPort]
	desc setting.Port
}

func (r portRow) key() string { return r.desc.Key }

func (r portRow) field(v view) setting.Field {
	d := r.desc
	d.Default = r.at(v.defaults).Port()
	saved := r.at(v.saved).Port()
	f := d.Field(&saved)
	f.Running = int64(r.at(v.running).Port())
	f.About = r.about(v, f.About)
	if v.boundHTTP != 0 {
		f.Running = int64(v.boundHTTP)
		f.PendingRestart = f.PendingRestart && saved != v.boundHTTP
	}
	return f
}

func (r portRow) text(v nodesystem.Value, saved *Config) (*string, error) {
	if v.IsReset() {
		return nil, nil
	}
	addr := *r.at(saved)
	port, ok := v.Number()
	if s, isText := v.Text(); isText {
		parsed, err := netip.ParseAddrPort(s)
		if err != nil {
			return nil, wrongForm(r.key(), "An address and port, such as [::]:8484.")
		}
		addr, port, ok = parsed, int64(parsed.Port()), true
	}
	if !ok {
		return nil, wrongForm(r.key(), "A port number.")
	}
	stored, err := r.desc.Parse(&port)
	if err != nil {
		return nil, err
	}
	s := netip.AddrPortFrom(addr.Addr(), *stored.Override()).String()
	return &s, nil
}

// check passes port 0, which binds any free port, since only a hand edit
// or the environment can set it.
func (r portRow) check(*Config) error { return nil }

type countRow struct {
	rowOf[int]
	desc setting.Count
}

func (r countRow) key() string { return r.desc.Key }

func (r countRow) field(v view) setting.Field {
	d := r.desc
	d.Default = int64(*r.at(v.defaults))
	saved := int64(*r.at(v.saved))
	f := d.Field(&saved)
	f.Running = int64(*r.at(v.running))
	f.About = r.about(v, f.About)
	return f
}

func (r countRow) text(v nodesystem.Value, _ *Config) (*string, error) {
	if v.IsReset() {
		return nil, nil
	}
	n, ok := v.Number()
	if s, isText := v.Text(); isText {
		parsed, err := strconv.ParseInt(s, 10, 64)
		n, ok = parsed, err == nil
	}
	if !ok {
		return nil, wrongForm(r.key(), "A whole number.")
	}
	if _, err := r.desc.Parse(&n); err != nil {
		return nil, err
	}
	s := strconv.FormatInt(n, 10)
	return &s, nil
}

func (r countRow) check(c *Config) error {
	n := int64(*r.at(c))
	_, err := r.desc.Parse(&n)
	return refused(err)
}

type choiceRow struct {
	rowOf[logs.Level]
	desc setting.Choice[logs.Level]
}

func (r choiceRow) key() string { return r.desc.Key }

func (r choiceRow) field(v view) setting.Field {
	d := r.desc
	d.Default = *r.at(v.defaults)
	f := d.Field(r.at(v.saved))
	f.Running = string(*r.at(v.running))
	f.About = r.about(v, f.About)
	return f
}

// text takes a Word or Text naming an option. Any other value is the empty
// word, which the Choice refuses with its options' names.
func (r choiceRow) text(v nodesystem.Value, _ *Config) (*string, error) {
	if v.IsReset() {
		return nil, nil
	}
	s, ok := v.Word()
	if !ok {
		s, _ = v.Text()
	}
	if _, err := r.desc.Parse(&s); err != nil {
		return nil, err
	}
	return &s, nil
}

func (r choiceRow) check(c *Config) error {
	s := string(*r.at(c))
	_, err := r.desc.Parse(&s)
	return refused(err)
}

type switchRow struct {
	rowOf[bool]
	desc setting.Switch
}

func (r switchRow) key() string { return r.desc.Key }

func (r switchRow) field(v view) setting.Field {
	d := r.desc
	d.Default = *r.at(v.defaults)
	f := d.Field(r.at(v.saved))
	f.Running = *r.at(v.running)
	f.About = r.about(v, f.About)
	return f
}

func (r switchRow) text(v nodesystem.Value, _ *Config) (*string, error) {
	if v.IsReset() {
		return nil, nil
	}
	b, ok := v.Switch()
	if s, isText := v.Text(); isText {
		parsed, err := strconv.ParseBool(s)
		b, ok = parsed, err == nil
	}
	if !ok {
		return nil, wrongForm(r.key(), "On or off: true or false.")
	}
	s := strconv.FormatBool(b)
	return &s, nil
}

func (r switchRow) check(*Config) error { return nil }

// configFile is a Node's config.yaml, loaded and written under the table.
// The host CLI uses it while it holds the data dir lock, and a running
// Node uses it for the API and the config control op.
type configFile struct {
	dir     DataDir
	environ []string
	types   contenttype.Set
	kind    install.Kind
}

func (e Env) configFile() configFile {
	return configFile{dir: e.DataDir, environ: e.Environ, types: e.Types, kind: e.Install}
}

func (f configFile) load() (Config, []Key, error) {
	return LoadConfig(f.dir, f.environ, f.types, f.kind)
}

// set applies every change or none, and returns the saved Config after. A
// change is refused with a *nodesystem.Refusal.
func (f configFile) set(changes []nodesystem.Change) (Config, []Key, error) {
	saved, keys, err := f.load()
	if err != nil || len(changes) == 0 {
		return saved, keys, err
	}
	sources := keysByName(keys)
	edits := make([]config.Change, len(changes))
	for i, c := range changes {
		r, ok := settingRowOf(c.Key)
		if !ok {
			return Config{}, nil, &nodesystem.Refusal{Key: c.Key, Reason: nodesystem.UnknownSetting, Sentence: "No setting has this key."}
		}
		if k := sources[c.Key]; k.Source == config.SourceDotEnv || k.Source == config.SourceEnv {
			return Config{}, nil, &nodesystem.Refusal{Key: c.Key, Reason: nodesystem.SetByEnvironment, Sentence: "Set by " + k.Env + ".", Variable: k.Env}
		}
		text, err := r.text(c.Value, &saved)
		if err != nil {
			return Config{}, nil, outOfRange(c.Key, err)
		}
		edits[i] = config.Change{Key: c.Key, Text: text}
	}
	err = config.Edit(f.dir.Path(), edits, func(fsys fs.FS) error {
		_, _, err := loadConfig(fsys, f.environ, f.types, f.kind)
		return err
	})
	if err != nil {
		return Config{}, nil, refusal(err)
	}
	return f.load()
}

func outOfRange(key string, err error) error {
	if r, ok := errors.AsType[*setting.RangeError](err); ok {
		return &nodesystem.Refusal{Key: key, Reason: nodesystem.OutOfRange, Sentence: r.Sentence}
	}
	return err
}

// refusal is why config.Edit refused, as a Refusal where it names one.
func refusal(err error) error {
	if errors.Is(err, config.ErrNotRegular) {
		return &nodesystem.Refusal{Reason: nodesystem.NotWritable, Sentence: "SlipStream can't rewrite config.yaml, a link or a mounted file. Edit it by hand."}
	}
	loadErr, ok := errors.AsType[*config.Error](err)
	if !ok {
		return err
	}
	p := loadErr.Problems[0]
	if used, ok := errors.AsType[*PortUsedError](p.Err); ok {
		return &nodesystem.Refusal{Key: p.Key, Reason: nodesystem.PortUsedByNode, Sentence: used.Sentence()}
	}
	return &nodesystem.Refusal{Key: p.Key, Reason: nodesystem.Invalid, Sentence: p.String()}
}
