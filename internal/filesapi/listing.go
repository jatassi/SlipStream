package filesapi

import (
	"context"
	"errors"
	"path"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
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

func (h handlers) listWorkFiles(ctx context.Context, by identity.Admin, in *cellsInput) (*fileListOutput, error) {
	u, err := storage.ParseUUID(in.WorkID)
	if err != nil {
		return nil, workNotFound
	}
	l, err := open(h.lib).WorkFiles(ctx, by, works.WorkID{UUID: u})
	if errors.Is(err, works.ErrNoWork) {
		return nil, workNotFound
	}
	if err != nil {
		return nil, err
	}
	out := fileList{Folder: l.Folder, Files: make([]listedFile, len(l.Files))}
	for i, f := range l.Files {
		fills := make([]fill, len(f.Fills))
		for j, k := range f.Fills {
			fills[j] = fill{ItemID: k.Item.String(), Edition: string(k.Edition), TargetID: k.Target.String()}
		}
		out.Files[i] = listedFile{LibraryFile: fileOf(f.File), Folder: path.Dir(f.Path), Kept: f.Kept(), Fills: fills}
	}
	return &fileListOutput{Body: out}, nil
}
