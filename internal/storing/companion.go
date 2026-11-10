package storing

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"slices"
	"strings"

	"github.com/jatassi/slipstream/internal/media"
)

// companions names the companions of the file video among a folder's
// entries: a file that isn't a video and whose name is video's stem, ".",
// and a tail, unless a longer stem of another video in the folder claims
// it, so "Heat.1995.1080p.REMUX.en.srt" beside "Heat.mkv" and
// "Heat.1995.1080p.REMUX.mkv" is the second one's.
func companions(entries []Entry, video string) []string {
	stem := stemOf(video)
	var stems []string
	for _, e := range entries {
		if s := stemOf(e.Name); !e.Dir && isVideo(e.Name) && len(s) > len(stem) {
			stems = append(stems, s)
		}
	}
	var out []string
	for _, e := range entries {
		if e.Dir || e.Link || isVideo(e.Name) || transient(e.Name) || !strings.HasPrefix(e.Name, stem+".") {
			continue
		}
		if !slices.ContainsFunc(stems, func(s string) bool { return strings.HasPrefix(e.Name, s+".") }) {
			out = append(out, e.Name)
		}
	}
	return out
}

func stemOf(name string) string { return strings.TrimSuffix(name, path.Ext(name)) }

func isVideo(name string) bool {
	return slices.Contains(media.Extensions, strings.ToLower(strings.TrimPrefix(path.Ext(name), ".")))
}

// transient is a name a write leaves only midway: a copy not yet renamed
// into place, or a case-only rename not yet finished.
func transient(name string) bool {
	return strings.HasSuffix(name, ".part") || strings.HasSuffix(name, recaseSuffix)
}

// carry moves the companions of from, once from itself lies at to, each to
// to's stem with its own tail; from's folder gone holds none. It answers the companions that didn't take
// that name because another file holds it, where each now lies: in place,
// or, out of the holding folder, at "<to's stem>.<8 hex of the file
// id>.<tail>" beside to, since a companion left in holding would be swept
// unseen.
func (p placer) carry(ctx context.Context, from, to Rel) ([]Rel, error) {
	folder, video := from.split()
	into, name := to.split()
	entries, err := p.d.ReadDir(ctx, p.style.Under(p.root, folder))
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return nil, nil
	case err != nil:
		return nil, fmt.Errorf("list the file's folder: %w", err)
	}
	if Fold(folder.s) == Fold(into.s) {
		if err := p.finishRecases(ctx, into, entries, stemOf(name)); err != nil {
			return nil, err
		}
	}
	var left []Rel
	for _, c := range companions(entries, video) {
		tail := strings.TrimPrefix(c, stemOf(video)+".")
		src := folder.join(c)
		err := p.carryOne(ctx, src, into.join(stemOf(name)+"."+tail))
		if errors.Is(err, ErrOccupied) && folder.s == HoldingDir {
			beside := into.join(stemOf(name) + "." + shortID(video) + "." + tail)
			if err = p.carryOne(ctx, src, beside); err == nil {
				left = append(left, beside)
				continue
			}
		}
		switch {
		case errors.Is(err, ErrOccupied):
			left = append(left, src)
		case err != nil:
			return nil, err
		}
	}
	return left, nil
}

// carryOne renames the companion from to to. A file at to that is from
// spelled in another case takes to's spelling; any other is ErrOccupied.
func (p placer) carryOne(ctx context.Context, from, to Rel) error {
	src, dst := p.style.Under(p.root, from), p.style.Under(p.root, to)
	at, err := p.d.Stat(ctx, dst)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		if err := p.d.Rename(ctx, src, dst); err != nil {
			return fmt.Errorf("move a companion: %w", err)
		}
		return nil
	case err != nil:
		return fmt.Errorf("read a companion's target: %w", err)
	}
	was, err := p.d.Stat(ctx, src)
	if err != nil {
		return fmt.Errorf("read a companion: %w", err)
	}
	if !was.Same(at) || Fold(from.s) != Fold(to.s) {
		return ErrOccupied
	}
	return p.recase(ctx, src, dst)
}

// finishRecases finishes each case-only companion rename to stem a crash
// cut short, which left the companion at its temporary name.
func (p placer) finishRecases(ctx context.Context, folder Rel, entries []Entry, stem string) error {
	for _, e := range entries {
		name, ok := strings.CutSuffix(e.Name, recaseSuffix)
		if !ok || !strings.HasPrefix(name, stem+".") {
			continue
		}
		dst := p.style.Under(p.root, folder.join(name))
		if _, err := p.d.Stat(ctx, dst); !errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err := p.d.Rename(ctx, dst+recaseSuffix, dst); err != nil {
			return fmt.Errorf("finish renaming a companion: %w", err)
		}
	}
	return nil
}

// shortID is the first 8 hex digits of the id a holding name starts with.
func shortID(held string) string {
	id, _, _ := strings.Cut(held, ".")
	return strings.ReplaceAll(id, "-", "")[:8]
}
