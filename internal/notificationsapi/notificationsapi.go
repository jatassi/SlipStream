// Package notificationsapi is the notification connections, the bell, its
// feed and the Notifications settings. Every operation is the Admin's.
package notificationsapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/history/entry"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/notifications"
	"github.com/jatassi/slipstream/internal/notifications/providers"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/settingapi"
	"github.com/jatassi/slipstream/internal/storage"
)

// Notes is the Controller's notifications, false on a Node that is no
// Controller.
type Notes interface {
	Notifications() (*notifications.Service, bool)
}

var (
	notFound     = api.NewCode(http.StatusNotFound, "notification_connection_not_found", "No notification connection has this id")
	unknownEvent = api.NewCode(http.StatusUnprocessableEntity, "unknown_event", "One of the events isn't in the list")
	fieldRefused = api.NewCarrying[string](http.StatusUnprocessableEntity, "notification_field_refused", "A field of the connection can't be saved", "field")
	testFailed   = api.NewCarrying[string](http.StatusUnprocessableEntity, "notification_test_failed", "The Test failed", "field")
)

type provider struct {
	ID     string  `json:"id"`
	Label  string  `json:"label"`
	Image  string  `json:"image" enum:"none,bytes,publicUrl,bodyUrl" doc:"How the provider takes a poster"`
	Fields []field `json:"fields" nullable:"false" doc:"In the order the sheet shows them"`
}

type field struct {
	Key      string          `json:"key"`
	Label    string          `json:"label"`
	Kind     string          `json:"kind" enum:"text,url,number,select,switch,list,headers,node,duration"`
	Secret   bool            `json:"secret" doc:"Its value is never returned"`
	Optional bool            `json:"optional"`
	Details  bool            `json:"details" doc:"Shown under Details in the sheet"`
	Default  json.RawMessage `json:"default,omitempty"`
	Help     string          `json:"help"`
	Options  []option        `json:"options,omitempty" doc:"A select's choices"`
	Min      *int            `json:"min,omitempty" doc:"A number's lowest value, or a duration's, in seconds"`
	Max      *int            `json:"max,omitempty" doc:"A number's highest value, or a duration's, in seconds"`
}

type option struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

type event struct {
	ID            string `json:"id"`
	Label         string `json:"label"`
	Group         string `json:"group" enum:"grabs,imports,library,subtitles,health,backups,moves,conversions,system"`
	OnByDefault   bool   `json:"onByDefault" doc:"A new connection gets it unless the Admin picks otherwise"`
	BellByDefault bool   `json:"bellByDefault"`
	Fixed         bool   `json:"fixed" doc:"Never chosen: a Test is sent only when the Admin asks"`
}

type connection struct {
	ID       string                     `json:"id" format:"uuid"`
	Name     string                     `json:"name"`
	Provider string                     `json:"provider"`
	On       bool                       `json:"on"`
	Events   []string                   `json:"events" nullable:"false"`
	Warnings bool                       `json:"warnings" doc:"Whether it gets Health warnings as well as errors"`
	Posters  bool                       `json:"posters"`
	Node     api.Nullable[string]       `json:"node" doc:"The Node a Custom Script runs on; null for every other provider"`
	Values   map[string]json.RawMessage `json:"values" doc:"The provider's settings that aren't secret"`
	Sealed   []string                   `json:"sealed" nullable:"false" doc:"The secret fields that have a value. Secrets are never returned"`
	Shown    map[string]string          `json:"shown" doc:"A part of a secret value that is safe to show, such as a webhook URL's host"`
	Status   status                     `json:"status"`
}

type status struct {
	Shown    string                  `json:"shown" enum:"off,ok,failing,waiting,paused,holding"`
	Since    api.Nullable[time.Time] `json:"since" doc:"When it started failing, waiting or paused; null otherwise"`
	Until    api.Nullable[time.Time] `json:"until" doc:"When a waiting connection sends again; null otherwise"`
	Reason   string                  `json:"reason" doc:"Why the connection isn't sending, in a sentence. Empty when it is sending."`
	Waiting  int                     `json:"waiting" doc:"Messages waiting to be sent."`
	GivenUp  int                     `json:"givenUp" doc:"Messages given up since the connection was last saved or tested, because they waited too long or the provider refused them."`
	LastSent api.Nullable[time.Time] `json:"lastSent"`
}

type draftBody struct {
	Name     string                     `json:"name" maxLength:"100"`
	Provider string                     `json:"provider" doc:"One of the ids listNotificationProviders lists"`
	On       bool                       `json:"on,omitempty"`
	Events   []string                   `json:"events,omitempty" doc:"The events it gets; left out, the recommended ones"`
	Warnings bool                       `json:"warnings,omitempty" doc:"Whether it gets Health warnings as well as errors"`
	Posters  bool                       `json:"posters,omitempty"`
	Node     string                     `json:"node,omitempty" format:"uuid" doc:"The Node a Custom Script runs on; left out for every other provider"`
	Values   map[string]json.RawMessage `json:"values,omitempty" doc:"The provider's settings that aren't secret, by key"`
	Secrets  map[string]string          `json:"secrets,omitempty" doc:"The secret settings by key. Write-only"`
}

type editBody struct {
	Name     api.Patch[string]            `json:"name,omitempty"`
	On       api.Patch[bool]              `json:"on,omitempty"`
	Events   api.Patch[[]string]          `json:"events,omitempty"`
	Warnings api.Patch[bool]              `json:"warnings,omitempty"`
	Posters  api.Patch[bool]              `json:"posters,omitempty"`
	Node     api.Patch[string]            `json:"node,omitempty" doc:"A Custom Script's Node id"`
	Values   map[string]json.RawMessage   `json:"values,omitempty" doc:"Settings to change, by key; the others keep their values"`
	Secrets  map[string]api.Patch[string] `json:"secrets,omitempty" doc:"A secret to replace. Left out or empty keeps it; null removes an optional one. Write-only"`
}

type idInput struct {
	ID string `path:"id" format:"uuid"`
}

type draftInput struct{ Body draftBody }

type editInput struct {
	ID   string `path:"id" format:"uuid"`
	Body editBody
}

type providersOutput struct {
	Body struct {
		Providers []provider `json:"providers" nullable:"false"`
	}
}

type eventsOutput struct {
	Body struct {
		Events []event `json:"events" nullable:"false" doc:"Grouped by group, in the order the sheet shows them"`
	}
}

type connectionsOutput struct {
	Body struct {
		Connections []connection `json:"connections" nullable:"false" doc:"By name"`
	}
}

type connectionOutput struct{ Body connection }

type testPassed struct {
	Node string `json:"node" doc:"The Node that sent the Test"`
}

type testOutput struct{ Body testPassed }

type bell struct {
	Events   []string `json:"events" nullable:"false"`
	Warnings bool     `json:"warnings" doc:"Whether it shows Health warnings as well as errors"`
}

type bellOutput struct{ Body bell }

type bellInput struct {
	Body struct {
		Events   []string `json:"events,omitempty" doc:"The events it shows; left out, unchanged"`
		Warnings *bool    `json:"warnings,omitempty"`
	}
}

type feedInput struct {
	Before int64 `query:"before" minimum:"0" doc:"The before of the previous page; left out for the newest"`
	Limit  int   `query:"limit" minimum:"1" maximum:"100" default:"50"`
}

type feedRow struct {
	Seq    int64                `json:"seq"`
	At     time.Time            `json:"at"`
	Event  string               `json:"event"`
	Title  string               `json:"title"`
	Line   string               `json:"line" doc:"The message's first line; empty for none"`
	Link   api.Nullable[string] `json:"link" doc:"Where the row leads; null while no Address is set"`
	Unread api.Nullable[bool]   `json:"unread" doc:"Null for a request made with the API key"`
}

type feed struct {
	Rows   []feedRow           `json:"rows" nullable:"false" doc:"Newest first"`
	Unread api.Nullable[int]   `json:"unread" doc:"How many rows are unread, at most 100; null for a request made with the API key"`
	Before api.Nullable[int64] `json:"before" doc:"The cursor for the next page; null at the end"`
}

type feedOutput struct{ Body feed }

type seenInput struct {
	Body struct {
		Through int64 `json:"through" minimum:"0" doc:"The seq of the newest row the Admin has seen"`
	}
}

type settingsOutput struct {
	Body struct {
		Settings []api.OneOf[settingapi.Field] `json:"settings" nullable:"false" doc:"Wait before sending a new Health item, Give up on a message after and Warn when a connection has failed for"`
	}
}

type settingsInput struct {
	Body struct {
		Values map[string]json.RawMessage `json:"values" doc:"Each setting's new value by key, all or none; null resets one to its default"`
	}
}

type handlers struct{ notes Notes }

func (h handlers) svc() *notifications.Service {
	s, ok := h.notes.Notifications()
	if !ok {
		panic("notificationsapi: an operation ran on a Node that is no Controller")
	}
	return s
}

// answer is err as the problem an Admin reads.
func answer(err error) error {
	if f, ok := errors.AsType[*notifications.FieldError](err); ok {
		return fieldRefused.Explained(f.Reason, f.Field)
	}
	if _, ok := errors.AsType[*notifications.UnknownEventError](err); ok {
		return unknownEvent
	}
	if errors.Is(err, notifications.ErrNotFound) {
		return notFound
	}
	return settingapi.Refused(err)
}

func connectionID(raw string) (notifications.ConnectionID, error) {
	u, err := storage.ParseUUID(raw)
	if err != nil {
		return notifications.ConnectionID{}, notFound
	}
	return notifications.ConnectionID{UUID: u}, nil
}

func nodeID(raw string) (role.NodeID, error) {
	u, err := storage.ParseUUID(raw)
	if err != nil {
		return role.NodeID{}, fieldRefused.Explained("Pick the Node that runs this script.", "node")
	}
	return role.NodeID{UUID: u}, nil
}

func kinds(ids []string) []entry.Kind {
	out := make([]entry.Kind, len(ids))
	for i, id := range ids {
		out[i] = entry.Kind(id)
	}
	return out
}

func (b draftBody) draft() (notifications.Draft, error) {
	d := notifications.Draft{
		Name: b.Name, Provider: providers.ID(b.Provider), On: b.On, Warnings: b.Warnings, Posters: b.Posters,
		Settings: b.Values, Secrets: b.Secrets,
	}
	if b.Events != nil {
		d.Events = kinds(b.Events)
	}
	if b.Node != "" {
		n, err := nodeID(b.Node)
		if err != nil {
			return d, err
		}
		d.Node = n
	}
	return d, nil
}

func (b editBody) edit() (notifications.Edit, error) {
	e := notifications.Edit{
		Name: b.Name.Value(), On: b.On.Value(), Warnings: b.Warnings.Value(), Posters: b.Posters.Value(), Settings: b.Values,
	}
	if v := b.Events.Value(); v != nil {
		events := kinds(*v)
		e.Events = &events
	}
	if v := b.Node.Value(); v != nil {
		n, err := nodeID(*v)
		if err != nil {
			return e, err
		}
		e.Node = &n
	}
	if b.Secrets != nil {
		e.Secrets = map[string]*string{}
		for k, v := range b.Secrets {
			e.Secrets[k] = v.Value()
		}
	}
	return e, nil
}

var imageWords = map[providers.ImageWay]string{
	providers.NoImage: "none", providers.Bytes: "bytes", providers.PublicURL: "publicUrl", providers.BodyURL: "bodyUrl",
}

func (h handlers) listProviders(context.Context, identity.Admin, *struct{}) (*providersOutput, error) {
	out := &providersOutput{}
	out.Body.Providers = []provider{}
	for _, p := range h.svc().Providers() {
		w := provider{ID: string(p.ID), Label: p.Label, Image: imageWords[p.Image], Fields: []field{}}
		for _, f := range p.Fields {
			wf := field{
				Key: f.Key, Label: f.Label, Kind: string(f.Kind), Secret: f.Secret, Optional: f.Optional, Details: f.Details,
				Default: f.Default, Help: f.Help, Min: f.Min, Max: f.Max,
			}
			for _, o := range f.Options {
				wf.Options = append(wf.Options, option{Value: o.Value, Label: o.Label})
			}
			w.Fields = append(w.Fields, wf)
		}
		out.Body.Providers = append(out.Body.Providers, w)
	}
	return out, nil
}

func (h handlers) listEvents(context.Context, identity.Admin, *struct{}) (*eventsOutput, error) {
	out := &eventsOutput{}
	out.Body.Events = []event{}
	for _, s := range h.svc().Events() {
		out.Body.Events = append(out.Body.Events, event{
			ID: string(s.Kind), Label: s.Label, Group: string(s.Group), OnByDefault: s.OnByDefault, BellByDefault: s.BellByDefault, Fixed: s.Fixed,
		})
	}
	return out, nil
}

func timeOf(t time.Time) api.Nullable[time.Time] {
	if t.IsZero() {
		return api.Nullable[time.Time]{}
	}
	return api.Some(t)
}

func (h handlers) wire(v notifications.View) connection {
	d := h.svc().Describe(v)
	c := connection{
		ID: v.ID.String(), Name: v.Name, Provider: string(v.Provider), On: v.On, Events: []string{}, Warnings: v.Warnings,
		Posters: v.Posters, Values: v.Settings, Sealed: d.Sealed, Shown: d.Shown,
		Status: status{
			Shown: string(v.Status.Shown), Since: timeOf(v.Status.Since), Until: timeOf(v.Status.Until), Reason: d.Reason,
			Waiting: v.Status.Waiting, GivenUp: v.Status.GivenUp, LastSent: timeOf(v.Status.LastSent),
		},
	}
	for _, k := range v.Events {
		c.Events = append(c.Events, string(k))
	}
	if !v.Node.IsZero() {
		c.Node = api.Some(v.Node.String())
	}
	return c
}

func (h handlers) one(v notifications.View, err error) (*connectionOutput, error) {
	if err != nil {
		return nil, answer(err)
	}
	return &connectionOutput{Body: h.wire(v)}, nil
}

func (h handlers) listConnections(ctx context.Context, _ identity.Admin, _ *struct{}) (*connectionsOutput, error) {
	views, err := h.svc().Connections(ctx)
	if err != nil {
		return nil, err
	}
	out := &connectionsOutput{}
	out.Body.Connections = make([]connection, len(views))
	for i, v := range views {
		out.Body.Connections[i] = h.wire(v)
	}
	return out, nil
}

func (h handlers) getConnection(ctx context.Context, _ identity.Admin, in *idInput) (*connectionOutput, error) {
	id, err := connectionID(in.ID)
	if err != nil {
		return nil, err
	}
	return h.one(h.svc().Connection(ctx, id))
}

func (h handlers) addConnection(ctx context.Context, _ identity.Admin, in *draftInput) (*connectionOutput, error) {
	d, err := in.Body.draft()
	if err != nil {
		return nil, err
	}
	return h.one(h.svc().Create(ctx, d))
}

func (h handlers) updateConnection(ctx context.Context, _ identity.Admin, in *editInput) (*connectionOutput, error) {
	id, err := connectionID(in.ID)
	if err != nil {
		return nil, err
	}
	e, err := in.Body.edit()
	if err != nil {
		return nil, err
	}
	return h.one(h.svc().Edit(ctx, id, e))
}

func (h handlers) deleteConnection(ctx context.Context, _ identity.Admin, in *idInput) (*struct{}, error) {
	id, err := connectionID(in.ID)
	if err != nil {
		return nil, err
	}
	if err := h.svc().Delete(ctx, id); err != nil {
		return nil, answer(err)
	}
	return &struct{}{}, nil
}

func tested(res notifications.TestResult, err error) (*testOutput, error) {
	if err != nil {
		return nil, answer(err)
	}
	if !res.Passed {
		return nil, testFailed.Explained(res.Reason, res.Field)
	}
	return &testOutput{Body: testPassed{Node: res.Node}}, nil
}

func (h handlers) testDraft(ctx context.Context, _ identity.Admin, in *draftInput) (*testOutput, error) {
	d, err := in.Body.draft()
	if err != nil {
		return nil, err
	}
	return tested(h.svc().Test(ctx, d))
}

func (h handlers) testSaved(ctx context.Context, _ identity.Admin, in *editInput) (*testOutput, error) {
	id, err := connectionID(in.ID)
	if err != nil {
		return nil, err
	}
	e, err := in.Body.edit()
	if err != nil {
		return nil, err
	}
	return tested(h.svc().TestSaved(ctx, id, e))
}

func bellOf(b notifications.Bell, err error) (*bellOutput, error) {
	if err != nil {
		return nil, answer(err)
	}
	out := &bellOutput{Body: bell{Events: []string{}, Warnings: b.Warnings}}
	for _, k := range b.Events {
		out.Body.Events = append(out.Body.Events, string(k))
	}
	return out, nil
}

func (h handlers) getBell(ctx context.Context, _ identity.Admin, _ *struct{}) (*bellOutput, error) {
	return bellOf(h.svc().Bell(ctx))
}

func (h handlers) updateBell(ctx context.Context, _ identity.Admin, in *bellInput) (*bellOutput, error) {
	var events []entry.Kind
	if in.Body.Events != nil {
		events = kinds(in.Body.Events)
	}
	return bellOf(h.svc().SetBell(ctx, events, in.Body.Warnings))
}

func (h handlers) listFeed(ctx context.Context, by identity.Admin, in *feedInput) (*feedOutput, error) {
	f, err := h.svc().Feed(ctx, by, notifications.Seq(in.Before), in.Limit)
	if err != nil {
		return nil, err
	}
	_, account := by.Account()
	out := &feedOutput{Body: feed{Rows: make([]feedRow, len(f.Rows))}}
	for i, r := range f.Rows {
		row := feedRow{Seq: int64(r.Seq), At: r.At, Event: string(r.Kind), Title: r.Title, Line: r.Line}
		if r.Link != "" {
			row.Link = api.Some(r.Link)
		}
		if account {
			row.Unread = api.Some(!r.Read)
		}
		out.Body.Rows[i] = row
	}
	if f.Unread != nil {
		out.Body.Unread = api.Some(*f.Unread)
	}
	if f.Before != 0 {
		out.Body.Before = api.Some(int64(f.Before))
	}
	return out, nil
}

func (h handlers) markSeen(ctx context.Context, by identity.Admin, in *seenInput) (*struct{}, error) {
	if err := h.svc().Seen(ctx, by, notifications.Seq(in.Body.Through)); errors.Is(err, notifications.ErrNeedsAccount) {
		return nil, api.NeedsAccount
	} else if err != nil {
		return nil, err
	}
	return &struct{}{}, nil
}

func settingsOf(ctx context.Context, p setting.Panel) (*settingsOutput, error) {
	fields, err := p.Fields(ctx)
	if err != nil {
		return nil, err
	}
	out := &settingsOutput{}
	out.Body.Settings = settingapi.FieldsOf(fields)
	return out, nil
}

func (h handlers) getSettings(ctx context.Context, _ identity.Admin, _ *struct{}) (*settingsOutput, error) {
	return settingsOf(ctx, h.svc().Settings())
}

func (h handlers) updateSettings(ctx context.Context, _ identity.Admin, in *settingsInput) (*settingsOutput, error) {
	p := h.svc().Settings()
	if err := p.Apply(ctx, in.Body.Values); err != nil {
		return nil, settingapi.Refused(err)
	}
	return settingsOf(ctx, p)
}

func Routes(r *api.Router, notes Notes) {
	o := r.Owner("notifications", api.Admin)
	h := handlers{notes}
	const base = "/notifications"
	saving := []*api.Code{fieldRefused.Code}
	api.Register(o, api.Admin, api.Op{
		ID: "listNotificationProviders", Method: http.MethodGet, Path: base + "/providers", Status: http.StatusOK,
		Summary: "List the providers a notification connection can send through, each with its fields",
	}, h.listProviders)
	api.Register(o, api.Admin, api.Op{
		ID: "listNotificationEvents", Method: http.MethodGet, Path: base + "/events", Status: http.StatusOK,
		Summary: "List the events a connection or the bell can get",
	}, h.listEvents)
	api.Register(o, api.Admin, api.Op{
		ID: "listNotificationConnections", Method: http.MethodGet, Path: base + "/connections", Status: http.StatusOK,
		Summary: "List the notification connections, each with whether it is sending",
	}, h.listConnections)
	api.Register(o, api.Admin, api.Op{
		ID: "addNotificationConnection", Method: http.MethodPost, Path: base + "/connections", Status: http.StatusCreated,
		Summary: "Add a notification connection",
		Errors:  saving,
	}, h.addConnection)
	api.Register(o, api.Admin, api.Op{
		ID: "testNotificationDraft", Method: http.MethodPost, Path: base + "/connections/test", Status: http.StatusOK,
		Summary: "Send a Test message to a connection that isn't saved yet",
		Errors:  []*api.Code{fieldRefused.Code, testFailed.Code},
	}, h.testDraft)
	api.Register(o, api.Admin, api.Op{
		ID: "getNotificationConnection", Method: http.MethodGet, Path: base + "/connections/{id}", Status: http.StatusOK,
		Summary: "Get a notification connection with whether it is sending",
		Errors:  []*api.Code{notFound},
	}, h.getConnection)
	api.Register(o, api.Admin, api.Op{
		ID: "updateNotificationConnection", Method: http.MethodPatch, Path: base + "/connections/{id}", Status: http.StatusOK,
		Summary: "Change a notification connection, which also sends the messages it held while paused",
		Errors:  []*api.Code{notFound, fieldRefused.Code},
	}, h.updateConnection)
	api.Register(o, api.Admin, api.Op{
		ID: "deleteNotificationConnection", Method: http.MethodDelete, Path: base + "/connections/{id}", Status: http.StatusNoContent,
		Summary: "Delete a notification connection, whose waiting messages aren't sent",
		Errors:  []*api.Code{notFound},
	}, h.deleteConnection)
	api.Register(o, api.Admin, api.Op{
		ID: "testNotificationConnection", Method: http.MethodPost, Path: base + "/connections/{id}/test", Status: http.StatusOK,
		Summary: "Send a Test message to a saved connection, with any unsaved edits applied. A passing Test of the saved values also sends the messages held while it was paused",
		Errors:  []*api.Code{notFound, fieldRefused.Code, testFailed.Code},
	}, h.testSaved)
	api.Register(o, api.Admin, api.Op{
		ID: "getNotificationBell", Method: http.MethodGet, Path: base + "/bell", Status: http.StatusOK,
		Summary: "Get the events the bell shows",
	}, h.getBell)
	api.Register(o, api.Admin, api.Op{
		ID: "updateNotificationBell", Method: http.MethodPatch, Path: base + "/bell", Status: http.StatusOK,
		Summary: "Change the events the bell shows from now on",
		Errors:  []*api.Code{unknownEvent},
	}, h.updateBell)
	api.Register(o, api.Admin, api.Op{
		ID: "listNotifications", Method: http.MethodGet, Path: base + "/feed", Status: http.StatusOK,
		Summary: "List what the bell showed, newest first, with how many rows are unread",
	}, h.listFeed)
	api.Register(o, api.Admin, api.Op{
		ID: "markNotificationsSeen", Method: http.MethodPost, Path: base + "/feed/seen", Status: http.StatusNoContent,
		Summary: "Mark the bell's rows up to through as read for the signed-in Admin",
		Errors:  []*api.Code{api.NeedsAccount},
	}, h.markSeen)
	api.Register(o, api.Admin, api.Op{
		ID: "getNotificationSettings", Method: http.MethodGet, Path: base + "/settings", Status: http.StatusOK,
		Summary: "Get the Notifications settings",
	}, h.getSettings)
	api.Register(o, api.Admin, api.Op{
		ID: "updateNotificationSettings", Method: http.MethodPatch, Path: base + "/settings", Status: http.StatusOK,
		Summary: "Change the Notifications settings by key, all or none",
		Errors:  []*api.Code{settingapi.ErrOutOfRange, settingapi.ErrUnknown},
	}, h.updateSettings)
}
