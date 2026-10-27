package rootsapi

import (
	"context"
	"errors"
	"io/fs"
	"net/http"
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/settingapi"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/works"
)

type Folders interface {
	// Roots is the library roots, false unless the Node is a Controller.
	Roots() (*roots.Roots, bool)
}

func open(f Folders) *roots.Roots {
	r, ok := f.Roots()
	if !ok {
		panic("rootsapi: an operation ran on a Node that is no Controller")
	}
	return r
}

var (
	contentTypeNotFound = api.NewCode(http.StatusNotFound, "content_type_not_found", "No Content Type has that id")
	rootNotFound        = api.NewCode(http.StatusNotFound, "root_not_found", "No root has that id")
	nodeNotStorage      = api.NewCode(http.StatusConflict, "node_not_storage", "The Node doesn't hold Storage")
	nodeNotFound        = api.NewCode(http.StatusNotFound, "node_not_found", "No Node has this id")
	nodeOffline         = api.NewCode(http.StatusServiceUnavailable, "node_offline", "The Node is offline")
	rootUnusable        = api.NewCarrying[refusal](http.StatusUnprocessableEntity, "root_unusable", "The folder can't be a root", "refusal")
	folderUnreadable    = api.NewCarrying[folderProblem](http.StatusUnprocessableEntity, "folder_unreadable", "The Node can't read the folder", "problem")
	folderInvalid       = api.NewCode(http.StatusUnprocessableEntity, "folder_invalid", "The Node can't use that path or folder name")
	folderExists        = api.NewCode(http.StatusConflict, "folder_exists", "A folder of that name is already there")
	rehomeMissing       = api.NewCarrying[missingFiles](http.StatusConflict, "rehome_missing", "Recorded files aren't at the new place", "missing")
	rootInUse           = api.NewCarrying[inUse](http.StatusConflict, "root_in_use", "The root holds files or has a Grab in flight", "use")
	rootIsDefault       = api.NewCode(http.StatusConflict, "root_is_default", "The root is its Content Type's default while another root exists")
	worksToRequired     = api.NewCode(http.StatusUnprocessableEntity, "works_to_required", "The root's Works need another root of their Content Type to go to")
	worksToWrongType    = api.NewCode(http.StatusUnprocessableEntity, "works_to_wrong_type", "The root's Works can go only to a root of their Content Type")
)

var codes = []struct {
	err  error
	code *api.Code
}{
	{roots.ErrNoType, contentTypeNotFound},
	{roots.ErrNotFound, rootNotFound},
	{roots.ErrNotStorage, nodeNotStorage},
	{roots.ErrNoNode, nodeNotFound},
	{roots.ErrNodeAway, nodeOffline},
	{roots.ErrIsDefault, rootIsDefault},
	{roots.ErrWorksTo, worksToRequired},
	{roots.ErrWorksToType, worksToWrongType},
	{fs.ErrExist, folderExists},
}

func answer(err error) error {
	if unusable, ok := errors.AsType[*roots.UnusableError](err); ok {
		return rootUnusable.With(refusalOf(unusable.Refusal))
	}
	if missing, ok := errors.AsType[*roots.MissingError](err); ok {
		return rehomeMissing.With(missingOf(missing))
	}
	if use, ok := errors.AsType[*roots.InUseError](err); ok {
		return rootInUse.With(inUse{Files: use.Files, Grabs: use.Grabs})
	}
	if fe, ok := errors.AsType[*storing.FolderError](err); ok {
		return folderUnreadable.With(folderProblem{Kind: fe.Kind.String(), OSText: fe.Text, Account: fe.Account})
	}
	if errors.Is(err, storing.ErrInvalid) {
		return folderInvalid.Because(err.Error())
	}
	for _, c := range codes {
		if errors.Is(err, c.err) {
			return c.code
		}
	}
	return err
}

type root struct {
	ID          string    `json:"id" format:"uuid"`
	ContentType string    `json:"contentType" pattern:"^[a-z][a-z0-9]{1,15}$"`
	NodeID      string    `json:"nodeId" format:"uuid" doc:"The Storage Node whose disk holds it"`
	Path        string    `json:"path" doc:"The folder as its Node cleaned it"`
	Default     bool      `json:"default" doc:"Where its Content Type's new Works go"`
	State       string    `json:"state" enum:"ready,missing,unmounted,foreign,unwritable,unresponsive,checking,offline,written_off" doc:"Only ready takes imports. checking: its Node hasn't reported it yet; offline and written_off come from its Node's standing"`
	VolumeID    string    `json:"volumeId,omitempty" doc:"The volume its Node last found it ready on; absent before it did"`
	Works       int64     `json:"works" doc:"The Works that live in it"`
	FileBytes   int64     `json:"fileBytes" doc:"The bytes of its files"`
	Rehome      *progress `json:"rehome,omitempty" doc:"The re-home checking its files at the new place now; absent while none is"`
}

type progress struct {
	Checked int64 `json:"checked" doc:"The recorded files checked so far"`
	Total   int64 `json:"total" doc:"The recorded files to check"`
}

func rootOf(v roots.View) root {
	out := root{
		ID: v.ID.String(), ContentType: string(v.ContentType), NodeID: v.Node.String(), Path: v.Path,
		Default: v.Default, State: string(v.State), VolumeID: v.VolumeID, Works: v.Works, FileBytes: v.Use.Bytes,
	}
	if v.Rehome != nil {
		out.Rehome = &progress{Checked: v.Rehome.Checked, Total: v.Rehome.Total}
	}
	return out
}

type missingFiles struct {
	Count int           `json:"count" doc:"The recorded files not at the new place"`
	First []missingFile `json:"first" nullable:"false" doc:"Up to the first 20 of them, in file order"`
}

type missingFile struct {
	Path   string `json:"path" doc:"Where the file records its bytes, relative to the root and / separated"`
	Reason string `json:"reason" enum:"missing,size_differs" doc:"missing: nothing is at that path; size_differs: a file is there at another size"`
}

func missingOf(e *roots.MissingError) missingFiles {
	out := missingFiles{Count: e.Count, First: []missingFile{}}
	for _, a := range e.First {
		out.First = append(out.First, missingFile{Path: a.Path.String(), Reason: a.Reason.String()})
	}
	return out
}

type inUse struct {
	Files int64 `json:"files" doc:"The files it holds, those waiting to be deleted counted"`
	Grabs int64 `json:"grabs" doc:"The Grabs in flight for its Works or into it"`
}

type refusal struct {
	Reason string `json:"reason" enum:"not_absolute,not_found,not_a_folder,unwritable,inside_root,contains_root,inside_data,contains_data,inside_downloads,contains_downloads,unresponsive,network_share,node_offline" doc:"Why the folder can't serve its purpose"`
	RootID string `json:"rootId,omitempty" format:"uuid" doc:"The root the folder is inside, contains or carries the marker of"`
}

func refusalOf(r roots.Refusal) refusal {
	out := refusal{Reason: string(r.Reason)}
	if r.Root != nil {
		out.RootID = r.Root.String()
	}
	return out
}

type folderProblem struct {
	Kind    string `json:"kind" enum:"not_found,not_a_folder,denied,unresponsive,io"`
	OSText  string `json:"osText" doc:"The error as the Node's OS put it"`
	Account string `json:"account,omitempty" doc:"For denied, the account SlipStream runs as on the Node"`
}

type volume struct {
	ID        string    `json:"id" doc:"The filesystem id, or the volume serial on Windows"`
	Name      string    `json:"name" doc:"The volume as its Node names it, such as D: (Media) or /mnt/media"`
	Mount     string    `json:"mount"`
	Kind      string    `json:"kind" enum:"local,removable,network"`
	Total     int64     `json:"total"`
	Free      int64     `json:"free" doc:"The bytes SlipStream may write"`
	SampledAt time.Time `json:"sampledAt"`
}

func volumeOf(v *storing.Volume) *volume {
	if v == nil {
		return nil
	}
	return &volume{ID: v.ID, Name: v.Name, Mount: v.Mount, Kind: v.Kind.String(), Total: v.Total, Free: v.Free, SampledAt: v.SampledAt}
}

type newRoot struct {
	ContentType string `json:"contentType" pattern:"^[a-z][a-z0-9]{1,15}$"`
	NodeID      string `json:"nodeId" format:"uuid" doc:"A Node that holds Storage"`
	Path        string `json:"path" minLength:"1" maxLength:"4096" doc:"An absolute path to an existing folder on the Node, in its own path style"`
	MakeDefault bool   `json:"makeDefault,omitempty" doc:"Make it its Content Type's default; a Content Type's first root is its default whatever this says"`
}

type createRootInput struct{ Body newRoot }

type rootOutput struct{ Body root }

type rootList struct {
	Roots []root `json:"roots" nullable:"false" doc:"By Content Type, oldest first in each"`
}

type rootListOutput struct{ Body rootList }

type rootInput struct {
	RootID string `path:"rootId" format:"uuid"`
}

type checkFolderInput struct {
	NodeID string `path:"id" format:"uuid"`
	Body   struct {
		Path    string `json:"path" minLength:"1" maxLength:"4096"`
		Purpose string `json:"purpose" enum:"root,downloads" doc:"root: a library root's rules, on a Storage Node; downloads: a downloads folder's, writable, on no network share and apart from every root on the Node"`
		RootID  string `json:"rootId,omitempty" format:"uuid" doc:"For root: the root being re-homed, which the folder may overlap"`
	}
}

type verdict struct {
	Usable  bool     `json:"usable"`
	Path    string   `json:"path" doc:"The folder as its Node cleaned it"`
	Volume  *volume  `json:"volume,omitempty" doc:"The volume it sits on, when the Node read it"`
	Refusal *refusal `json:"refusal,omitempty" doc:"Why it can't serve its purpose; absent when usable"`
}

type verdictOutput struct{ Body verdict }

type rehomeRootInput struct {
	RootID string `path:"rootId" format:"uuid"`
	Body   struct {
		NodeID string `json:"nodeId,omitempty" format:"uuid" doc:"The Storage Node whose disk holds it now; left out, its Node"`
		Path   string `json:"path,omitempty" maxLength:"4096" doc:"The folder that holds it now, in its Node's path style; left out, its folder"`
		Forget *int   `json:"forget,omitempty" minimum:"0" doc:"The missing file count the Admin confirmed from a rehome_missing response: those files are forgotten, so their cells read missing and wanted. A different count is refused with rehome_missing again"`
	}
}

type deleteRootInput struct {
	RootID  string `path:"rootId" format:"uuid"`
	WorksTo string `query:"worksTo" format:"uuid" doc:"The root of the same Content Type its Works go to; required while it has Works and another root of its Content Type exists"`
}

type browseInput struct {
	NodeID string `path:"id" format:"uuid"`
	Path   string `query:"path" maxLength:"4096" doc:"An absolute folder in the Node's path style; empty for its places"`
	Hidden bool   `query:"hidden" doc:"List hidden and system folders too"`
}

type segment struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type folder struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	Readable bool   `json:"readable" doc:"SlipStream can list it"`
}

type place struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	Kind   string `json:"kind,omitempty" enum:"local,removable,network" doc:"Absent for a root whose volume its Node hasn't read yet"`
	Free   int64  `json:"free"`
	Total  int64  `json:"total"`
	RootID string `json:"rootId,omitempty" format:"uuid" doc:"The root this place is"`
}

type listing struct {
	Path     string    `json:"path" doc:"The folder as its Node cleaned it; empty for the Node's places"`
	Parent   string    `json:"parent" doc:"The folder above it; empty at a drive, a share or /, and for the places"`
	Segments []segment `json:"segments" nullable:"false" doc:"Its breadcrumbs as its Node splits it; a drive or a share is one"`
	Volume   *volume   `json:"volume,omitempty" doc:"The volume it sits on, when the Node read it"`
	Folders  []folder  `json:"folders" nullable:"false" doc:"Its subfolders, in natural order without case"`
	Hidden   int       `json:"hidden" doc:"The hidden subfolders left out"`
	Places   []place   `json:"places" nullable:"false" doc:"For the empty path: the roots on the Node, then its drives or mounts"`
	Service  bool      `json:"service" doc:"SlipStream runs as a Windows service there, which sees no mapped drive letters"`
}

func listingOf(l storing.Listing) listing {
	out := listing{
		Path: l.Path, Parent: l.Parent, Volume: volumeOf(l.Volume), Hidden: l.Hidden, Service: l.Service,
		Segments: []segment{}, Folders: []folder{}, Places: []place{},
	}
	for _, s := range l.Segments {
		out.Segments = append(out.Segments, segment(s))
	}
	for _, f := range l.Folders {
		out.Folders = append(out.Folders, folder(f))
	}
	for _, p := range l.Places {
		pl := place{Name: p.Name, Path: p.Path, Free: p.Free, Total: p.Total}
		if p.Kind != 0 {
			pl.Kind = p.Kind.String()
		}
		if !p.Root.IsZero() {
			pl.RootID = p.Root.String()
		}
		out.Places = append(out.Places, pl)
	}
	return out
}

type listingOutput struct{ Body listing }

type suggestInput struct {
	NodeID string `path:"id" format:"uuid"`
	Text   string `query:"text" maxLength:"4096" doc:"A path as typed, in the Node's path style"`
}

type suggestions struct {
	Suggestions []string `json:"suggestions" nullable:"false" doc:"Up to 25 folders completing the text, in the separators it was typed with"`
}

type suggestOutput struct{ Body suggestions }

type makeFolderInput struct {
	NodeID string `path:"id" format:"uuid"`
	Body   struct {
		Parent string `json:"parent" minLength:"1" maxLength:"4096" doc:"An existing folder on the Node"`
		Name   string `json:"name" minLength:"1" maxLength:"255" doc:"The new folder's name"`
	}
}

type madeFolder struct {
	Path string `json:"path" doc:"The new folder, joined by the Node"`
}

type madeFolderOutput struct{ Body madeFolder }

type nodeVolumes struct {
	NodeID  string       `json:"nodeId" format:"uuid"`
	Live    bool         `json:"live" doc:"The Node is online, so its figures are current. Otherwise each volume's figures are from its sampledAt"`
	Volumes []rootVolume `json:"volumes" nullable:"false" doc:"Its volumes that hold a root, by name"`
}

type rootVolume struct {
	ID        string    `json:"id" doc:"The filesystem id, or the volume serial on Windows"`
	Name      string    `json:"name" doc:"The volume as its Node names it, such as D: (Media) or /mnt/media"`
	Mount     string    `json:"mount"`
	Kind      string    `json:"kind" enum:"local,removable,network"`
	Total     int64     `json:"total"`
	Free      int64     `json:"free" doc:"The bytes SlipStream may write"`
	SampledAt time.Time `json:"sampledAt" doc:"When its Node read these figures"`
	RootIDs   []string  `json:"rootIds" nullable:"false" doc:"The roots its Node last found ready on it"`
}

type volumeList struct {
	Nodes []nodeVolumes `json:"nodes" nullable:"false" doc:"Each Storage Node with a volume that holds a root, in id order"`
}

type volumeListOutput struct{ Body volumeList }

type rootSettings struct {
	SampleEvery setting.DurationField `json:"sampleEvery"`
}

type rootSettingsOutput struct{ Body rootSettings }

type rootSettingsInput struct {
	Body struct {
		SampleEvery api.Patch[int64] `json:"sampleEvery,omitempty" doc:"Seconds; null resets it to its default"`
	}
}

func Routes(r *api.Router, folders Folders) {
	o := r.Owner("roots", api.Admin)
	h := handlers{folders}

	api.Register(o, api.Admin, api.Op{
		ID:      "createRoot",
		Method:  http.MethodPost,
		Path:    "/roots",
		Summary: "Make a folder on a Storage Node a library root of a Content Type",
		Status:  http.StatusCreated,
		Errors:  []*api.Code{contentTypeNotFound, nodeNotStorage, rootUnusable.Code, folderUnreadable.Code},
	}, h.createRoot)

	api.Register(o, api.Admin, api.Op{
		ID:      "listRoots",
		Method:  http.MethodGet,
		Path:    "/roots",
		Summary: "List the library roots",
		Status:  http.StatusOK,
	}, h.listRoots)

	api.Register(o, api.Admin, api.Op{
		ID:      "listVolumes",
		Method:  http.MethodGet,
		Path:    "/volumes",
		Summary: "List each Storage Node's volumes that hold a library root",
		Status:  http.StatusOK,
	}, h.listVolumes)

	api.Register(o, api.Admin, api.Op{
		ID:      "getRootSettings",
		Method:  http.MethodGet,
		Path:    "/roots/settings",
		Summary: "Get the library roots settings",
		Status:  http.StatusOK,
	}, h.getRootSettings)

	api.Register(o, api.Admin, api.Op{
		ID:      "updateRootSettings",
		Method:  http.MethodPatch,
		Path:    "/roots/settings",
		Summary: "Change the library roots settings",
		Status:  http.StatusOK,
		Errors:  []*api.Code{settingapi.ErrOutOfRange},
	}, h.updateRootSettings)

	api.Register(o, api.Admin, api.Op{
		ID:      "makeDefaultRoot",
		Method:  http.MethodPost,
		Path:    "/roots/{rootId}/default",
		Summary: "Make a root where its Content Type's new Works go",
		Status:  http.StatusNoContent,
		Errors:  []*api.Code{rootNotFound},
	}, h.makeDefaultRoot)

	api.Register(o, api.Admin, api.Op{
		ID:      "rehomeRoot",
		Method:  http.MethodPatch,
		Path:    "/roots/{rootId}",
		Summary: "Re-home a root onto another folder or Node once every file it records is there",
		Status:  http.StatusOK,
		Errors:  []*api.Code{rootNotFound, nodeNotStorage, nodeOffline, rootUnusable.Code, rehomeMissing.Code, folderUnreadable.Code},
	}, h.rehomeRoot)

	api.Register(o, api.Admin, api.Op{
		ID:      "deleteRoot",
		Method:  http.MethodDelete,
		Path:    "/roots/{rootId}",
		Summary: "Delete a root that holds no files, leaving its folder on disk",
		Status:  http.StatusNoContent,
		Errors:  []*api.Code{rootNotFound, rootInUse.Code, rootIsDefault, worksToRequired, worksToWrongType},
	}, h.deleteRoot)

	api.Register(o, api.Admin, api.Op{
		ID:      "browseFolders",
		Method:  http.MethodGet,
		Path:    "/nodes/{id}/folders",
		Summary: "List a folder on a Node, or its places",
		Status:  http.StatusOK,
		Errors:  []*api.Code{nodeNotFound, nodeOffline, folderUnreadable.Code, folderInvalid},
	}, h.browseFolders)

	api.Register(o, api.Admin, api.Op{
		ID:      "suggestFolders",
		Method:  http.MethodGet,
		Path:    "/nodes/{id}/folders/suggest",
		Summary: "List the folders on a Node that complete a typed path",
		Status:  http.StatusOK,
		Errors:  []*api.Code{nodeNotFound, nodeOffline, folderUnreadable.Code},
	}, h.suggestFolders)

	api.Register(o, api.Admin, api.Op{
		ID:      "makeFolder",
		Method:  http.MethodPost,
		Path:    "/nodes/{id}/folders",
		Summary: "Make a folder in an existing folder on a Node",
		Status:  http.StatusCreated,
		Errors:  []*api.Code{nodeNotFound, nodeOffline, folderUnreadable.Code, folderInvalid, folderExists},
	}, h.makeFolder)

	api.Register(o, api.Admin, api.Op{
		ID:      "checkFolder",
		Method:  http.MethodPost,
		Path:    "/nodes/{id}/folders/check",
		Summary: "Judge a folder on a Node for a purpose without saving it",
		Status:  http.StatusOK,
		Errors:  []*api.Code{nodeNotFound, nodeNotStorage, folderUnreadable.Code},
	}, h.checkFolder)
}

type handlers struct{ folders Folders }

func (h handlers) createRoot(ctx context.Context, by identity.Admin, in *createRootInput) (*rootOutput, error) {
	node, err := nodeID(in.Body.NodeID)
	if err != nil {
		return nil, err
	}
	r, err := open(h.folders).Create(ctx, by, roots.NewRoot{
		ContentType: works.TypeID(in.Body.ContentType), Node: node, Path: in.Body.Path, MakeDefault: in.Body.MakeDefault,
	})
	if err != nil {
		return nil, answer(err)
	}
	return &rootOutput{Body: rootOf(r)}, nil
}

func (h handlers) checkFolder(ctx context.Context, by identity.Admin, in *checkFolderInput) (*verdictOutput, error) {
	node, err := nodeID(in.NodeID)
	if err != nil {
		return nil, err
	}
	purpose, err := storing.ParsePurposeWord(in.Body.Purpose)
	if err != nil {
		return nil, err
	}
	var except *roots.ID
	if in.Body.RootID != "" {
		id, err := rootID(in.Body.RootID)
		if err != nil {
			return nil, err
		}
		except = &id
	}
	v, err := open(h.folders).Check(ctx, by, node, in.Body.Path, purpose, except)
	if err != nil {
		return nil, answer(err)
	}
	out := verdict{Usable: v.Refusal == nil, Path: v.Path, Volume: volumeOf(v.Volume)}
	if v.Refusal != nil {
		r := refusalOf(*v.Refusal)
		out.Refusal = &r
	}
	return &verdictOutput{Body: out}, nil
}

func (h handlers) listRoots(ctx context.Context, by identity.Admin, _ *struct{}) (*rootListOutput, error) {
	list, err := open(h.folders).List(ctx, by)
	if err != nil {
		return nil, err
	}
	out := rootList{Roots: []root{}}
	for _, r := range list {
		out.Roots = append(out.Roots, rootOf(r))
	}
	return &rootListOutput{Body: out}, nil
}

func (h handlers) listVolumes(ctx context.Context, by identity.Admin, _ *struct{}) (*volumeListOutput, error) {
	list, err := open(h.folders).Volumes(ctx, by)
	if err != nil {
		return nil, err
	}
	out := volumeList{Nodes: []nodeVolumes{}}
	for _, n := range list {
		nv := nodeVolumes{NodeID: n.Node.String(), Live: n.Live, Volumes: []rootVolume{}}
		for _, v := range n.Volumes {
			rv := rootVolume{
				ID: v.ID, Name: v.Name, Mount: v.Mount, Kind: v.Kind.String(), Total: v.Total, Free: v.Free, SampledAt: v.SampledAt, RootIDs: []string{},
			}
			for _, id := range v.Roots {
				rv.RootIDs = append(rv.RootIDs, id.String())
			}
			nv.Volumes = append(nv.Volumes, rv)
		}
		out.Nodes = append(out.Nodes, nv)
	}
	return &volumeListOutput{Body: out}, nil
}

func (h handlers) getRootSettings(ctx context.Context, _ identity.Admin, _ *struct{}) (*rootSettingsOutput, error) {
	return currentSettings(ctx, open(h.folders))
}

func (h handlers) updateRootSettings(ctx context.Context, _ identity.Admin, in *rootSettingsInput) (*rootSettingsOutput, error) {
	r := open(h.folders)
	if p := in.Body.SampleEvery; p.Sent() {
		v, err := roots.SampleEvery.Parse(p.Value())
		if err != nil {
			return nil, settingapi.Refused(err)
		}
		if err := r.SetSampleEvery(ctx, v); err != nil {
			return nil, err
		}
	}
	return currentSettings(ctx, r)
}

func currentSettings(ctx context.Context, r *roots.Roots) (*rootSettingsOutput, error) {
	every, err := r.SampleEveryField(ctx)
	if err != nil {
		return nil, err
	}
	return &rootSettingsOutput{Body: rootSettings{SampleEvery: every}}, nil
}

func (h handlers) makeDefaultRoot(ctx context.Context, by identity.Admin, in *rootInput) (*struct{}, error) {
	id, err := rootID(in.RootID)
	if err != nil {
		return nil, err
	}
	return nil, answer(open(h.folders).MakeDefault(ctx, by, id))
}

func (h handlers) rehomeRoot(ctx context.Context, by identity.Admin, in *rehomeRootInput) (*rootOutput, error) {
	id, err := rootID(in.RootID)
	if err != nil {
		return nil, err
	}
	move := roots.Rehome{Forget: in.Body.Forget}
	if in.Body.NodeID != "" {
		node, err := nodeID(in.Body.NodeID)
		if err != nil {
			return nil, err
		}
		move.Node = &node
	}
	if in.Body.Path != "" {
		move.Path = &in.Body.Path
	}
	v, err := open(h.folders).Rehome(ctx, by, id, move)
	if err != nil {
		return nil, answer(err)
	}
	return &rootOutput{Body: rootOf(v)}, nil
}

func (h handlers) deleteRoot(ctx context.Context, by identity.Admin, in *deleteRootInput) (*struct{}, error) {
	id, err := rootID(in.RootID)
	if err != nil {
		return nil, err
	}
	var to *roots.ID
	if in.WorksTo != "" {
		dest, err := rootID(in.WorksTo)
		if err != nil {
			return nil, err
		}
		to = &dest
	}
	return nil, answer(open(h.folders).Delete(ctx, by, id, to))
}

func (h handlers) browseFolders(ctx context.Context, by identity.Admin, in *browseInput) (*listingOutput, error) {
	node, err := nodeID(in.NodeID)
	if err != nil {
		return nil, err
	}
	l, err := open(h.folders).Browse(ctx, by, node, in.Path, in.Hidden)
	if err != nil {
		return nil, answer(err)
	}
	return &listingOutput{Body: listingOf(l)}, nil
}

func (h handlers) suggestFolders(ctx context.Context, by identity.Admin, in *suggestInput) (*suggestOutput, error) {
	node, err := nodeID(in.NodeID)
	if err != nil {
		return nil, err
	}
	paths, err := open(h.folders).Suggest(ctx, by, node, in.Text)
	if err != nil {
		return nil, answer(err)
	}
	return &suggestOutput{Body: suggestions{Suggestions: append([]string{}, paths...)}}, nil
}

func (h handlers) makeFolder(ctx context.Context, by identity.Admin, in *makeFolderInput) (*madeFolderOutput, error) {
	node, err := nodeID(in.NodeID)
	if err != nil {
		return nil, err
	}
	p, err := open(h.folders).MakeFolder(ctx, by, node, in.Body.Parent, in.Body.Name)
	if err != nil {
		return nil, answer(err)
	}
	return &madeFolderOutput{Body: madeFolder{Path: p}}, nil
}

func rootID(s string) (roots.ID, error) {
	u, err := storage.ParseUUID(s)
	if err != nil {
		return roots.ID{}, rootNotFound
	}
	return roots.ID{UUID: u}, nil
}

func nodeID(s string) (role.NodeID, error) {
	u, err := storage.ParseUUID(s)
	if err != nil {
		return role.NodeID{}, nodeNotStorage
	}
	return role.NodeID{UUID: u}, nil
}
