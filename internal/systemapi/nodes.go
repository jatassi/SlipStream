package systemapi

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"slices"
	"strconv"
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/nodesystem"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/settingapi"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/system"
)

// Systems is where the per-Node operations reach every Node's system.
// Only a Controller admits an Admin, and it has one.
type Systems interface {
	Systems() (*nodesystem.Nodes, bool)
}

func open(s Systems) *nodesystem.Nodes {
	nodes, ok := s.Systems()
	if !ok {
		panic("systemapi: an Admin operation ran on a Node that is no Controller")
	}
	return nodes
}

var (
	nodeNotFound      = api.NewCode(http.StatusNotFound, "node_not_found", "No Node has this id")
	nodeOffline       = api.NewCode(http.StatusServiceUnavailable, "node_offline", "The Node is offline")
	nodeMismatched    = api.NewCode(http.StatusConflict, "node_mismatched", "The Node runs another version")
	alreadyRestarting = api.NewCode(http.StatusConflict, "already_restarting", "The Node is already restarting")
	setByEnvironment  = api.NewCode(http.StatusConflict, "set_by_environment", "An environment variable sets this setting")
	portUsedByNode    = api.NewCode(http.StatusUnprocessableEntity, "port_used_by_node", "The Node uses this port already")
	unknownSetting    = api.NewCode(http.StatusUnprocessableEntity, "unknown_setting", "No setting has this key")
	folderUnusable    = api.NewCode(http.StatusUnprocessableEntity, "folder_unusable", "SlipStream can't use this folder")
	configNotWritable = api.NewCode(http.StatusConflict, "config_not_writable", "SlipStream can't rewrite config.yaml")
)

// reach is every code a call to a Node answers before its own.
var reach = []*api.Code{nodeNotFound, nodeOffline, nodeMismatched}

// answer is a Node's domain error as the code the operation answers.
func answer(err error) error {
	if r, ok := errors.AsType[*nodesystem.Refusal](err); ok {
		return refused(r)
	}
	for _, m := range []struct {
		err  error
		code *api.Code
	}{
		{nodesystem.ErrNotFound, nodeNotFound},
		{nodesystem.ErrOffline, nodeOffline},
		{nodesystem.ErrMismatched, nodeMismatched},
		{nodesystem.ErrAlreadyRestarting, alreadyRestarting},
	} {
		if errors.Is(err, m.err) {
			return m.code
		}
	}
	return err
}

func refused(r *nodesystem.Refusal) error {
	switch r.Reason {
	case nodesystem.SetByEnvironment:
		return setByEnvironment.On(r.Key, r.Sentence)
	case nodesystem.OutOfRange:
		return settingapi.ErrOutOfRange.On(r.Key, r.Sentence)
	case nodesystem.PortUsedByNode:
		return portUsedByNode.On(r.Key, r.Sentence)
	case nodesystem.UnknownSetting:
		return unknownSetting.On(r.Key, r.Sentence)
	case nodesystem.NotWritable:
		return configNotWritable.Because(r.Sentence)
	case nodesystem.UnusableFolder:
		return folderUnusable.On(r.Key, r.Sentence)
	case nodesystem.Invalid:
	}
	return api.Invalid.On(r.Key, r.Sentence)
}

type idInput struct {
	ID string `path:"id" format:"uuid"`
}

func nodeID(s string) (role.NodeID, error) {
	u, err := storage.ParseUUID(s)
	if err != nil {
		return role.NodeID{}, nodeNotFound
	}
	return role.NodeID{UUID: u}, nil
}

type nodeConfig struct {
	Sections []configSection `json:"sections" nullable:"false"`
}

type configSection struct {
	ID     string                        `json:"id" enum:"web,log,torrent"`
	Fields []api.OneOf[settingapi.Field] `json:"fields" nullable:"false"`
}

type configOutput struct {
	Body nodeConfig
}

func configOf(c nodesystem.Config) *configOutput {
	out := nodeConfig{Sections: make([]configSection, len(c.Sections))}
	for i, s := range c.Sections {
		fields := make([]api.OneOf[settingapi.Field], len(s.Fields))
		for j, f := range s.Fields {
			fields[j] = settingapi.FieldOf(f)
		}
		out.Sections[i] = configSection{ID: string(s.ID), Fields: fields}
	}
	return &configOutput{Body: out}
}

type configInput struct {
	ID   string         `path:"id" format:"uuid"`
	Body map[string]any `doc:"Each key's new value: a number, a word, true or false, or null to reset it to its default"`
}

// changes is the body's keys in order, each value as the setting kinds
// read it.
func (in *configInput) changes() ([]nodesystem.Change, error) {
	keys := make([]string, 0, len(in.Body))
	for k := range in.Body {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	out := make([]nodesystem.Change, len(keys))
	for i, k := range keys {
		c := nodesystem.Change{Key: k}
		switch v := in.Body[k].(type) {
		case nil:
			c.Value = nodesystem.Reset()
		case bool:
			c.Value = nodesystem.Switch(v)
		case string:
			c.Value = nodesystem.Word(v)
		case float64:
			if v != math.Trunc(v) || math.Abs(v) > 1<<53 {
				return nil, api.Invalid.On(k, "A whole number.")
			}
			c.Value = nodesystem.Number(int64(v))
		default:
			return nil, api.Invalid.On(k, "A number, a word, true or false, or null.")
		}
		out[i] = c
	}
	return out, nil
}

type restarting struct {
	StartedAt time.Time `json:"startedAt" doc:"When the stopping process started; its successor's startedAt is later"`
	BoundPort int       `json:"boundPort" doc:"The port the stopping process serves on"`
	NextPort  int       `json:"nextPort,omitempty" doc:"The port the successor binds, when http.listen waits for this restart"`
}

type restartOutput struct {
	Body restarting
}

type logsInput struct {
	ID          string `path:"id" format:"uuid"`
	LastEventID string `header:"Last-Event-ID" doc:"The seq of the last record a dropped stream received; it wins over after"`
	After       uint64 `query:"after" doc:"Send the records after this seq"`
}

// after is where the stream resumes.
func (in *logsInput) after() (uint64, error) {
	if in.LastEventID == "" {
		return in.After, nil
	}
	after, err := strconv.ParseUint(in.LastEventID, 10, 64)
	if err != nil {
		return 0, api.Invalid.Because("Last-Event-ID is not a log record's seq.")
	}
	return after, nil
}

type logRecord struct {
	Seq       int64           `json:"seq"`
	Time      time.Time       `json:"time"`
	Level     string          `json:"level" enum:"debug,info,warn,error"`
	Subsystem string          `json:"subsystem"`
	Message   string          `json:"msg"`
	Attrs     json.RawMessage `json:"attrs" doc:"The record's other attributes"`
}

type logReset struct {
	Reason    string    `json:"reason" enum:"lapped,restarted"`
	StartedAt time.Time `json:"startedAt" doc:"When the serving process started"`
}

var logEvents = []api.StreamEvent{
	{Name: "log.record", Payload: logRecord{}},
	{Name: "log.reset", Payload: logReset{}},
}

// seq is a record's seq as an event id: its process's start in Unix
// microseconds plus a count, far below the int64 limit.
func seq(s uint64) int64 { return int64(min(s, math.MaxInt64)) }

// sendBatch writes a reset, then each record with its seq as the event's
// id, or a keepalive for an empty batch. A reset's id is the seq the next
// record follows, so a reconnect after it resumes there.
func sendBatch(send api.Send, b system.LogBatch) error {
	if r := b.Reset; r != nil {
		if err := send(api.StreamMessage{ID: seq(r.After), Data: logReset{Reason: r.Reason.String(), StartedAt: r.StartedAt}}); err != nil {
			return err
		}
	}
	if len(b.Records) == 0 && b.Reset == nil {
		return send(api.StreamMessage{Comment: "keepalive"})
	}
	for _, r := range b.Records {
		attrs := r.Attrs
		if len(attrs) == 0 {
			attrs = json.RawMessage(`{}`)
		}
		if err := send(api.StreamMessage{ID: seq(r.Seq), Data: logRecord{
			Seq:       seq(r.Seq),
			Time:      r.Time,
			Level:     r.Level,
			Subsystem: r.Subsystem,
			Message:   r.Message,
			Attrs:     attrs,
		}}); err != nil {
			return err
		}
	}
	return nil
}

type nodeHandlers struct{ systems Systems }

func (h nodeHandlers) getNodeSystem(ctx context.Context, _ identity.Admin, in *idInput) (*reportOutput, error) {
	id, err := nodeID(in.ID)
	if err != nil {
		return nil, err
	}
	f, err := open(h.systems).Facts(ctx, id)
	if err != nil {
		return nil, answer(err)
	}
	return &reportOutput{Body: reportOf(f)}, nil
}

func (h nodeHandlers) getNodeConfig(ctx context.Context, _ identity.Admin, in *idInput) (*configOutput, error) {
	id, err := nodeID(in.ID)
	if err != nil {
		return nil, err
	}
	c, err := open(h.systems).Config(ctx, id)
	if err != nil {
		return nil, answer(err)
	}
	return configOf(c), nil
}

func (h nodeHandlers) updateNodeConfig(ctx context.Context, _ identity.Admin, in *configInput) (*configOutput, error) {
	id, err := nodeID(in.ID)
	if err != nil {
		return nil, err
	}
	changes, err := in.changes()
	if err != nil {
		return nil, err
	}
	c, err := open(h.systems).SetConfig(ctx, id, changes)
	if err != nil {
		return nil, answer(err)
	}
	return configOf(c), nil
}

func (h nodeHandlers) restartNode(ctx context.Context, _ identity.Admin, in *idInput) (*restartOutput, error) {
	id, err := nodeID(in.ID)
	if err != nil {
		return nil, err
	}
	r, err := open(h.systems).Restart(ctx, id)
	if err != nil {
		return nil, answer(err)
	}
	return &restartOutput{Body: restarting{StartedAt: r.StartedAt, BoundPort: int(r.BoundPort), NextPort: int(r.NextPort)}}, nil
}

// streamNodeLogs answers a Node it can't reach with a problem, then
// follows its log through the Node going offline and back.
func (h nodeHandlers) streamNodeLogs(ctx context.Context, _ identity.Admin, in *logsInput) (api.StreamFunc, error) {
	id, err := nodeID(in.ID)
	if err != nil {
		return nil, err
	}
	after, err := in.after()
	if err != nil {
		return nil, err
	}
	nodes := open(h.systems)
	if _, err := nodes.Facts(ctx, id); err != nil {
		return nil, answer(err)
	}
	return func(ctx context.Context, send api.Send) error {
		return nodes.Follow(ctx, id, after, func(b system.LogBatch) error { return sendBatch(send, b) })
	}, nil
}

func (h nodeHandlers) downloadNodeLogs(ctx context.Context, _ identity.Admin, in *idInput) (*api.Download, error) {
	id, err := nodeID(in.ID)
	if err != nil {
		return nil, err
	}
	a, err := open(h.systems).Archive(ctx, id)
	if err != nil {
		return nil, answer(err)
	}
	return &api.Download{Name: a.Name, Size: a.Size, Body: a.Body}, nil
}
