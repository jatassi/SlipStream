package rootsapi

import (
	"context"
	"errors"
	"net/http"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/roots"
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
	rootExists          = api.NewCode(http.StatusConflict, "root_exists", "The Content Type already has a root")
	rootUnusable        = api.NewCode(http.StatusUnprocessableEntity, "root_unusable", "The folder can't be a root")
)

func answer(err error) error {
	var unusable *roots.UnusableError
	switch {
	case errors.As(err, &unusable):
		return rootUnusable.Because(unusable.Reason)
	case errors.Is(err, roots.ErrExists):
		return rootExists
	case errors.Is(err, roots.ErrNoType):
		return contentTypeNotFound
	}
	return err
}

type root struct {
	ID          string `json:"id" format:"uuid"`
	ContentType string `json:"contentType" pattern:"^[a-z][a-z0-9]{1,15}$"`
	NodeID      string `json:"nodeId" format:"uuid" doc:"The Node whose disk holds it"`
	Path        string `json:"path"`
}

func rootOf(r roots.Root) root {
	return root{ID: r.ID.String(), ContentType: string(r.ContentType), NodeID: r.Node.String(), Path: r.Path}
}

type newRoot struct {
	ContentType string `json:"contentType" pattern:"^[a-z][a-z0-9]{1,15}$"`
	Path        string `json:"path" minLength:"1" maxLength:"4096" doc:"An absolute path to an existing folder on the Controller"`
}

type createRootInput struct{ Body newRoot }

type rootOutput struct{ Body root }

type rootList struct {
	Roots []root `json:"roots" nullable:"false" doc:"By Content Type"`
}

type rootListOutput struct{ Body rootList }

func Routes(r *api.Router, folders Folders) {
	o := r.Owner("roots", api.Admin)
	h := handlers{folders}

	api.Register(o, api.Admin, api.Op{
		ID:      "createRoot",
		Method:  http.MethodPost,
		Path:    "/roots",
		Summary: "Make a folder on the Controller a Content Type's library root",
		Status:  http.StatusCreated,
		Errors:  []*api.Code{contentTypeNotFound, rootExists, rootUnusable},
	}, h.createRoot)

	api.Register(o, api.Admin, api.Op{
		ID:      "listRoots",
		Method:  http.MethodGet,
		Path:    "/roots",
		Summary: "List the library roots",
		Status:  http.StatusOK,
	}, h.listRoots)
}

type handlers struct{ folders Folders }

func (h handlers) createRoot(ctx context.Context, by identity.Admin, in *createRootInput) (*rootOutput, error) {
	r, err := open(h.folders).Create(ctx, by, works.TypeID(in.Body.ContentType), in.Body.Path)
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
