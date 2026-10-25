// Package healthapi is the Health page's operations: what is wrong, Check
// now, and the Storage levels.
package healthapi

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/health"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/settingapi"
	"github.com/jatassi/slipstream/internal/storage"
)

type Watch interface {
	// Health is the Health owner, false unless the Node is a Controller.
	Health() (*health.Health, bool)
}

func open(w Watch) *health.Health {
	h, ok := w.Health()
	if !ok {
		panic("healthapi: an operation ran on a Node that is no Controller")
	}
	return h
}

var (
	categoryNotFound = api.NewCode(http.StatusNotFound, "health_category_not_found", "No Health category has that id")
	itemNotFound     = api.NewCode(http.StatusNotFound, "health_item_not_found", "The category has no item with that key")
	noCheck          = api.NewCode(http.StatusNotFound, "no_check", "There is no Check now for that")
	volumeNotFound   = api.NewCode(http.StatusNotFound, "health_volume_not_found", "No volume reports at that Node and mount")
)

func answer(err error) error {
	switch {
	case errors.Is(err, health.ErrNoCategory):
		return categoryNotFound
	case errors.Is(err, health.ErrNoItem):
		return itemNotFound
	case errors.Is(err, health.ErrNoCheck):
		return noCheck
	case errors.Is(err, health.ErrNoVolume):
		return volumeNotFound
	}
	return settingapi.Refused(err)
}

// categoryID is the registered categories' ids, so the web's union of them
// follows health.IDs.
type categoryID struct{}

func (categoryID) Values() []string {
	ids := health.IDs()
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = string(id)
	}
	return out
}

type report struct {
	Level      *string               `json:"level" enum:"warn,fail" doc:"The worst level over every category, null while nothing is wrong"`
	First      api.Nullable[pointer] `json:"first" doc:"The item the status sentence leads with: the worst, then the oldest, then in page order; null while nothing is wrong"`
	Categories []category            `json:"categories" nullable:"false" doc:"In page order"`
}

type pointer struct {
	Category api.Enum[categoryID] `json:"category"`
	Key      string               `json:"key"`
	Text     string               `json:"text"`
}

type category struct {
	ID       api.Enum[categoryID] `json:"id"`
	Label    string               `json:"label"`
	Phrase   string               `json:"phrase" doc:"The label inside a sentence, such as storage"`
	Level    *string              `json:"level" enum:"warn,fail" doc:"Its worst item's level, null while it has none"`
	CanCheck bool                 `json:"canCheck" doc:"It has a Check now"`
	Unread   string               `json:"unread,omitempty" doc:"Why its last check failed. Its items then show their last known state. Absent while checks succeed"`
	Items    []item               `json:"items" nullable:"false" doc:"Errors first, then the oldest"`
}

type item struct {
	Key       string               `json:"key" doc:"Stable within its category and safe as one path segment"`
	Level     string               `json:"level" enum:"warn,fail"`
	Subject   string               `json:"subject"`
	Text      string               `json:"text" doc:"What to do first"`
	Detail    []fact               `json:"detail" nullable:"false"`
	Action    api.Nullable[action] `json:"action" doc:"Its one action, null when it has none"`
	Since     time.Time            `json:"since" doc:"When it first appeared at any level"`
	ChangedAt time.Time            `json:"changedAt" doc:"When its level last changed"`
	Unsure    bool                 `json:"unsure" doc:"Its owner can't judge it now, so it reads as it last did"`
}

type fact struct {
	Label string     `json:"label"`
	Text  string     `json:"text,omitempty" doc:"Exactly one of text, bytes and at is present"`
	Bytes *int64     `json:"bytes,omitempty"`
	At    *time.Time `json:"at,omitempty"`
}

type action struct {
	Kind   string `json:"kind" enum:"check,run,link" doc:"check: Check now on the item; run: Run now on taskId; link: a page in the web"`
	TaskID string `json:"taskId,omitempty"`
	Href   string `json:"href,omitempty"`
	Label  string `json:"label,omitempty" doc:"The link's words"`
}

func levelOf(l health.Level) *string {
	if l == health.None {
		return nil
	}
	return new(l.String())
}

func reportOf(r health.Report) report {
	out := report{Level: levelOf(r.Level), Categories: []category{}}
	if r.First != nil {
		out.First = api.Some(pointer{Category: api.NewEnum[categoryID](string(r.First.Category)), Key: string(r.First.Key), Text: r.First.Text})
	}
	for _, c := range r.Categories {
		cat := category{
			ID: api.NewEnum[categoryID](string(c.ID)), Label: c.Label, Phrase: c.Phrase, Level: levelOf(c.Level),
			CanCheck: c.CanCheck, Unread: c.Unread, Items: []item{},
		}
		for _, s := range c.Items {
			cat.Items = append(cat.Items, itemOf(s))
		}
		out.Categories = append(out.Categories, cat)
	}
	return out
}

func itemOf(s health.Shown) item {
	out := item{
		Key: string(s.Key), Level: s.Level.String(), Subject: s.Subject, Text: s.Text, Detail: []fact{},
		Since: s.Since, ChangedAt: s.ChangedAt, Unsure: s.Unsure,
	}
	for _, f := range s.Detail {
		wf := fact{Label: f.Label, Text: f.Text, Bytes: f.Bytes}
		if !f.At.IsZero() {
			wf.At = &f.At
		}
		out.Detail = append(out.Detail, wf)
	}
	if a := s.Action; a.Kind() != health.NoAction {
		out.Action = api.Some(action{Kind: a.Kind().String(), TaskID: a.Task(), Href: a.Href(), Label: a.Label()})
	}
	return out
}

type reportOutput struct{ Body report }

type unreached struct {
	Key   string `json:"key,omitempty" doc:"The item it was for; absent for a whole category"`
	Words string `json:"words" doc:"What couldn't be reached, as a sentence"`
}

type checked struct {
	Report    report      `json:"report" doc:"What is wrong right after the check"`
	Unreached []unreached `json:"unreached" nullable:"false" doc:"What the check couldn't reach; those items stay as they were"`
}

type checkedOutput struct{ Body checked }

func checkedOf(c health.Checked) *checkedOutput {
	out := checked{Report: reportOf(c.Report), Unreached: []unreached{}}
	for _, u := range c.Unreached {
		out.Unreached = append(out.Unreached, unreached{Key: string(u.Key), Words: u.Words})
	}
	return &checkedOutput{Body: out}
}

type categoryInput struct {
	Category string `path:"category"`
}

type itemInput struct {
	Category string `path:"category"`
	Key      string `path:"key"`
}

type settings struct {
	WarnBelow  setting.SpaceField `json:"warnBelow"`
	ErrorBelow setting.SpaceField `json:"errorBelow"`
}

type settingsOutput struct{ Body settings }

type healthSettingsInput struct {
	Body struct {
		WarnBelow  api.Patch[setting.Amount] `json:"warnBelow,omitempty" doc:"null resets it to its default"`
		ErrorBelow api.Patch[setting.Amount] `json:"errorBelow,omitempty" doc:"null resets it to its default"`
	}
}

type volume struct {
	NodeID       string             `json:"nodeId" format:"uuid"`
	NodeName     string             `json:"nodeName"`
	Mount        string             `json:"mount" doc:"As its Node names it, such as D:\\ or /mnt/media"`
	Name         string             `json:"name" doc:"Such as D: (Media)"`
	Total        int64              `json:"total"`
	Free         int64              `json:"free" doc:"The bytes SlipStream may write"`
	SampledAt    time.Time          `json:"sampledAt"`
	Live         bool               `json:"live" doc:"Its Node is in step, so its figures are current. Otherwise they are from sampledAt"`
	WarnBelow    setting.SpaceField `json:"warnBelow" doc:"Its own warning level, the default while it follows the global one"`
	ErrorBelow   setting.SpaceField `json:"errorBelow" doc:"Its own error level, the default while it follows the global one"`
	WarnBytes    int64              `json:"warnBytes" doc:"The warning level it follows, in bytes"`
	ErrorBytes   int64              `json:"errorBytes" doc:"The error level it follows, in bytes"`
	SkipsWarning bool               `json:"skipsWarning" doc:"Its error level is at or above its warning level, so it goes straight to error"`
	Level        *string            `json:"level" enum:"warn,fail" doc:"Its Storage item's level, null while it has none"`
}

func volumeOf(v health.ListedVolume) volume {
	return volume{
		NodeID: v.Node.String(), NodeName: v.NodeName, Mount: v.Mount, Name: v.Name, Total: v.Total, Free: v.Free,
		SampledAt: v.SampledAt, Live: v.Live, WarnBelow: v.WarnBelow, ErrorBelow: v.ErrorBelow,
		WarnBytes: v.Levels.Warn.Bytes, ErrorBytes: v.Levels.Error.Bytes, SkipsWarning: v.Levels.SkipsWarning(), Level: levelOf(v.Level),
	}
}

type notReporting struct {
	NodeID     string                       `json:"nodeId" format:"uuid"`
	NodeName   string                       `json:"nodeName"`
	Mount      string                       `json:"mount"`
	WarnBelow  api.Nullable[setting.Amount] `json:"warnBelow" doc:"Its own warning level, null while it follows the global one"`
	ErrorBelow api.Nullable[setting.Amount] `json:"errorBelow" doc:"Its own error level, null while it follows the global one"`
}

func nullable(a *setting.Amount) api.Nullable[setting.Amount] {
	if a == nil {
		return api.Nullable[setting.Amount]{}
	}
	return api.Some(*a)
}

type volumeList struct {
	Volumes      []volume       `json:"volumes" nullable:"false" doc:"Every volume a Node reports, by Node then mount"`
	NotReporting []notReporting `json:"notReporting" nullable:"false" doc:"Volumes with levels of their own that no Node reports now, by Node then mount"`
}

type volumeListOutput struct{ Body volumeList }

type volumeOutput struct{ Body volume }

type healthVolumeInput struct {
	Body struct {
		NodeID     string                    `json:"nodeId" format:"uuid"`
		Mount      string                    `json:"mount" minLength:"1"`
		WarnBelow  api.Patch[setting.Amount] `json:"warnBelow,omitempty" doc:"null makes it follow the global level"`
		ErrorBelow api.Patch[setting.Amount] `json:"errorBelow,omitempty" doc:"null makes it follow the global level"`
	}
}

type deleteVolumeInput struct {
	NodeID string `query:"nodeId" format:"uuid" required:"true"`
	Mount  string `query:"mount" minLength:"1" required:"true"`
}

func Routes(r *api.Router, w Watch) {
	o := r.Owner("health", api.Admin)
	h := handlers{w}

	api.Register(o, api.Admin, api.Op{
		ID:      "getSystemHealth",
		Method:  http.MethodGet,
		Path:    "/system/health",
		Summary: "Get what is wrong with SlipStream, by category",
		Status:  http.StatusOK,
	}, h.getSystemHealth)

	api.Register(o, api.Admin, api.Op{
		ID:      "checkSystemHealth",
		Method:  http.MethodPost,
		Path:    "/system/health/check",
		Summary: "Check every category now",
		Status:  http.StatusOK,
	}, h.checkSystemHealth)

	api.Register(o, api.Admin, api.Op{
		ID:      "checkHealthCategory",
		Method:  http.MethodPost,
		Path:    "/system/health/{category}/check",
		Summary: "Check one category now",
		Status:  http.StatusOK,
		Errors:  []*api.Code{categoryNotFound, noCheck},
	}, h.checkHealthCategory)

	api.Register(o, api.Admin, api.Op{
		ID:      "checkHealthItem",
		Method:  http.MethodPost,
		Path:    "/system/health/{category}/items/{key}/check",
		Summary: "Check one item now",
		Status:  http.StatusOK,
		Errors:  []*api.Code{categoryNotFound, itemNotFound, noCheck},
	}, h.checkHealthItem)

	api.Register(o, api.Admin, api.Op{
		ID:      "getHealthSettings",
		Method:  http.MethodGet,
		Path:    "/system/health/settings",
		Summary: "Get the global Storage levels",
		Status:  http.StatusOK,
	}, h.getHealthSettings)

	api.Register(o, api.Admin, api.Op{
		ID:      "updateHealthSettings",
		Method:  http.MethodPatch,
		Path:    "/system/health/settings",
		Summary: "Change the global Storage levels",
		Status:  http.StatusOK,
		Errors:  []*api.Code{settingapi.ErrOutOfRange},
	}, h.updateHealthSettings)

	api.Register(o, api.Admin, api.Op{
		ID:      "listHealthVolumes",
		Method:  http.MethodGet,
		Path:    "/system/health/volumes",
		Summary: "List every volume with the levels it follows",
		Status:  http.StatusOK,
	}, h.listHealthVolumes)

	api.Register(o, api.Admin, api.Op{
		ID:      "updateHealthVolume",
		Method:  http.MethodPatch,
		Path:    "/system/health/volumes",
		Summary: "Set or reset a volume's own levels",
		Status:  http.StatusOK,
		Errors:  []*api.Code{volumeNotFound, settingapi.ErrOutOfRange},
	}, h.updateHealthVolume)

	api.Register(o, api.Admin, api.Op{
		ID:      "deleteHealthVolume",
		Method:  http.MethodDelete,
		Path:    "/system/health/volumes",
		Summary: "Remove a volume's own levels",
		Status:  http.StatusNoContent,
	}, h.deleteHealthVolume)
}

type handlers struct{ w Watch }

func (h handlers) getSystemHealth(ctx context.Context, _ identity.Admin, _ *struct{}) (*reportOutput, error) {
	r, err := open(h.w).Report(ctx)
	if err != nil {
		return nil, err
	}
	return &reportOutput{Body: reportOf(r)}, nil
}

func (h handlers) checkSystemHealth(ctx context.Context, _ identity.Admin, _ *struct{}) (*checkedOutput, error) {
	c, err := open(h.w).CheckAll(ctx)
	if err != nil {
		return nil, err
	}
	return checkedOf(c), nil
}

func (h handlers) checkHealthCategory(ctx context.Context, _ identity.Admin, in *categoryInput) (*checkedOutput, error) {
	c, err := open(h.w).Check(ctx, health.ID(in.Category), "")
	if err != nil {
		return nil, answer(err)
	}
	return checkedOf(c), nil
}

func (h handlers) checkHealthItem(ctx context.Context, _ identity.Admin, in *itemInput) (*checkedOutput, error) {
	c, err := open(h.w).Check(ctx, health.ID(in.Category), health.Key(in.Key))
	if err != nil {
		return nil, answer(err)
	}
	return checkedOf(c), nil
}

func (h handlers) getHealthSettings(_ context.Context, _ identity.Admin, _ *struct{}) (*settingsOutput, error) {
	return currentSettings(open(h.w)), nil
}

func (h handlers) updateHealthSettings(ctx context.Context, _ identity.Admin, in *healthSettingsInput) (*settingsOutput, error) {
	hl := open(h.w)
	p, err := pairPatch(in.Body.WarnBelow, in.Body.ErrorBelow)
	if err != nil {
		return nil, err
	}
	if err := hl.SetGlobalLevels(ctx, p); err != nil {
		return nil, answer(err)
	}
	return currentSettings(hl), nil
}

func currentSettings(hl *health.Health) *settingsOutput {
	warn, errorBelow := hl.GlobalLevels()
	return &settingsOutput{Body: settings{WarnBelow: warn, ErrorBelow: errorBelow}}
}

func pairPatch(warn, errorBelow api.Patch[setting.Amount]) (health.PairPatch, error) {
	var p health.PairPatch
	var err error
	if p.Warn, err = parsed(health.StorageWarnBelow, warn); err != nil {
		return health.PairPatch{}, err
	}
	if p.Error, err = parsed(health.StorageErrorBelow, errorBelow); err != nil {
		return health.PairPatch{}, err
	}
	return p, nil
}

// parsed is the level p sends, nil when it sends none.
func parsed(s setting.Space, p api.Patch[setting.Amount]) (*setting.Stored[setting.Amount], error) {
	if !p.Sent() {
		return nil, nil
	}
	v, err := s.Parse(p.Value())
	if err != nil {
		return nil, settingapi.Refused(err)
	}
	return &v, nil
}

func (h handlers) listHealthVolumes(ctx context.Context, _ identity.Admin, _ *struct{}) (*volumeListOutput, error) {
	list, err := open(h.w).VolumeList(ctx)
	if err != nil {
		return nil, err
	}
	out := volumeList{Volumes: []volume{}, NotReporting: []notReporting{}}
	for _, v := range list.Volumes {
		out.Volumes = append(out.Volumes, volumeOf(v))
	}
	for _, o := range list.NotReporting {
		out.NotReporting = append(out.NotReporting, notReporting{
			NodeID: o.Node.String(), NodeName: o.NodeName, Mount: o.Mount, WarnBelow: nullable(o.Own.Warn), ErrorBelow: nullable(o.Own.Error),
		})
	}
	return &volumeListOutput{Body: out}, nil
}

func (h handlers) updateHealthVolume(ctx context.Context, _ identity.Admin, in *healthVolumeInput) (*volumeOutput, error) {
	node, err := nodeID(in.Body.NodeID)
	if err != nil {
		return nil, volumeNotFound
	}
	p, err := pairPatch(in.Body.WarnBelow, in.Body.ErrorBelow)
	if err != nil {
		return nil, err
	}
	hl := open(h.w)
	if err := hl.SetVolumeLevels(ctx, node, in.Body.Mount, p); err != nil {
		return nil, answer(err)
	}
	list, err := hl.VolumeList(ctx)
	if err != nil {
		return nil, err
	}
	for _, v := range list.Volumes {
		if v.Node == node && v.Mount == in.Body.Mount {
			return &volumeOutput{Body: volumeOf(v)}, nil
		}
	}
	return nil, volumeNotFound
}

func (h handlers) deleteHealthVolume(ctx context.Context, _ identity.Admin, in *deleteVolumeInput) (*struct{}, error) {
	node, err := nodeID(in.NodeID)
	if err != nil {
		return nil, err
	}
	return nil, open(h.w).RemoveVolumeLevels(ctx, node, in.Mount)
}

func nodeID(s string) (role.NodeID, error) {
	u, err := storage.ParseUUID(s)
	if err != nil {
		return role.NodeID{}, err
	}
	return role.NodeID{UUID: u}, nil
}
