// Package importsapi is Pending imports and the Library scan settings over
// the API.
package importsapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/imports"
	"github.com/jatassi/slipstream/internal/metadata/artwork"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/scanning"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/settingapi"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

type Pending interface {
	// Imports is Pending imports, false unless the Node is a Controller.
	Imports() (*imports.Imports, bool)
	// Roots is the library roots, false unless the Node is a Controller.
	Roots() (*roots.Roots, bool)
	// Minter signs the artwork URLs the answers carry, false unless the
	// Node is a Controller.
	Minter() (artwork.Minter, bool)
}

func open(p Pending) *imports.Imports {
	im, ok := p.Imports()
	if !ok {
		panic("importsapi: an operation ran on a Node that is no Controller")
	}
	return im
}

func minter(p Pending) artwork.Minter {
	m, ok := p.Minter()
	if !ok {
		panic("importsapi: an operation ran on a Node that is no Controller")
	}
	return m
}

var (
	pendingNotFound     = api.NewCode(http.StatusNotFound, "pending_not_found", "No pending import has that id")
	pendingNoSuggestion = api.NewCode(http.StatusConflict, "pending_no_suggestion", "The pending import has no suggested Work")
	pendingElsewhere    = api.NewCode(http.StatusConflict, "pending_elsewhere", "The Work lives in another library root")
	pendingChanged      = api.NewCode(http.StatusConflict, "pending_changed", "The pending import no longer lists a chosen file")
	pendingItems        = api.NewCode(http.StatusUnprocessableEntity, "pending_items", "A chosen Item isn't in the Work")
	contentTypeDisabled = api.NewCode(http.StatusConflict, "content_type_disabled", "That Content Type is turned off")
	metadataNotFound    = api.NewCode(http.StatusNotFound, "metadata_not_found", "The metadata source has no record with that key")
	metadataUnavailable = api.NewCode(http.StatusBadGateway, "metadata_unavailable", "The metadata source didn't respond. Try again later.")
	metadataInvalid     = api.NewCode(http.StatusBadGateway, "metadata_invalid", "The metadata source sent a record SlipStream can't use")
	metadataRefused     = api.NewCode(http.StatusBadGateway, "metadata_refused", "The metadata source refused SlipStream's key")
	notOnSource         = api.NewCode(http.StatusUnprocessableEntity, "not_on_source", "The metadata source doesn't list this record")
)

var codes = []struct {
	err  error
	code *api.Code
}{
	{imports.ErrNoEntry, pendingNotFound},
	{imports.ErrNoSuggestion, pendingNoSuggestion},
	{imports.ErrElsewhere, pendingElsewhere},
	{imports.ErrChanged, pendingChanged},
	{imports.ErrNotOfWork, pendingItems},
	{works.ErrTypeOff, contentTypeDisabled},
	{works.ErrNotFound, metadataNotFound},
	{works.ErrUnavailable, metadataUnavailable},
	{works.ErrInvalid, metadataInvalid},
	{works.ErrRefused, metadataRefused},
	{works.ErrNotOnSource, notOnSource},
}

func answer(err error) error {
	for _, c := range codes {
		if errors.Is(err, c.err) {
			return c.code
		}
	}
	return err
}

type rootLabel struct {
	ID    string `json:"id" format:"uuid"`
	Label string `json:"label" doc:"The root as copy names it, such as Movies library root on Media server"`
}

type lead struct {
	Root        rootLabel `json:"root"`
	ContentType string    `json:"contentType" pattern:"^[a-z][a-z0-9]{1,15}$"`
	Added       int       `json:"added" minimum:"0" doc:"The Works the root's last library scan added"`
	NeedsYou    int       `json:"needsYou" minimum:"0" doc:"The root's pending imports that aren't ignored"`
}

type hint struct {
	Title string   `json:"title" doc:"The title the folder or file name carries, as written"`
	Year  *int     `json:"year" doc:"Null when the name carries none"`
	IDs   []hintID `json:"ids" nullable:"false" doc:"The provider ids the name's tags carry"`
}

type hintID struct {
	Provider string `json:"provider" enum:"tmdb,imdb,tvdb"`
	Value    string `json:"value"`
}

type suggestion struct {
	Key       string  `json:"key" doc:"The record's key at its Content Type's source"`
	Label     string  `json:"label"`
	Year      *int    `json:"year" doc:"Null when the record has none"`
	PosterURL *string `json:"posterUrl" doc:"The artwork route for its poster, without a size; null when it has none"`
	WorkID    *string `json:"workId" format:"uuid" doc:"The Work, once it is in the library"`
	Confident bool    `json:"confident" doc:"It meets the rules that let a library scan add a Work by itself"`
}

type reason struct {
	Code  string     `json:"code" enum:"several,no_match,no_year,elsewhere,unlisted,no_number,no_such_item,several_items,unreadable,left,no_rule" doc:"Why the file joins no Work"`
	Title string     `json:"title,omitempty" doc:"For several, no_match, no_year, elsewhere and unlisted: the name's title or the Work's label"`
	Year  int        `json:"year,omitempty" doc:"The year beside title, when it has one"`
	Count int        `json:"count,omitempty" doc:"For several: the candidates that match"`
	Names string     `json:"names,omitempty" doc:"For no_such_item: the Item the file name names, such as S03E14"`
	Root  *rootLabel `json:"root,omitempty" doc:"For elsewhere: the root the Work lives in"`
	Line  string     `json:"line,omitempty" doc:"For unreadable: the first line ffprobe printed"`
}

type pendingFile struct {
	Path    string   `json:"path" doc:"Relative to the root, / separated"`
	Size    int64    `json:"size" minimum:"0"`
	Quality *string  `json:"quality" enum:"sdtv,dvd,webrip-480p,hdtv-720p,webrip-720p,webdl-720p,bluray-720p,hdtv-1080p,webrip-1080p,webdl-1080p,bluray-1080p,remux-1080p,hdtv-2160p,webrip-2160p,webdl-2160p,bluray-2160p,remux-2160p,cam" doc:"The quality its name and media info read as; null when ffprobe can't read it"`
	Cut     string   `json:"cut" doc:"The key of the cut its name carries"`
	ItemIDs []string `json:"itemIds" nullable:"false" doc:"The Items of the suggested Work its path names; empty until a Work places it"`
	Reason  reason   `json:"reason"`
}

type entry struct {
	ID          string                   `json:"id" format:"uuid"`
	Root        rootLabel                `json:"root"`
	Folder      string                   `json:"folder" doc:"Its Work folder relative to the root, or the file's path when it sits loose at the root's top"`
	ContentType string                   `json:"contentType" pattern:"^[a-z][a-z0-9]{1,15}$"`
	Hint        hint                     `json:"hint" doc:"What the folder or file name says"`
	Suggestion  api.Nullable[suggestion] `json:"suggestion" doc:"The best candidate; null when none matched"`
	FoundAt     time.Time                `json:"foundAt"`
	IgnoredAt   *time.Time               `json:"ignoredAt" doc:"Null while it needs the Admin"`
	Files       []pendingFile            `json:"files" nullable:"false" doc:"In path order"`
}

type converter struct {
	m     artwork.Minter
	roots map[roots.ID]string
}

func (c converter) label(id roots.ID) rootLabel {
	return rootLabel{ID: id.String(), Label: c.roots[id]}
}

func (c converter) entry(e imports.Entry) entry {
	out := entry{
		ID: e.ID.String(), Root: c.label(e.Root), Folder: e.Folder, ContentType: string(e.ContentType),
		Hint: hintOf(e.Hint), FoundAt: e.FoundAt.Time(), Files: make([]pendingFile, len(e.Files)),
	}
	if e.Suggestion != nil {
		out.Suggestion = api.Some(c.suggestion(*e.Suggestion))
	}
	if e.Ignored() {
		out.IgnoredAt = new(e.IgnoredAt.Time())
	}
	for i, f := range e.Files {
		out.Files[i] = c.file(f)
	}
	return out
}

func hintOf(h scanning.Hint) hint {
	out := hint{Title: h.Title, Year: yearOrNull(h.Year), IDs: make([]hintID, len(h.IDs))}
	for i, id := range h.IDs {
		out.IDs[i] = hintID{Provider: string(id.Provider), Value: id.Value}
	}
	return out
}

func yearOrNull(y int) *int {
	if y == 0 {
		return nil
	}
	return &y
}

func (c converter) suggestion(s imports.Suggestion) suggestion {
	out := suggestion{Key: string(s.Key), Label: s.Label, Year: yearOrNull(s.Year), Confident: s.Confident}
	if !s.Poster.IsZero() {
		out.PosterURL = new(c.m.URL(s.Poster))
	}
	if !s.WorkID.IsZero() {
		out.WorkID = new(s.WorkID.String())
	}
	return out
}

func (c converter) file(f imports.PendingFile) pendingFile {
	out := pendingFile{Path: f.Path, Size: f.Size, Cut: string(f.Cut), ItemIDs: make([]string, len(f.Items))}
	if f.Media != nil {
		out.Quality = new(f.Quality.String())
	}
	for i, it := range f.Items {
		out.ItemIDs[i] = it.String()
	}
	r := f.Reason
	out.Reason = reason{Code: string(r.Code), Title: r.Title, Year: r.Year, Count: r.Count, Names: r.Names, Line: r.Line}
	if id, err := storage.ParseUUID(r.Root); err == nil {
		out.Reason.Root = new(c.label(roots.ID{UUID: id}))
	}
	return out
}

type pendingListInput struct {
	RootID  string `query:"rootId" format:"uuid" doc:"Only this root's pending imports and lead; every root's when left out"`
	Ignored bool   `query:"ignored" doc:"List the ignored pending imports instead of the ones that need the Admin"`
}

type list struct {
	Leads   []lead  `json:"leads" nullable:"false" doc:"One per root a library scan compared, in root order"`
	Entries []entry `json:"entries" nullable:"false" doc:"In root and folder order"`
}

type listOutput struct{ Body list }

type pendingIDsInput struct {
	Body struct {
		IDs []string `json:"ids" format:"uuid" minItems:"1" nullable:"false"`
	}
}

type matchPendingInput struct {
	ID   string `path:"id" format:"uuid"`
	Body struct {
		Key string `json:"key" minLength:"1" doc:"A record's key at the Content Type's source, as searchCatalog lists it"`
	}
}

type entryOutput struct{ Body entry }

type chosenFile struct {
	Path    string   `json:"path" minLength:"1" doc:"One of the pending import's paths"`
	ItemIDs []string `json:"itemIds" minItems:"1" maxItems:"1" doc:"The Item of the Work the file holds. One for now; S33 (Import pipeline) allows several"`
}

type importIntoInput struct {
	ID   string `path:"id" format:"uuid"`
	Body struct {
		WorkID string       `json:"workId" format:"uuid" doc:"A Work in the pending import's root"`
		Files  []chosenFile `json:"files" minItems:"1" nullable:"false"`
	}
}

type imported struct {
	Entry api.Nullable[entry] `json:"entry" doc:"The files left waiting; null once none is"`
}

type importedOutput struct{ Body imported }

type outcome struct{}

func (outcome) Variants() (discriminator string, variants []any) {
	return "outcome", []any{addedOutcome{}, failedOutcome{}}
}

type addedOutcome struct {
	ID      string `json:"id" format:"uuid"`
	Outcome string `json:"outcome" enum:"added"`
	WorkID  string `json:"workId" format:"uuid" doc:"The Work its files joined"`
}

type failedOutcome struct {
	ID      string         `json:"id" format:"uuid"`
	Outcome string         `json:"outcome" enum:"failed"`
	Problem outcomeProblem `json:"problem"`
}

type outcomeProblem struct {
	Code   string `json:"code" doc:"The code a request for this pending import alone would have been refused with"`
	Title  string `json:"title"`
	Detail string `json:"detail,omitempty"`
}

type addedList struct {
	Outcomes []api.OneOf[outcome] `json:"outcomes" nullable:"false" doc:"One per pending import, in request order"`
}

type addedOutput struct{ Body addedList }

type settings struct {
	Settings []api.OneOf[settingapi.Field] `json:"settings" nullable:"false" doc:"The stop share, then the files each Node probes at once"`
}

type settingsOutput struct{ Body settings }

type libraryScanSettingsInput struct {
	Body struct {
		Values map[string]json.RawMessage `json:"values" doc:"Each setting's new value by key, all or none; null resets one to its default"`
	}
}

func Routes(r *api.Router, p Pending) {
	o := r.Owner("imports", api.Admin)
	h := handlers{p}

	api.Register(o, api.Admin, api.Op{
		ID:      "listPendingImports",
		Method:  http.MethodGet,
		Path:    "/imports/pending",
		Summary: "List the pending imports, with what each root's last library scan added",
		Status:  http.StatusOK,
	}, h.list)

	api.Register(o, api.Admin, api.Op{
		ID:      "addPendingImports",
		Method:  http.MethodPost,
		Path:    "/imports/pending/add",
		Summary: "Add each pending import's suggested Work to its root and attach its files",
		Status:  http.StatusOK,
	}, h.add)

	api.Register(o, api.Admin, api.Op{
		ID:      "matchPendingImport",
		Method:  http.MethodPost,
		Path:    "/imports/pending/{id}/match",
		Summary: "Make a record at the Content Type's source a pending import's suggestion",
		Status:  http.StatusOK,
		Errors:  []*api.Code{pendingNotFound, contentTypeDisabled, metadataNotFound, metadataUnavailable, metadataInvalid, metadataRefused, notOnSource},
	}, h.match)

	api.Register(o, api.Admin, api.Op{
		ID:      "importPendingInto",
		Method:  http.MethodPost,
		Path:    "/imports/pending/{id}/import",
		Summary: "Attach a pending import's chosen files to a Work in its root, at the Items chosen",
		Status:  http.StatusOK,
		Errors:  []*api.Code{pendingNotFound, pendingElsewhere, pendingChanged, pendingItems},
	}, h.importInto)

	api.Register(o, api.Admin, api.Op{
		ID:      "ignorePendingImports",
		Method:  http.MethodPost,
		Path:    "/imports/pending/ignore",
		Summary: "Ignore pending imports until their files change",
		Status:  http.StatusNoContent,
	}, h.ignore)

	api.Register(o, api.Admin, api.Op{
		ID:      "unignorePendingImports",
		Method:  http.MethodPost,
		Path:    "/imports/pending/unignore",
		Summary: "Bring ignored pending imports back to the ones that need the Admin",
		Status:  http.StatusNoContent,
	}, h.unignore)

	api.Register(o, api.Admin, api.Op{
		ID:      "getLibraryScanSettings",
		Method:  http.MethodGet,
		Path:    "/imports/settings",
		Summary: "Get the Library scan settings",
		Status:  http.StatusOK,
	}, h.getSettings)

	api.Register(o, api.Admin, api.Op{
		ID:      "updateLibraryScanSettings",
		Method:  http.MethodPatch,
		Path:    "/imports/settings",
		Summary: "Change the Library scan settings",
		Status:  http.StatusOK,
		Errors:  []*api.Code{settingapi.ErrOutOfRange, settingapi.ErrUnknown},
	}, h.updateSettings)
}

type handlers struct{ pending Pending }

func (h handlers) list(ctx context.Context, by identity.Admin, in *pendingListInput) (*listOutput, error) {
	f := imports.Filter{Ignored: in.Ignored}
	if in.RootID != "" {
		id, err := storage.ParseUUID(in.RootID)
		if err != nil {
			return nil, err
		}
		f.Root = &roots.ID{UUID: id}
	}
	l, err := open(h.pending).Pending(ctx, by, f)
	if err != nil {
		return nil, err
	}
	c := converter{m: minter(h.pending), roots: l.Roots}
	out := list{Leads: make([]lead, len(l.Leads)), Entries: make([]entry, len(l.Entries))}
	for i, ld := range l.Leads {
		out.Leads[i] = lead{Root: c.label(ld.Root), ContentType: string(ld.ContentType), Added: ld.Added, NeedsYou: ld.NeedsYou}
	}
	for i, e := range l.Entries {
		out.Entries[i] = c.entry(e)
	}
	return &listOutput{Body: out}, nil
}

func (h handlers) add(ctx context.Context, by identity.Admin, in *pendingIDsInput) (*addedOutput, error) {
	ids, err := entryIDs(in.Body.IDs)
	if err != nil {
		return nil, err
	}
	os, err := open(h.pending).Add(ctx, by, ids)
	if err != nil {
		return nil, err
	}
	out := addedList{Outcomes: make([]api.OneOf[outcome], len(os))}
	for i, o := range os {
		id := o.ID.String()
		if o.Err == nil {
			out.Outcomes[i] = api.NewOneOf[outcome](addedOutcome{ID: id, Outcome: "added", WorkID: o.Work.String()})
			continue
		}
		p, ok := api.ProblemOf(answer(o.Err))
		if !ok {
			return nil, o.Err
		}
		out.Outcomes[i] = api.NewOneOf[outcome](failedOutcome{ID: id, Outcome: "failed", Problem: outcomeProblem{Code: p.Code, Title: p.Title, Detail: p.Detail}})
	}
	return &addedOutput{Body: out}, nil
}

func (h handlers) match(ctx context.Context, by identity.Admin, in *matchPendingInput) (*entryOutput, error) {
	id, err := entryID(in.ID)
	if err != nil {
		return nil, err
	}
	e, err := open(h.pending).Match(ctx, by, id, works.Key(in.Body.Key))
	if err != nil {
		return nil, answer(err)
	}
	c, err := h.converter(ctx)
	if err != nil {
		return nil, err
	}
	return &entryOutput{Body: c.entry(e)}, nil
}

func (h handlers) importInto(ctx context.Context, by identity.Admin, in *importIntoInput) (*importedOutput, error) {
	id, err := entryID(in.ID)
	if err != nil {
		return nil, err
	}
	work, err := storage.ParseUUID(in.Body.WorkID)
	if err != nil {
		return nil, pendingItems
	}
	chosen := make([]imports.Chosen, len(in.Body.Files))
	for i, f := range in.Body.Files {
		item, err := storage.ParseUUID(f.ItemIDs[0])
		if err != nil {
			return nil, pendingItems
		}
		chosen[i] = imports.Chosen{Path: f.Path, Item: works.ItemID{UUID: item}}
	}
	left, err := open(h.pending).Into(ctx, by, id, works.WorkID{UUID: work}, chosen)
	if err != nil {
		return nil, answer(err)
	}
	var out imported
	if left != nil {
		c, err := h.converter(ctx)
		if err != nil {
			return nil, err
		}
		out.Entry = api.Some(c.entry(*left))
	}
	return &importedOutput{Body: out}, nil
}

func (h handlers) converter(ctx context.Context) (converter, error) {
	r, ok := h.pending.Roots()
	if !ok {
		panic("importsapi: an operation ran on a Node that is no Controller")
	}
	words, err := r.Words(ctx)
	if err != nil {
		return converter{}, err
	}
	return converter{m: minter(h.pending), roots: words}, nil
}

func (h handlers) ignore(ctx context.Context, by identity.Admin, in *pendingIDsInput) (*struct{}, error) {
	ids, err := entryIDs(in.Body.IDs)
	if err != nil {
		return nil, err
	}
	return nil, open(h.pending).Ignore(ctx, by, ids)
}

func (h handlers) unignore(ctx context.Context, by identity.Admin, in *pendingIDsInput) (*struct{}, error) {
	ids, err := entryIDs(in.Body.IDs)
	if err != nil {
		return nil, err
	}
	return nil, open(h.pending).Unignore(ctx, by, ids)
}

func (h handlers) getSettings(ctx context.Context, _ identity.Admin, _ *struct{}) (*settingsOutput, error) {
	return settingsOf(ctx, open(h.pending).Settings())
}

func (h handlers) updateSettings(ctx context.Context, _ identity.Admin, in *libraryScanSettingsInput) (*settingsOutput, error) {
	p := open(h.pending).Settings()
	if err := p.Apply(ctx, in.Body.Values); err != nil {
		return nil, settingapi.Refused(err)
	}
	return settingsOf(ctx, p)
}

func settingsOf(ctx context.Context, p setting.Panel) (*settingsOutput, error) {
	fields, err := p.Fields(ctx)
	if err != nil {
		return nil, err
	}
	return &settingsOutput{Body: settings{Settings: settingapi.FieldsOf(fields)}}, nil
}

func entryID(s string) (imports.EntryID, error) {
	u, err := storage.ParseUUID(s)
	if err != nil {
		return imports.EntryID{}, pendingNotFound
	}
	return imports.EntryID{UUID: u}, nil
}

func entryIDs(ss []string) ([]imports.EntryID, error) {
	out := make([]imports.EntryID, len(ss))
	for i, s := range ss {
		u, err := storage.ParseUUID(s)
		if err != nil {
			return nil, err
		}
		out[i] = imports.EntryID{UUID: u}
	}
	return out, nil
}
