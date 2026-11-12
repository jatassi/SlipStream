// Package importsapi is Imports, Pending imports' actions and the Library
// scan settings over the API.
package importsapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"strings"
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
	importNotFound      = api.NewCode(http.StatusNotFound, "import_not_found", "No Import has that id")
	importNoSuggestion  = api.NewCode(http.StatusConflict, "import_no_suggestion", "The Import has no suggested Work")
	importElsewhere     = api.NewCode(http.StatusConflict, "import_elsewhere", "The Work lives in another library root")
	importChanged       = api.NewCode(http.StatusConflict, "import_changed", "The Import no longer has a chosen file waiting")
	importItems         = api.NewCode(http.StatusUnprocessableEntity, "import_items", "A chosen Item isn't in the Work")
	importNotOffered    = api.NewCode(http.StatusConflict, "import_not_offered", "The file doesn't offer that action now. Get the Import again to see what it offers.")
	importListMoved     = api.NewCode(http.StatusConflict, "import_list_moved", "The Import the last page ended at is no longer listed. List again from the first page.")
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
	{imports.ErrNoImport, importNotFound},
	{imports.ErrNoSuggestion, importNoSuggestion},
	{imports.ErrElsewhere, importElsewhere},
	{imports.ErrChanged, importChanged},
	{imports.ErrNotOfWork, importItems},
	{imports.ErrListMoved, importListMoved},
	{imports.ErrOutside, importOutside},
	{imports.ErrNotOffered, importNotOffered},
	{imports.ErrMoveSeeding, importMoveSeeding},
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
	NeedsYou    int       `json:"needsYou" minimum:"0" doc:"The root's unfinished root Imports that aren't ignored"`
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
	Code     string               `json:"code" enum:"sample,bonus,hidden_folder,too_small,extension,runtime_sample,sibling_better,kept_better,meets_nothing,not_an_upgrade,edition_not_wanted,other_items,no_number,parts,name_taken,name_too_long,settings,busy,confirm,several,no_match,no_year,elsewhere,unlisted,no_such_item,unreadable,left,no_rule,import_failed,root_changed,by_admin,removed,canceled,blocklisted,work_deleted" doc:"Why the file is left out, waits, isn't better or failed"`
	Sentence api.Nullable[string] `json:"sentence" doc:"The one sentence the row leads with; null for a root Import's reason, which its other fields word"`
	Title    string               `json:"title,omitempty" doc:"For several, no_match, no_year, elsewhere and unlisted: the name's title or the Work's label"`
	Year     int                  `json:"year,omitempty" doc:"The year beside title, when it has one"`
	Count    int                  `json:"count,omitempty" doc:"For several: the candidates that match"`
	Names    string               `json:"names,omitempty" doc:"For no_such_item: the Item the file name names, such as S03E14"`
	Root     *rootLabel           `json:"root,omitempty" doc:"For elsewhere: the root the Work lives in"`
	Line     string               `json:"line,omitempty" doc:"For unreadable: the first line ffprobe printed"`
}

type importFile struct {
	Position    int                            `json:"position" minimum:"0" doc:"The file's place in its Import, which names it in actions"`
	Path        string                         `json:"path" doc:"For a root Import relative to the root, / separated"`
	Size        int64                          `json:"size" minimum:"0"`
	Quality     *string                        `json:"quality" enum:"sdtv,dvd,webrip-480p,hdtv-720p,webrip-720p,webdl-720p,bluray-720p,hdtv-1080p,webrip-1080p,webdl-1080p,bluray-1080p,remux-1080p,hdtv-2160p,webrip-2160p,webdl-2160p,bluray-2160p,remux-2160p,cam" doc:"The quality its name and media info read as; null when ffprobe can't read it"`
	Cut         string                         `json:"cut" doc:"The key of the cut its name carries"`
	ItemIDs     []string                       `json:"itemIds" nullable:"false" doc:"The Items of the Work it holds, in tree order; empty until a Work places it"`
	Outcome     string                         `json:"outcome" enum:"pending,left_out,not_better,waiting,sending,placing,placed,declined,failed"`
	Reason      api.Nullable[reason]           `json:"reason" doc:"Why it is left out, waits, isn't better or failed; null otherwise"`
	Hold        api.Nullable[fileHold]         `json:"hold" doc:"What it waits on that SlipStream retries automatically; null while nothing holds it"`
	Fills       []cellRef                      `json:"fills" nullable:"false" doc:"The cells it fills, or would fill once confirmed"`
	Verdicts    []targetVerdict                `json:"verdicts" nullable:"false" doc:"Each Target's verdict on it, for its Details"`
	Recommended api.Nullable[api.Enum[action]] `json:"recommended" doc:"The action its row offers first; null when it offers none"`
	Actions     []api.Enum[action]             `json:"actions" nullable:"false" doc:"The other actions its row offers"`
	Forces      []forceOption                  `json:"forces" nullable:"false" doc:"Each Target it can be forced into, with the files that deletes"`
	PlacedBy    api.Nullable[string]           `json:"placedBy" enum:"link,clone,copy,transfer,move,in_place" doc:"How it went into the library; null until it is placing"`
	FileID      api.Nullable[string]           `json:"fileId" format:"uuid" doc:"The library file it became; null until it is placed"`
	Staged      int64                          `json:"staged" minimum:"0" doc:"Bytes its holding or staged copy uses while it waits"`
	Movable     bool                           `json:"movable" doc:"Move files may take it: no tracked torrent seeds it"`
}

type action struct{}

func (action) Values() []string {
	return []string{"confirm", "retry", "decline", "force", "match", "notThisRelease", "changeEditions", "turnOnRenaming", "changeSetting"}
}

type fileHold struct {
	Code     string `json:"code" enum:"no_space,root_not_ready,node_away,probe_timeout,probe_error"`
	Sentence string `json:"sentence" doc:"What it waits on and when SlipStream retries"`
	Attempts int    `json:"attempts" minimum:"0" doc:"The tries that ran into the hold"`
}

type cellRef struct {
	ItemID   string `json:"itemId" format:"uuid"`
	Edition  string `json:"edition" doc:"any under any cut, else the key of the cut the cell wants"`
	TargetID string `json:"targetId" format:"uuid"`
}

type targetVerdict struct {
	TargetID string `json:"targetId" format:"uuid"`
	Sentence string `json:"sentence" doc:"How the Target judges the file, such as Fills 2160p for Heat (1995)."`
}

type forceOption struct {
	Cell    cellRef  `json:"cell"`
	Deletes []string `json:"deletes" nullable:"false" doc:"The paths of the library files the force deletes, relative to their root"`
}

type importEntry struct {
	ID          string                      `json:"id" format:"uuid"`
	Source      string                      `json:"source" enum:"grab,root,folder" doc:"root: a folder a library scan found in a root and couldn't place"`
	Stage       string                      `json:"stage" enum:"probing,waiting,placing,done,failed"`
	Grab        api.Nullable[importGrab]    `json:"grab" doc:"The Grab whose download this is; null for a folder or a root"`
	NodeID      *string                     `json:"nodeId" format:"uuid" doc:"The Node the files are on, for a download or a folder; null for a root"`
	Root        api.Nullable[rootLabel]     `json:"root" doc:"The library root the folder is in; null outside a root Import"`
	Folder      string                      `json:"folder" doc:"For a root Import, its Work folder relative to the root, or the file's path when it sits loose at the root's top. The folder on its Node for a folder Import, and empty for a download, whose file paths are relative to its torrent"`
	WorkID      *string                     `json:"workId" format:"uuid" doc:"The Work its files go into; null for a root Import, whose suggestion names one"`
	Edition     *string                     `json:"edition" doc:"The key of the cut its files fill; null for any under any cut"`
	MoveFiles   bool                        `json:"moveFiles" doc:"A folder Import moves its files into the library and deletes the folder once it is empty"`
	Seeding     bool                        `json:"seeding" doc:"The download keeps seeding while a file waits"`
	Failure     api.Nullable[importFailure] `json:"failure" doc:"Why the whole Import failed; null unless it did"`
	ContentType string                      `json:"contentType" pattern:"^[a-z][a-z0-9]{1,15}$"`
	Hint        hint                        `json:"hint" doc:"What the folder or file name says"`
	Suggestion  api.Nullable[suggestion]    `json:"suggestion" doc:"The best candidate; null when none matched"`
	CreatedAt   time.Time                   `json:"createdAt"`
	IgnoredAt   *time.Time                  `json:"ignoredAt" doc:"Null while it needs the Admin"`
	FinishedAt  *time.Time                  `json:"finishedAt" doc:"Null until every file is placed or settled"`
	Files       []importFile                `json:"files" nullable:"false" doc:"For a root Import in path order, else by position"`
}

type importGrab struct {
	ID      string `json:"id" format:"uuid"`
	Release string `json:"release" doc:"The Release's title"`
	Stage   string `json:"stage" enum:"fetching,downloading,probing,transferring,importing,imported,failed"`
}

type importFailure struct {
	Code     string `json:"code" enum:"no_video_file,unreadable"`
	Sentence string `json:"sentence"`
}

type converter struct {
	m         artwork.Minter
	roots     map[roots.ID]string
	sentences map[imports.ImportID]map[int]string
}

func (c converter) label(id roots.ID) rootLabel {
	return rootLabel{ID: id.String(), Label: c.roots[id]}
}

func (c converter) entry(e imports.Import) importEntry {
	out := importEntry{
		ID: e.ID.String(), Source: string(e.Source), Stage: string(e.Stage()), Folder: e.Folder, ContentType: string(e.ContentType),
		Hint: hintOf(e.Hint), CreatedAt: e.Created.Time(), Files: make([]importFile, len(e.Files)),
	}
	if !e.Root.IsZero() {
		out.Root = api.Some(c.label(e.Root))
	}
	if !e.Node.IsZero() {
		out.NodeID = new(e.Node.String())
	}
	if !e.Work.IsZero() {
		out.WorkID = new(e.Work.String())
	}
	if e.Edition != nil {
		out.Edition = new(string(*e.Edition))
	}
	out.MoveFiles = e.MoveFiles
	if e.Suggestion != nil {
		out.Suggestion = api.Some(c.suggestion(*e.Suggestion))
	}
	if e.IsIgnored() {
		out.IgnoredAt = new(e.Ignored.Time())
	}
	if !e.Finished.IsZero() {
		out.FinishedAt = new(e.Finished.Time())
	}
	for i, f := range e.Files {
		out.Files[i] = c.file(e, f)
	}
	if e.Source == imports.FromRoot {
		slices.SortFunc(out.Files, func(a, b importFile) int { return strings.Compare(a.Path, b.Path) })
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

func (c converter) file(e imports.Import, f imports.File) importFile {
	out := importFile{
		Position: f.Position, Path: f.Path, Size: f.Size, Cut: string(f.Cut), Outcome: string(f.Outcome), ItemIDs: make([]string, len(f.Items)),
		Fills: []cellRef{}, Verdicts: []targetVerdict{}, Actions: []api.Enum[action]{}, Forces: []forceOption{},
	}
	out.Movable = e.Source == imports.FromFolder && !f.Seeded
	if f.PlacedBy != "" {
		out.PlacedBy = api.Some(string(f.PlacedBy))
	}
	if !f.FileID.IsZero() {
		out.FileID = api.Some(f.FileID.String())
	}
	if f.Probe != nil {
		out.Quality = new(f.Quality.String())
	}
	for i, it := range f.Items {
		out.ItemIDs[i] = it.String()
	}
	for _, fl := range f.Fills {
		out.Fills = append(out.Fills, cellRef{ItemID: fl.Cell.Item.String(), Edition: string(fl.Cell.Edition), TargetID: fl.Cell.Target.String()})
	}
	recommended, also := f.Actions(e)
	if recommended != "" {
		out.Recommended = api.Some(api.NewEnum[action](string(recommended)))
	}
	for _, a := range also {
		out.Actions = append(out.Actions, api.NewEnum[action](string(a)))
	}
	if f.Reason == nil {
		return out
	}
	r := f.Scan()
	rs := reason{Code: string(f.Reason.Code), Title: r.Title, Year: r.Year, Count: r.Count, Names: r.Names, Line: r.Line}
	if f.Reason.Scan == nil {
		rs.Line = f.Reason.Line
	}
	if said, ok := c.sentences[e.ID][f.Position]; ok {
		rs.Sentence = api.Some(said)
	}
	if id, err := storage.ParseUUID(r.Root); err == nil {
		rs.Root = new(c.label(roots.ID{UUID: id}))
	}
	out.Reason = api.Some(rs)
	return out
}

type listInput struct {
	Source  string `query:"source" enum:"grab,root,folder" doc:"Only Imports from this source; every source's when left out"`
	RootID  string `query:"rootId" format:"uuid" doc:"Only this root's Imports and lead; every root's when left out"`
	WorkID  string `query:"work" format:"uuid" doc:"Only the Imports into this Work; every Work's when left out"`
	Ignored bool   `query:"ignored" doc:"List the ignored Imports instead of the ones that need the Admin"`
	Waiting bool   `query:"waiting" doc:"Only the Imports with a file that waits for the Admin"`
	After   string `query:"after" format:"uuid" doc:"The next value of the page before; the first page when left out"`
	Limit   int    `query:"limit" minimum:"1" maximum:"500" doc:"The most Imports a page holds; every one when left out"`
}

type list struct {
	Leads   []lead        `json:"leads" nullable:"false" doc:"One per root a library scan compared, in root order; empty when source leaves out root"`
	Imports []importEntry `json:"imports" nullable:"false" doc:"The unfinished Imports, in root and folder order"`
	Next    *string       `json:"next" format:"uuid" doc:"Pass it as after for the next page; null on the last page"`
}

type listOutput struct{ Body list }

type importIDsInput struct {
	Body struct {
		ImportIDs []string `json:"importIds" format:"uuid" minItems:"1" nullable:"false"`
	}
}

type importIDInput struct {
	ID string `path:"id" format:"uuid"`
}

type entryOutput struct{ Body importEntry }

type rootAction struct{}

func (rootAction) Variants() (discriminator string, variants []any) {
	return "action", []any{matchAction{}, intoAction{}}
}

type matchAction struct {
	Action string `json:"action" enum:"match"`
	Key    string `json:"key" minLength:"1" doc:"A record's key at the Content Type's source, as searchCatalog lists it; it becomes the suggestion"`
}

type intoAction struct {
	Action string       `json:"action" enum:"into"`
	WorkID string       `json:"workId" format:"uuid" doc:"A Work in the Import's root"`
	Files  []chosenFile `json:"files" minItems:"1" nullable:"false"`
}

type chosenFile struct {
	Position int      `json:"position" minimum:"0" doc:"A waiting file of the Import"`
	ItemIDs  []string `json:"itemIds" minItems:"1" nullable:"false" doc:"The Items of the Work the file holds, in any order"`
}

type rootActionInput struct {
	ID   string `path:"id" format:"uuid"`
	Body api.OneOf[rootAction]
}

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
	Code   string `json:"code" doc:"The code a request for this Import alone would have been refused with"`
	Title  string `json:"title"`
	Detail string `json:"detail,omitempty"`
}

type addedList struct {
	Outcomes []api.OneOf[outcome] `json:"outcomes" nullable:"false" doc:"One per Import, in request order"`
}

type addedOutput struct{ Body addedList }

type settings struct {
	Settings []api.OneOf[settingapi.Field] `json:"settings" nullable:"false" doc:"The stop share, then the files each Node probes at once"`
}

type settingsOutput struct{ Body settings }

type importingSettings struct {
	Settings []api.OneOf[settingapi.Field] `json:"settings" nullable:"false" doc:"The video file extensions, the smallest video file, then the runtime under which a video is a sample"`
}

type importingOutput struct{ Body importingSettings }

type libraryScanSettingsInput struct {
	Body struct {
		Values map[string]json.RawMessage `json:"values" doc:"Each setting's new value by key, all or none; null resets one to its default"`
	}
}

func Routes(r *api.Router, p Pending) {
	o := r.Owner("imports", api.Admin)
	h := handlers{p}

	api.Register(o, api.Admin, api.Op{
		ID:      "listImports",
		Method:  http.MethodGet,
		Path:    "/imports",
		Summary: "List the unfinished Imports, with what each root's last library scan added",
		Status:  http.StatusOK,
		Errors:  []*api.Code{importListMoved},
	}, h.list)

	api.Register(o, api.Admin, api.Op{
		ID:      "getImport",
		Method:  http.MethodGet,
		Path:    "/imports/{id}",
		Summary: "Get an Import with its files",
		Status:  http.StatusOK,
		Errors:  []*api.Code{importNotFound},
	}, h.get)

	api.Register(o, api.Admin, api.Op{
		ID:      "actOnRootImport",
		Method:  http.MethodPost,
		Path:    "/imports/{id}",
		Summary: "Match a root Import to a record at the Content Type's source, or import its chosen files into a Work in its root",
		Status:  http.StatusOK,
		Errors: []*api.Code{
			importNotFound, importElsewhere, importChanged, importItems, contentTypeDisabled, metadataNotFound, metadataUnavailable,
			metadataInvalid, metadataRefused, notOnSource,
		},
	}, h.act)

	api.Register(o, api.Admin, api.Op{
		ID:      "addRootImports",
		Method:  http.MethodPost,
		Path:    "/imports/add",
		Summary: "Add each root Import's suggested Work to its root and attach its files",
		Status:  http.StatusOK,
	}, h.add)

	api.Register(o, api.Admin, api.Op{
		ID:      "ignoreImports",
		Method:  http.MethodPost,
		Path:    "/imports/ignore",
		Summary: "Ignore Imports until their files change",
		Status:  http.StatusNoContent,
	}, h.ignore)

	api.Register(o, api.Admin, api.Op{
		ID:      "unignoreImports",
		Method:  http.MethodPost,
		Path:    "/imports/unignore",
		Summary: "Bring ignored Imports back to the ones that need the Admin",
		Status:  http.StatusNoContent,
	}, h.unignore)

	api.Register(o, api.Admin, api.Op{
		ID:      "getLibraryScanSettings",
		Method:  http.MethodGet,
		Path:    "/imports/scan-settings",
		Summary: "Get the Library scan settings",
		Status:  http.StatusOK,
	}, h.getSettings)

	api.Register(o, api.Admin, api.Op{
		ID:      "updateLibraryScanSettings",
		Method:  http.MethodPatch,
		Path:    "/imports/scan-settings",
		Summary: "Change the Library scan settings",
		Status:  http.StatusOK,
		Errors:  []*api.Code{settingapi.ErrOutOfRange, settingapi.ErrUnknown},
	}, h.updateSettings)

	api.Register(o, api.Admin, api.Op{
		ID:      "importFolder",
		Method:  http.MethodPost,
		Path:    "/imports",
		Summary: "Open the Imports of a folder on a Node, one per Work and Edition its files match",
		Status:  http.StatusCreated,
		Also:    []int{http.StatusOK},
		Errors:  []*api.Code{importOutside},
	}, h.importFolder)

	api.Register(o, api.Admin, api.Op{
		ID:      "actOnImportFile",
		Method:  http.MethodPost,
		Path:    "/imports/{id}/files/{position}",
		Summary: "Confirm, retry, decline, force or match one file of an Import",
		Status:  http.StatusOK,
		Errors:  []*api.Code{notBuilt, importNotFound, importNotOffered, importItems},
	}, h.actOnFile)

	api.Register(o, api.Admin, api.Op{
		ID:      "confirmImports",
		Method:  http.MethodPost,
		Path:    "/imports/confirm",
		Summary: "Import every file of these Imports that waits for confirmation",
		Status:  http.StatusOK,
		Errors:  []*api.Code{importNotFound},
	}, h.confirm)

	api.Register(o, api.Admin, api.Op{
		ID:      "setImportMoveFiles",
		Method:  http.MethodPost,
		Path:    "/imports/move-files",
		Summary: "Turn Move files on or off for folder Imports",
		Status:  http.StatusNoContent,
		Errors:  []*api.Code{importNotFound, importNotOffered, importMoveSeeding},
	}, h.moveFiles)

	api.Register(o, api.Admin, api.Op{
		ID:      "getImportingSettings",
		Method:  http.MethodGet,
		Path:    "/imports/importing-settings",
		Summary: "Get the Importing settings",
		Status:  http.StatusOK,
	}, h.getImporting)

	api.Register(o, api.Admin, api.Op{
		ID:      "updateImportingSettings",
		Method:  http.MethodPatch,
		Path:    "/imports/importing-settings",
		Summary: "Change the Importing settings",
		Status:  http.StatusOK,
		Errors:  []*api.Code{settingapi.ErrOutOfRange, settingapi.ErrUnknown},
	}, h.updateImporting)

	api.Register(o, api.Admin, api.Op{
		ID:      "getExtensionImpact",
		Method:  http.MethodGet,
		Path:    "/imports/importing-settings/extension-impact",
		Summary: "Count the library and pending files a video extension list stops counting as video",
		Status:  http.StatusOK,
	}, h.extensionImpact)
}

type handlers struct{ pending Pending }

func (h handlers) list(ctx context.Context, by identity.Admin, in *listInput) (*listOutput, error) {
	f := imports.Filter{Ignored: in.Ignored, Waiting: in.Waiting, Limit: in.Limit}
	if in.Source != "" {
		f.Source = new(imports.Source(in.Source))
	}
	for _, q := range []struct {
		value string
		set   func(storage.UUID)
	}{
		{in.RootID, func(id storage.UUID) { f.Root = &roots.ID{UUID: id} }},
		{in.WorkID, func(id storage.UUID) { f.Work = &works.WorkID{UUID: id} }},
		{in.After, func(id storage.UUID) { f.After = &imports.ImportID{UUID: id} }},
	} {
		if q.value == "" {
			continue
		}
		id, err := storage.ParseUUID(q.value)
		if err != nil {
			return nil, err
		}
		q.set(id)
	}
	l, err := open(h.pending).List(ctx, by, f)
	if err != nil {
		return nil, answer(err)
	}
	c := converter{m: minter(h.pending), roots: l.Roots, sentences: map[imports.ImportID]map[int]string{}}
	for _, e := range l.Imports {
		if c.sentences[e.ID], err = open(h.pending).Sentences(ctx, by, e); err != nil {
			return nil, err
		}
	}
	out := list{Leads: make([]lead, len(l.Leads)), Imports: make([]importEntry, len(l.Imports))}
	for i, ld := range l.Leads {
		out.Leads[i] = lead{Root: c.label(ld.Root), ContentType: string(ld.ContentType), Added: ld.Added, NeedsYou: ld.NeedsYou}
	}
	for i, e := range l.Imports {
		out.Imports[i] = c.entry(e)
	}
	if l.Next != nil {
		out.Next = new(l.Next.String())
	}
	return &listOutput{Body: out}, nil
}

func (h handlers) get(ctx context.Context, by identity.Admin, in *importIDInput) (*entryOutput, error) {
	id, err := importID(in.ID)
	if err != nil {
		return nil, err
	}
	e, words, err := open(h.pending).Get(ctx, by, id)
	if err != nil {
		return nil, answer(err)
	}
	said, err := open(h.pending).Sentences(ctx, by, e)
	if err != nil {
		return nil, err
	}
	c := converter{m: minter(h.pending), roots: words, sentences: map[imports.ImportID]map[int]string{e.ID: said}}
	return &entryOutput{Body: c.entry(e)}, nil
}

func (h handlers) add(ctx context.Context, by identity.Admin, in *importIDsInput) (*addedOutput, error) {
	ids, err := importIDs(in.Body.ImportIDs)
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

func (h handlers) act(ctx context.Context, by identity.Admin, in *rootActionInput) (*entryOutput, error) {
	id, err := importID(in.ID)
	if err != nil {
		return nil, err
	}
	switch a := in.Body.Value().(type) {
	case matchAction:
		_, err = open(h.pending).Match(ctx, by, id, works.Key(a.Key))
	case intoAction:
		_, err = h.into(ctx, by, id, a)
	}
	if err != nil {
		return nil, answer(err)
	}
	return h.get(ctx, by, &importIDInput{ID: in.ID})
}

func (h handlers) into(ctx context.Context, by identity.Admin, id imports.ImportID, a intoAction) (imports.Import, error) {
	work, err := storage.ParseUUID(a.WorkID)
	if err != nil {
		return imports.Import{}, importItems
	}
	chosen := make([]imports.Chosen, len(a.Files))
	for i, f := range a.Files {
		chosen[i] = imports.Chosen{Position: f.Position, Items: make([]works.ItemID, len(f.ItemIDs))}
		for j, s := range f.ItemIDs {
			item, err := storage.ParseUUID(s)
			if err != nil {
				return imports.Import{}, importItems
			}
			chosen[i].Items[j] = works.ItemID{UUID: item}
		}
	}
	return open(h.pending).Into(ctx, by, id, works.WorkID{UUID: work}, chosen)
}

func (h handlers) ignore(ctx context.Context, by identity.Admin, in *importIDsInput) (*struct{}, error) {
	ids, err := importIDs(in.Body.ImportIDs)
	if err != nil {
		return nil, err
	}
	return nil, open(h.pending).Ignore(ctx, by, ids)
}

func (h handlers) unignore(ctx context.Context, by identity.Admin, in *importIDsInput) (*struct{}, error) {
	ids, err := importIDs(in.Body.ImportIDs)
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

func (h handlers) getImporting(ctx context.Context, _ identity.Admin, _ *struct{}) (*importingOutput, error) {
	return importingOf(ctx, open(h.pending).Importing())
}

func (h handlers) updateImporting(ctx context.Context, _ identity.Admin, in *libraryScanSettingsInput) (*importingOutput, error) {
	p := open(h.pending).Importing()
	if err := p.Apply(ctx, in.Body.Values); err != nil {
		return nil, settingapi.Refused(err)
	}
	return importingOf(ctx, p)
}

func importingOf(ctx context.Context, p setting.Panel) (*importingOutput, error) {
	fields, err := p.Fields(ctx)
	if err != nil {
		return nil, err
	}
	return &importingOutput{Body: importingSettings{Settings: settingapi.FieldsOf(fields)}}, nil
}

func (h handlers) extensionImpact(ctx context.Context, by identity.Admin, in *extensionImpactInput) (*extensionImpactOutput, error) {
	library, pending, err := open(h.pending).ExtensionImpact(ctx, by, in.Extensions)
	if err != nil {
		return nil, err
	}
	return &extensionImpactOutput{Body: extensionImpact{Files: int(library), Pending: int(pending)}}, nil
}

func importID(s string) (imports.ImportID, error) {
	u, err := storage.ParseUUID(s)
	if err != nil {
		return imports.ImportID{}, importNotFound
	}
	return imports.ImportID{UUID: u}, nil
}

func importIDs(ss []string) ([]imports.ImportID, error) {
	out := make([]imports.ImportID, len(ss))
	for i, s := range ss {
		u, err := storage.ParseUUID(s)
		if err != nil {
			return nil, err
		}
		out[i] = imports.ImportID{UUID: u}
	}
	return out, nil
}
