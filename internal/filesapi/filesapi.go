package filesapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

type Library interface {
	// Files is the library's files, false unless the Node is a Controller.
	Files() (*files.Files, bool)
}

func open(l Library) *files.Files {
	f, ok := l.Files()
	if !ok {
		panic("filesapi: an operation ran on a Node that is no Controller")
	}
	return f
}

var workNotFound = api.NewCode(http.StatusNotFound, "work_not_found", "No Work has that id")

type probe struct {
	Width  int `json:"width" minimum:"0"`
	Height int `json:"height" minimum:"0"`
}

type libraryFile struct {
	ID    string `json:"id" format:"uuid"`
	Path  string `json:"path" doc:"Relative to its root, / separated"`
	Size  int64  `json:"size" minimum:"0"`
	Probe probe  `json:"probe"`
}

type cell struct{}

func (cell) Variants() (discriminator string, variants []any) {
	return "state", []any{missingCell{}, downloadingCell{}, failedCell{}, metCell{}}
}

// CellBase is exported so the document carries the fields each cell embeds.
type CellBase struct {
	ItemID     string `json:"itemId" format:"uuid"`
	Edition    string `json:"edition" enum:"standard"`
	TargetID   string `json:"targetId" format:"uuid"`
	TargetName string `json:"targetName"`
	Wanted     bool   `json:"wanted" doc:"Missing or failed, and its Item monitored"`
}

type missingCell struct {
	CellBase
	State string `json:"state" enum:"missing"`
}

type downloadingCell struct {
	CellBase
	State string `json:"state" enum:"downloading"`
}

type failedCell struct {
	CellBase
	State string `json:"state" enum:"failed"`
}

type metCell struct {
	CellBase
	State string      `json:"state" enum:"met"`
	File  libraryFile `json:"file" doc:"The file filling it"`
}

func variantOf(c files.Cell) api.OneOf[cell] {
	base := CellBase{ItemID: c.Item.ID.String(), Edition: string(c.Edition), TargetID: c.Target.ID.String(), TargetName: c.Target.Name, Wanted: c.Wanted()}
	state := c.State.String()
	switch c.State {
	case files.Missing:
		return api.NewOneOf[cell](missingCell{base, state})
	case files.Downloading:
		return api.NewOneOf[cell](downloadingCell{base, state})
	case files.Failed:
		return api.NewOneOf[cell](failedCell{base, state})
	case files.Met:
		return api.NewOneOf[cell](metCell{base, state, libraryFile{ID: c.File.ID.String(), Path: c.File.Path, Size: c.File.Size, Probe: probe(c.File.Probe)}})
	}
	panic(fmt.Sprintf("filesapi: a cell in state %d", c.State))
}

type cellsInput struct {
	WorkID string `path:"workId" format:"uuid"`
}

type cellList struct {
	Cells []api.OneOf[cell] `json:"cells" nullable:"false" doc:"By Item in tree order, then Target in Profile order"`
}

type cellListOutput struct{ Body cellList }

func Routes(r *api.Router, lib Library) {
	o := r.Owner("files", api.Admin)
	h := handlers{lib}

	api.Register(o, api.Admin, api.Op{
		ID:      "getWorkCells",
		Method:  http.MethodGet,
		Path:    "/works/{workId}/cells",
		Summary: "Get each of a Work's cells with its state and the file filling it",
		Status:  http.StatusOK,
		Errors:  []*api.Code{workNotFound},
	}, h.getWorkCells)
}

type handlers struct{ lib Library }

func (h handlers) getWorkCells(ctx context.Context, by identity.Admin, in *cellsInput) (*cellListOutput, error) {
	u, err := storage.ParseUUID(in.WorkID)
	if err != nil {
		return nil, workNotFound
	}
	cells, err := open(h.lib).WorkCells(ctx, by, works.WorkID{UUID: u})
	if errors.Is(err, works.ErrNoWork) {
		return nil, workNotFound
	}
	if err != nil {
		return nil, err
	}
	out := cellList{Cells: []api.OneOf[cell]{}}
	for _, c := range cells {
		out.Cells = append(out.Cells, variantOf(c))
	}
	return &cellListOutput{Body: out}, nil
}
