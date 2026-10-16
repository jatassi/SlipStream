package nodesystem

import (
	"encoding/json"
	"fmt"
	"net/netip"
	"time"

	"github.com/jatassi/slipstream/internal/buildinfo"
	systemv1 "github.com/jatassi/slipstream/internal/pb/slipstream/system/v1"
	"github.com/jatassi/slipstream/internal/system"
)

func factsToWire(f system.Facts) *systemv1.GetFactsResponse {
	out := &systemv1.GetFactsResponse{
		DataDir:           f.DataDir,
		LogsDir:           f.LogsDir,
		Pid:               int64(f.PID),
		Seed:              f.Seed,
		StartedAtUnixNano: f.StartedAt.UnixNano(),
		HttpConfigured:    f.HTTPConfigured.String(),
		Http:              f.Listeners.HTTP.String(),
		Torrent:           f.Listeners.Torrent.String(),
		Link:              f.Listeners.Link.String(),
		Control:           f.Listeners.Control,
	}
	if b := f.Build; b != nil {
		out.Build = &systemv1.Build{
			Version:      b.Version,
			Commit:       b.Commit,
			Fingerprint:  b.Fingerprint,
			Target:       b.Target,
			NativeBundle: b.NativeBundle,
			Libtorrent:   b.Libtorrent,
		}
	}
	return out
}

func factsFromWire(f *systemv1.GetFactsResponse) (system.Facts, error) {
	out := system.Facts{
		DataDir:   f.GetDataDir(),
		LogsDir:   f.GetLogsDir(),
		PID:       int(f.GetPid()),
		Seed:      f.GetSeed(),
		StartedAt: time.Unix(0, f.GetStartedAtUnixNano()),
		Listeners: system.Listeners{Control: f.GetControl()},
	}
	for _, a := range []struct {
		to   *netip.AddrPort
		from string
	}{
		{&out.HTTPConfigured, f.GetHttpConfigured()},
		{&out.Listeners.HTTP, f.GetHttp()},
		{&out.Listeners.Torrent, f.GetTorrent()},
		{&out.Listeners.Link, f.GetLink()},
	} {
		addr, err := netip.ParseAddrPort(a.from)
		if err != nil {
			return system.Facts{}, fmt.Errorf("read a Node's facts: %w", err)
		}
		*a.to = addr
	}
	if b := f.GetBuild(); b != nil {
		out.Build = &buildinfo.Stamp{
			Version:      b.GetVersion(),
			Commit:       b.GetCommit(),
			Fingerprint:  b.GetFingerprint(),
			Target:       b.GetTarget(),
			NativeBundle: b.GetNativeBundle(),
			Libtorrent:   b.GetLibtorrent(),
		}
	}
	return out, nil
}

func configToWire(c Config) *systemv1.Config {
	out := &systemv1.Config{}
	for _, s := range c.Sections {
		fields := make([][]byte, len(s.Fields))
		for i, f := range s.Fields {
			fields[i] = f
		}
		out.Sections = append(out.Sections, &systemv1.ConfigSection{Id: string(s.ID), Fields: fields})
	}
	return out
}

func configFromWire(c *systemv1.Config) Config {
	var out Config
	for _, s := range c.GetSections() {
		fields := make([]json.RawMessage, len(s.GetFields()))
		for i, f := range s.GetFields() {
			fields[i] = f
		}
		out.Sections = append(out.Sections, Section{ID: SectionID(s.GetId()), Fields: fields})
	}
	return out
}

func changeToWire(c Change) *systemv1.Change {
	out := &systemv1.Change{Key: c.Key}
	switch v := c.Value; v.form {
	case reset:
		out.Value = &systemv1.Change_Reset_{Reset_: true}
	case number:
		out.Value = &systemv1.Change_Number{Number: v.n}
	case word:
		out.Value = &systemv1.Change_Word{Word: v.s}
	case toggle:
		out.Value = &systemv1.Change_Switch{Switch: v.b}
	case text:
		out.Value = &systemv1.Change_Text{Text: v.s}
	}
	return out
}

func changeFromWire(c *systemv1.Change) Change {
	out := Change{Key: c.GetKey()}
	switch v := c.GetValue().(type) {
	case *systemv1.Change_Number:
		out.Value = Number(v.Number)
	case *systemv1.Change_Word:
		out.Value = Word(v.Word)
	case *systemv1.Change_Switch:
		out.Value = Switch(v.Switch)
	case *systemv1.Change_Text:
		out.Value = Text(v.Text)
	}
	return out
}

func refusalToWire(r *Refusal) *systemv1.Refusal {
	return &systemv1.Refusal{Key: r.Key, Reason: r.Reason.String(), Sentence: r.Sentence, Variable: r.Variable}
}

func refusalFromWire(r *systemv1.Refusal) (*Refusal, error) {
	out := &Refusal{Key: r.GetKey(), Sentence: r.GetSentence(), Variable: r.GetVariable()}
	if err := out.Reason.UnmarshalText([]byte(r.GetReason())); err != nil {
		return nil, err
	}
	return out, nil
}

func batchToWire(b system.LogBatch) *systemv1.ReadLogsResponse {
	out := &systemv1.ReadLogsResponse{Records: make([]*systemv1.LogRecord, len(b.Records))}
	if r := b.Reset; r != nil {
		out.Reset_ = &systemv1.LogReset{Reason: r.Reason.String(), StartedAtUnixNano: r.StartedAt.UnixNano(), After: r.After}
	}
	for i, r := range b.Records {
		out.Records[i] = &systemv1.LogRecord{
			Seq:          r.Seq,
			TimeUnixNano: r.Time.UnixNano(),
			Level:        r.Level,
			Subsystem:    r.Subsystem,
			Message:      r.Message,
			AttrsJson:    r.Attrs,
		}
	}
	return out
}

func batchFromWire(b *systemv1.ReadLogsResponse) (system.LogBatch, error) {
	var out system.LogBatch
	if r := b.GetReset_(); r != nil {
		out.Reset = &system.LogReset{StartedAt: time.Unix(0, r.GetStartedAtUnixNano()), After: r.GetAfter()}
		if err := out.Reset.Reason.UnmarshalText([]byte(r.GetReason())); err != nil {
			return system.LogBatch{}, err
		}
	}
	for _, r := range b.GetRecords() {
		out.Records = append(out.Records, system.LogRecord{
			Seq:       r.GetSeq(),
			Time:      time.Unix(0, r.GetTimeUnixNano()),
			Level:     r.GetLevel(),
			Subsystem: r.GetSubsystem(),
			Message:   r.GetMessage(),
			Attrs:     r.GetAttrsJson(),
		})
	}
	return out, nil
}
