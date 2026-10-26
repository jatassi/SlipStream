package filesapi

import (
	"context"

	"github.com/jatassi/slipstream/internal/identity"
)

type listedFile struct {
	LibraryFile
	Folder string `json:"folder" doc:"The folder it sits in, relative to its root, / separated"`
	Kept   bool   `json:"kept" doc:"A Kept file, which fills no cell"`
	Fills  []fill `json:"fills" nullable:"false" doc:"The cells it fills; empty for a Kept file"`
}

type fill struct {
	ItemID   string `json:"itemId" format:"uuid"`
	Edition  string `json:"edition" doc:"any under any cut, else the key of the cut the cell wants"`
	TargetID string `json:"targetId" format:"uuid"`
}

type fileList struct {
	Folder string       `json:"folder" doc:"The Work's folder under its root, the one its naming template gives while it has no file"`
	Files  []listedFile `json:"files" nullable:"false" doc:"Filling files first, in Profile order of the first Target each fills, then Kept files"`
}

type fileListOutput struct{ Body fileList }

func (h handlers) listWorkFiles(context.Context, identity.Admin, *cellsInput) (*fileListOutput, error) {
	return nil, notImplemented
}
