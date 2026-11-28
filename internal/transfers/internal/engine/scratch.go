package engine

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/jatassi/slipstream/internal/atomicfile"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/transfers/source"
)

// scratchStagingDir is the folder under the scratch folder a scratch
// destination stages in, so the rename into place stays on one volume.
const scratchStagingDir = ".transfers"

// scratchSource is the path of a scratch source, which opens only under
// the scratch folder.
func (e *Engine) scratchSource(s source.Scratch) (string, error) {
	rel, err := storing.ParseRel(s.Name)
	if err != nil {
		return "", &source.Absent{Reason: source.Gone, Detail: err.Error()}
	}
	return e.cfg.Disk.Host().Style.Under(e.cfg.Scratch(), rel), nil
}

// scratchStaging is the folder a scratch destination's Transfer stages
// in.
func (e *Engine) scratchStaging(ctx context.Context, id ID) (string, error) {
	return e.folders(ctx, e.cfg.Scratch(), scratchStagingDir, id.String())
}

// placeScratch renames a verified staged file to name under the scratch
// folder, replacing what is there.
func (e *Engine) placeScratch(ctx context.Context, staged string, name storing.Rel) error {
	segs := strings.Split(name.String(), "/")
	folder, err := e.folders(ctx, e.cfg.Scratch(), segs[:len(segs)-1]...)
	if err != nil {
		return err
	}
	to := e.cfg.Disk.Host().Style.Join(folder, segs[len(segs)-1])
	if err := atomicfile.Retried(e.cfg.Timings.Retry, func() error { return e.cfg.Disk.Rename(ctx, staged, to) }); err != nil {
		return fmt.Errorf("move the copy into place: %w", err)
	}
	return nil
}

// folders makes each of names under dir, in turn, and returns the last.
func (e *Engine) folders(ctx context.Context, dir string, names ...string) (string, error) {
	style := e.cfg.Disk.Host().Style
	for _, name := range names {
		dir = style.Join(dir, name)
		if err := e.cfg.Disk.Mkdir(ctx, dir); err != nil && !errors.Is(err, fs.ErrExist) {
			return "", fmt.Errorf("make the folder: %w", err)
		}
	}
	return dir, nil
}
