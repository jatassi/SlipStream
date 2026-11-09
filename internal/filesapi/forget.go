package filesapi

import (
	"context"
	"errors"
	"fmt"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/storage"
)

type forgetInput struct {
	Body struct {
		FileIDs []string `json:"fileIds" format:"uuid" minItems:"1" nullable:"false"`
	}
}

func (h handlers) forgetFiles(ctx context.Context, by identity.Admin, in *forgetInput) (*struct{}, error) {
	ids := make([]files.ID, len(in.Body.FileIDs))
	for i, s := range in.Body.FileIDs {
		u, err := storage.ParseUUID(s)
		if err != nil {
			return nil, fileNotFound
		}
		ids[i] = files.ID{UUID: u}
	}
	err := open(h.lib).Forget(ctx, by, ids)
	if errors.Is(err, files.ErrNoFile) {
		return nil, fileNotFound
	}
	if notGone, ok := errors.AsType[*files.NotGoneError](err); ok {
		return nil, filesNotGone.Because(onDisk(notGone.Paths))
	}
	return nil, err
}

// onDisk names the first of paths and counts the rest.
func onDisk(paths []string) string {
	switch len(paths) {
	case 1:
		return paths[0] + " is on disk."
	case 2:
		return paths[0] + " and 1 other file are on disk."
	}
	return fmt.Sprintf("%s and %d other files are on disk.", paths[0], len(paths)-1)
}
