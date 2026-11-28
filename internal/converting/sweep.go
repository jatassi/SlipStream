package converting

import (
	"context"
	"errors"
	"io/fs"
	"slices"
	"strings"
)

// The scratch folder's layout: source copies under in/, outputs and their
// .part names under out/, test clips under clips/. The Transfer receiver
// stages under .transfers/, which the holder never touches.
const (
	inDir    = "in"
	outDir   = "out"
	clipsDir = "clips"
)

// sweep deletes every file under in/ and out/ whose name keep doesn't
// list, nor the .part of one it does.
func (h *Holder) sweep(ctx context.Context, keep []string) {
	disk, style := h.cfg.Disk, h.cfg.Disk.Host().Style
	for _, dir := range []string{inDir, outDir} {
		folder := style.Join(h.cfg.Scratch(), dir)
		entries, err := disk.ReadDir(ctx, folder)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			h.cfg.Log.Warn("list the scratch folder", "folder", folder, "err", err.Error())
			continue
		}
		for _, e := range entries {
			name := dir + "/" + e.Name
			whole, _ := strings.CutSuffix(name, partSuffix)
			if e.Dir || slices.Contains(keep, whole) {
				continue
			}
			if err := disk.Remove(ctx, style.Join(folder, e.Name)); err != nil && !errors.Is(err, fs.ErrNotExist) {
				h.cfg.Log.Warn("delete a scratch file no Conversion needs", "file", name, "err", err.Error())
			}
		}
	}
}
