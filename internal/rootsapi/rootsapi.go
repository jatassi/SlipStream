package rootsapi

import (
	"context"
	"errors"
	"net/http"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
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
	workNotFound        = api.NewCode(http.StatusNotFound, "work_not_found", "No Work has that id")
	rootUnusable        = api.NewCode(http.StatusUnprocessableEntity, "root_unusable", "The folder can't be a root")
	rootWrongType       = api.NewCode(http.StatusUnprocessableEntity, "root_wrong_type", "The root holds another Content Type's Works")
	workHasFiles        = api.NewCode(http.StatusConflict, "work_has_files", "The Work has files, so its root can't change")
)

var codes = []struct {
	err  error
	code *api.Code
}{
	{roots.ErrNoType, contentTypeNotFound},
	{roots.ErrNotFound, rootNotFound},
	{works.ErrNoWork, workNotFound},
	{roots.ErrWrongType, rootWrongType},
	{roots.ErrHasFiles, workHasFiles},
}

func answer(err error) error {
	if unusable, ok := errors.AsType[*roots.UnusableError](err); ok {
		return rootUnusable.Because(unusable.Reason)
	}
	for _, c := range codes {
		if errors.Is(err, c.err) {
			return c.code
		}
	}
	return err
}

type root struct {
	ID          string `json:"id" format:"uuid"`
	ContentType string `json:"contentType" pattern:"^[a-z][a-z0-9]{1,15}$"`
	NodeID      string `json:"nodeId" format:"uuid" doc:"The Node whose disk holds it"`
	Path        string `json:"path"`
	Default     bool   `json:"default" doc:"Where its Content Type's new Works go"`
	Works       int64  `json:"works" doc:"The Works that live in it"`
	FileBytes   int64  `json:"fileBytes" doc:"The bytes of its files"`
}

func rootOf(v roots.View) root {
	return root{
		ID: v.ID.String(), ContentType: string(v.ContentType), NodeID: v.Node.String(), Path: v.Path,
		Default: v.Default, Works: v.Works, FileBytes: v.Use.Bytes,
	}
}

type newRoot struct {
	ContentType string `json:"contentType" pattern:"^[a-z][a-z0-9]{1,15}$"`
	Path        string `json:"path" minLength:"1" maxLength:"4096" doc:"An absolute path to an existing folder on the Controller"`
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

type workRoot struct {
	WorkID string `json:"workId" format:"uuid"`
	RootID string `json:"rootId" format:"uuid"`
}

type setWorkRootInput struct {
	WorkID string `path:"workId" format:"uuid"`
	Body   struct {
		RootID string `json:"rootId" format:"uuid"`
	}
}

type workRootOutput struct{ Body workRoot }

func Routes(r *api.Router, folders Folders) {
	o := r.Owner("roots", api.Admin)
	h := handlers{folders}

	api.Register(o, api.Admin, api.Op{
		ID:      "createRoot",
		Method:  http.MethodPost,
		Path:    "/roots",
		Summary: "Make a folder on the Controller a library root of a Content Type",
		Status:  http.StatusCreated,
		Errors:  []*api.Code{contentTypeNotFound, rootUnusable},
	}, h.createRoot)

	api.Register(o, api.Admin, api.Op{
		ID:      "listRoots",
		Method:  http.MethodGet,
		Path:    "/roots",
		Summary: "List the library roots",
		Status:  http.StatusOK,
	}, h.listRoots)

	api.Register(o, api.Admin, api.Op{
		ID:      "makeDefaultRoot",
		Method:  http.MethodPost,
		Path:    "/roots/{rootId}/default",
		Summary: "Make a root where its Content Type's new Works go",
		Status:  http.StatusNoContent,
		Errors:  []*api.Code{rootNotFound},
	}, h.makeDefaultRoot)

	api.Register(o, api.Admin, api.Op{
		ID:      "setWorkRoot",
		Method:  http.MethodPatch,
		Path:    "/works/{workId}",
		Summary: "Move a Work that has no files into another root of its Content Type",
		Status:  http.StatusOK,
		Errors:  []*api.Code{workNotFound, rootNotFound, workHasFiles, rootWrongType},
	}, h.setWorkRoot)
}

type handlers struct{ folders Folders }

func (h handlers) createRoot(ctx context.Context, by identity.Admin, in *createRootInput) (*rootOutput, error) {
	r, err := open(h.folders).Create(ctx, by, roots.NewRoot{ContentType: works.TypeID(in.Body.ContentType), Path: in.Body.Path, MakeDefault: in.Body.MakeDefault})
	if err != nil {
		return nil, answer(err)
	}
	return &rootOutput{Body: rootOf(r)}, nil
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

func (h handlers) makeDefaultRoot(ctx context.Context, by identity.Admin, in *rootInput) (*struct{}, error) {
	id, err := rootID(in.RootID)
	if err != nil {
		return nil, err
	}
	return nil, answer(open(h.folders).MakeDefault(ctx, by, id))
}

func (h handlers) setWorkRoot(ctx context.Context, by identity.Admin, in *setWorkRootInput) (*workRootOutput, error) {
	u, err := storage.ParseUUID(in.WorkID)
	if err != nil {
		return nil, workNotFound
	}
	id, err := rootID(in.Body.RootID)
	if err != nil {
		return nil, err
	}
	if err := open(h.folders).SetWorkRoot(ctx, by, works.WorkID{UUID: u}, id); err != nil {
		return nil, answer(err)
	}
	return &workRootOutput{Body: workRoot{WorkID: u.String(), RootID: id.String()}}, nil
}

func rootID(s string) (roots.ID, error) {
	u, err := storage.ParseUUID(s)
	if err != nil {
		return roots.ID{}, rootNotFound
	}
	return roots.ID{UUID: u}, nil
}
