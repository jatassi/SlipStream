package mstest

import (
	"context"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/mediaservers"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// Holds records fs as work's files whose bytes sit at their paths, in
// place of the files it held before.
func (w *World) Holds(work works.WorkID, fs ...files.File) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.files[work] = fs
}

func (w *World) workFiles(_ context.Context, _ storage.Reader[storage.Library], work works.WorkID) ([]files.File, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return append([]files.File(nil), w.files[work]...), nil
}

// Found is every media-servers.items-found published so far.
func (w *World) Found() []mediaservers.Found {
	w.mu.Lock()
	defer w.mu.Unlock()
	return append([]mediaservers.Found(nil), w.found...)
}

type foundEvents struct{ w *World }

func (e foundEvents) Publish(_ context.Context, v mediaservers.Found) {
	e.w.mu.Lock()
	defer e.w.mu.Unlock()
	e.w.found = append(e.w.found, v)
}
