package filesapi

import (
	"time"

	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/settingapi"
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

var (
	workNotFound = api.NewCode(http.StatusNotFound, "work_not_found", "No Work has that id")
	fileNotFound = api.NewCode(http.StatusNotFound, "file_not_found", "No file has that id")
	filesNotGone = api.NewCode(http.StatusConflict, "files_not_gone", "Only gone files can be forgotten")
)

type probe struct {
	Width  int `json:"width" minimum:"0"`
	Height int `json:"height" minimum:"0"`
}

// LibraryFile is exported so the document carries the fields a listed file
// embeds.
type LibraryFile struct {
	ID      string `json:"id" format:"uuid"`
	Path    string `json:"path" doc:"Relative to its root, / separated"`
	Size    int64  `json:"size" minimum:"0"`
	Quality string `json:"quality" enum:"sdtv,dvd,webrip-480p,hdtv-720p,webrip-720p,webdl-720p,bluray-720p,hdtv-1080p,webrip-1080p,webdl-1080p,bluray-1080p,remux-1080p,hdtv-2160p,webrip-2160p,webdl-2160p,bluray-2160p,remux-2160p,cam"`
	Probe   probe  `json:"probe"`
}

func fileOf(f files.File) LibraryFile {
	return LibraryFile{ID: f.ID.String(), Path: f.Path, Size: f.Size, Quality: f.Quality.String(), Probe: probe{Width: f.Media.Width, Height: f.Media.Height}}
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
	Wanted     bool   `json:"wanted" doc:"Missing, failed or upgradable with no Grab in flight, and its Item monitored, listed and released"`
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
	State  string                       `json:"state" enum:"downloading" doc:"A Grab is in flight for it, fetching included"`
	GrabID string                       `json:"grabId" format:"uuid" doc:"The latest Grab in flight for it"`
	File   api.Nullable[LibraryFile]    `json:"file" doc:"The file the Grab would replace; null when it fills no file"`
	Step   api.Nullable[api.Enum[step]] `json:"step" doc:"What the Grab or its Import is doing for the cell; null until SlipStream reads it"`
	From   *time.Time                   `json:"from" doc:"When the Grab entered its stage; null with step"`
	Until  *time.Time                   `json:"until" doc:"When the Import delay ends, for a download that waits for it; null otherwise"`
}

// step is what holds a cell, in the order a download walks it.
type step struct{}

func (step) Values() []string {
	return []string{"fetching", "downloading", "reading", "sending", "placing", "waiting"}
}

type heldCell struct {
	GrabID string         `json:"grabId" format:"uuid" doc:"The Grab in flight for the cell"`
	Step   api.Enum[step] `json:"step"`
	From   time.Time      `json:"from" doc:"When the Grab entered its stage"`
	Until  *time.Time     `json:"until" doc:"When the Import delay ends, for a download that waits for it; null otherwise"`
}

type failedCell struct {
	CellBase
	State  string                    `json:"state" enum:"failed"`
	GrabID string                    `json:"grabId" format:"uuid" doc:"The Grab that failed"`
	File   api.Nullable[LibraryFile] `json:"file" doc:"The file the Grab would have replaced; null when it fills no file"`
}

type upgradableCell struct {
	CellBase
	State string      `json:"state" enum:"upgradable" doc:"No Grab is in flight for it. A Grab in flight makes it downloading"`
	File  LibraryFile `json:"file" doc:"The file filling it, short of its Target's cutoff or upgrade-until score"`
}

type metCell struct {
	CellBase
	State string                 `json:"state" enum:"met"`
	File  LibraryFile            `json:"file" doc:"The file filling it"`
	Held  api.Nullable[heldCell] `json:"held" doc:"The Grab in flight for it; null while none is"`
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
		d := downloadingCell{CellBase: base, State: state, GrabID: c.Grab.String(), File: replaced(c), Until: untilOf(c.Doing)}
		if c.Doing.Step != "" {
			d.Step, d.From = api.Some(api.NewEnum[step](string(c.Doing.Step))), new(c.Doing.From)
		}
		return api.NewOneOf[cell](d)
	case files.Failed:
		return api.NewOneOf[cell](failedCell{base, state, c.Grab.String(), replaced(c)})
	case files.Upgradable:
		return api.NewOneOf[cell](upgradableCell{CellBase: base, State: state, File: fileOf(*c.File)})
	case files.Met:
		m := metCell{CellBase: base, State: state, File: fileOf(*c.File)}
		if c.Doing.Step != "" {
			m.Held = api.Some(heldCell{GrabID: c.Grab.String(), Step: api.NewEnum[step](string(c.Doing.Step)), From: c.Doing.From, Until: untilOf(c.Doing)})
		}
		return api.NewOneOf[cell](m)
	}
	panic(fmt.Sprintf("filesapi: a cell in state %d", c.State))
}

// untilOf is when the Import delay d waits for ends, nil when it waits for
// none.
func untilOf(d files.Doing) *time.Time {
	if d.Until.IsZero() {
		return nil
	}
	return new(d.Until)
}

// replaced is the file a Grab of c would replace.
func replaced(c files.Cell) api.Nullable[LibraryFile] {
	if c.File == nil {
		return api.Nullable[LibraryFile]{}
	}
	return api.Some(fileOf(*c.File))
}

type cellsInput struct {
	WorkID string `path:"workId" format:"uuid"`
}

type itemStatus struct {
	ItemID   string                        `json:"itemId" format:"uuid"`
	Status   api.Nullable[api.Enum[State]] `json:"status" doc:"The first state its cells hold, in the glossary's order; null for an Item with no cell"`
	Released bool                          `json:"released" doc:"Its availability date is today or earlier"`
}

type levelRollup struct {
	LevelID    string            `json:"levelId" format:"uuid"`
	Cells      api.Counts[State] `json:"cells" doc:"Its monitored Items' cells by state"`
	Items      int               `json:"items" minimum:"0" doc:"Its monitored Items that its source still lists"`
	ItemStatus api.Counts[State] `json:"itemStatus" doc:"Its monitored Items by status; an Item with no cell isn't counted"`
	Withdrawn  api.Counts[State] `json:"withdrawn" doc:"The cells of its Items its source no longer lists that a file still fills, by state"`
}

type cellList struct {
	Cells    []api.OneOf[cell] `json:"cells" nullable:"false" doc:"By Item in tree order, then Target in Profile order"`
	Items    []itemStatus      `json:"items" nullable:"false" doc:"Every Item in tree order"`
	Summary  summary           `json:"summary" doc:"What the cells add up to for the Work"`
	Levels   []levelRollup     `json:"levels" nullable:"false" doc:"What each level's cells add up to, in the order its first Item comes; empty for a type with no levels"`
	NeedsYou []api.OneOf[need] `json:"needsYou" nullable:"false" doc:"What the Work needs from an Admin, failures first"`
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
		Gzip:    true,
		Errors:  []*api.Code{workNotFound},
	}, h.getWorkCells)

	api.Register(o, api.Admin, api.Op{
		ID:      "listWorkFiles",
		Method:  http.MethodGet,
		Path:    "/works/{workId}/files",
		Summary: "List a Work's library files with the cells each fills, and the folder they sit in",
		Status:  http.StatusOK,
		Errors:  []*api.Code{workNotFound},
	}, h.listWorkFiles)

	api.Register(o, api.Admin, api.Op{
		ID:      "forgetFiles",
		Method:  http.MethodPost,
		Path:    "/files/forget",
		Summary: "Delete the records of gone files, all or none",
		Status:  http.StatusNoContent,
		Errors:  []*api.Code{fileNotFound, filesNotGone},
	}, h.forgetFiles)

	api.Register(o, api.Admin, api.Op{
		ID:      "planRenames",
		Method:  http.MethodPost,
		Path:    "/renames/plan",
		Summary: "Plan the renames of a Work, a Content Type or the library, each Work's counts and digest, under the stored naming settings or a draft",
		Status:  http.StatusOK,
		Gzip:    true,
		Errors:  []*api.Code{workNotFound, contentTypeNotFound, templateRefused, settingapi.ErrUnknown, rootNotReady},
	}, h.planRenames)

	api.Register(o, api.Admin, api.Op{
		ID:      "planWorkRenames",
		Method:  http.MethodPost,
		Path:    "/renames/plan/{workId}",
		Summary: "Plan the renames of one Work, each file's row with its companions, under the stored naming settings or a draft",
		Status:  http.StatusOK,
		Errors:  []*api.Code{workNotFound, contentTypeNotFound, templateRefused, settingapi.ErrUnknown, rootNotReady},
	}, h.planWorkRenames)

	api.Register(o, api.Admin, api.Op{
		ID:      "renameFiles",
		Method:  http.MethodPost,
		Path:    "/renames",
		Summary: "Rename the files of each Work whose plan still has the digest shown, and list the Works left because their plans differ now",
		Status:  http.StatusOK,
		MaxBody: confirmLimit,
		Errors:  []*api.Code{rootNotReady},
	}, h.renameFiles)

	api.Register(o, api.Admin, api.Op{
		ID:      "countUnmatched",
		Method:  http.MethodGet,
		Path:    "/renames/unmatched",
		Summary: "Count, per Content Type, the files whose path differs from what the naming templates make",
		Status:  http.StatusOK,
	}, h.countUnmatched)

	api.Register(o, api.Admin, api.Op{
		ID:      "countMoving",
		Method:  http.MethodGet,
		Path:    "/renames/moving",
		Summary: "Count the files the disk queue still moves, in a Work, a Content Type or the library",
		Status:  http.StatusOK,
	}, h.countMoving)
}

type handlers struct{ lib Library }

func (h handlers) getWorkCells(ctx context.Context, by identity.Admin, in *cellsInput) (*cellListOutput, error) {
	u, err := storage.ParseUUID(in.WorkID)
	if err != nil {
		return nil, workNotFound
	}
	items, cells, rollup, err := open(h.lib).WorkCells(ctx, by, works.WorkID{UUID: u})
	if errors.Is(err, works.ErrNoWork) {
		return nil, workNotFound
	}
	if err != nil {
		return nil, err
	}
	out := cellList{Cells: []api.OneOf[cell]{}, Items: []itemStatus{}, Summary: summaryOf(rollup), Levels: []levelRollup{}, NeedsYou: needsOf(rollup.Needs)}
	for _, c := range cells {
		out.Cells = append(out.Cells, variantOf(c))
	}
	for _, it := range items {
		out.Items = append(out.Items, itemStatus{ItemID: it.Item.ID.String(), Status: StatusOf(it.Status), Released: it.Released})
	}
	for _, l := range rollup.Levels {
		out.Levels = append(out.Levels, levelRollup{
			LevelID: l.Level.String(), Cells: CountsOf(l.Cells), Items: l.Items, ItemStatus: CountsOf(l.ByItem), Withdrawn: CountsOf(l.Withdrawn),
		})
	}
	return &cellListOutput{Body: out}, nil
}
