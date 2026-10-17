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
	ID      string `json:"id" format:"uuid"`
	Path    string `json:"path" doc:"Relative to its root, / separated"`
	Size    int64  `json:"size" minimum:"0"`
	Quality string `json:"quality" enum:"sdtv,dvd,webrip-480p,hdtv-720p,webrip-720p,webdl-720p,bluray-720p,hdtv-1080p,webrip-1080p,webdl-1080p,bluray-1080p,remux-1080p,hdtv-2160p,webrip-2160p,webdl-2160p,bluray-2160p,remux-2160p,cam"`
	Probe   probe  `json:"probe"`
}

func fileOf(f files.File) libraryFile {
	return libraryFile{ID: f.ID.String(), Path: f.Path, Size: f.Size, Quality: f.Quality.String(), Probe: probe(f.Probe)}
}

type cell struct{}

func (cell) Variants() (discriminator string, variants []any) {
	return "state", []any{unreleasedCell{}, missingCell{}, downloadingCell{}, failedCell{}, upgradableCell{}, metCell{}}
}

// CellBase is exported so the document carries the fields each cell embeds.
type CellBase struct {
	ItemID     string `json:"itemId" format:"uuid"`
	Edition    string `json:"edition" doc:"any under any cut, else the key of the cut the cell wants"`
	TargetID   string `json:"targetId" format:"uuid"`
	TargetName string `json:"targetName"`
	Wanted     bool   `json:"wanted" doc:"Missing or failed, and its Item monitored and released"`
}

type unreleasedCell struct {
	CellBase
	State string `json:"state" enum:"unreleased" doc:"Its Item has no release date, or a later one"`
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

type upgradableCell struct {
	CellBase
	State string      `json:"state" enum:"upgradable"`
	File  libraryFile `json:"file" doc:"The file filling it, below its Target's cutoff"`
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
	case files.Unreleased:
		return api.NewOneOf[cell](unreleasedCell{base, state})
	case files.Missing:
		return api.NewOneOf[cell](missingCell{base, state})
	case files.Downloading:
		return api.NewOneOf[cell](downloadingCell{base, state})
	case files.Failed:
		return api.NewOneOf[cell](failedCell{base, state})
	case files.Upgradable:
		return api.NewOneOf[cell](upgradableCell{base, state, fileOf(*c.File)})
	case files.Met:
		return api.NewOneOf[cell](metCell{base, state, fileOf(*c.File)})
	}
	panic(fmt.Sprintf("filesapi: a cell in state %d", c.State))
}

type cellsInput struct {
	WorkID string `path:"workId" format:"uuid"`
}

type itemStatus struct {
	ItemID string  `json:"itemId" format:"uuid"`
	Status *string `json:"status" enum:"unreleased,missing,downloading,failed,upgradable,met" doc:"The first state its cells hold, in the glossary's order; null for an Item with no cell"`
}

type cellList struct {
	Cells []api.OneOf[cell] `json:"cells" nullable:"false" doc:"By Item in tree order, then Target in Profile order"`
	Items []itemStatus      `json:"items" nullable:"false" doc:"Every Item in tree order"`
}

type cellListOutput struct{ Body cellList }

func Routes(r *api.Router, lib Library) {
	o := r.Owner("files", api.Admin)
	h := handlers{lib}

	api.Register(o, api.Admin, api.Op{
		ID:      "getWorkCells",
		Method:  http.MethodGet,
		Path:    "/works/{workId}/cells",
		Summary: "Get each of a Work's cells with its state and the file filling it, and each Item's status",
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
	items, cells, err := open(h.lib).WorkCells(ctx, by, works.WorkID{UUID: u})
	if errors.Is(err, works.ErrNoWork) {
		return nil, workNotFound
	}
	if err != nil {
		return nil, err
	}
	out := cellList{Cells: []api.OneOf[cell]{}, Items: []itemStatus{}}
	of := map[works.ItemID][]files.Cell{}
	for _, c := range cells {
		out.Cells = append(out.Cells, variantOf(c))
		of[c.Item.ID] = append(of[c.Item.ID], c)
	}
	for _, it := range items {
		row := itemStatus{ItemID: it.ID.String()}
		if s, ok := files.Status(of[it.ID]); ok {
			row.Status = new(s.String())
		}
		out.Items = append(out.Items, row)
	}
	return &cellListOutput{Body: out}, nil
}
