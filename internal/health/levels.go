package health

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/jatassi/slipstream/internal/health/internal/db"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
)

var (
	StorageWarnBelow = setting.Space{
		Key:     "health.storageWarnBelow",
		Label:   "Warn when free space is below",
		Default: setting.Amount{N: 100 * setting.GB, Unit: setting.BytesUnit},
	}
	StorageErrorBelow = setting.Space{
		Key:     "health.storageErrorBelow",
		Label:   "Show an error when free space is below",
		Default: setting.Amount{N: 20 * setting.GB, Unit: setting.BytesUnit},
	}
)

// globalLevels is where a volume's levels default from.
const globalLevels = "Storage levels"

var ErrNoVolume = errors.New("no volume reports at that Node and mount")

// Pair is a warning and an error level, each nil where it follows the one
// above it: the default for the global pair, the global level for a
// volume's own.
type Pair struct{ Warn, Error *setting.Amount }

// PairPatch is a change to a Pair: a nil field leaves its level, the zero
// Stored resets it.
type PairPatch struct {
	Warn, Error *setting.Stored[setting.Amount]
}

func (p Pair) patched(pp PairPatch) Pair {
	if pp.Warn != nil {
		p.Warn = pp.Warn.Override()
	}
	if pp.Error != nil {
		p.Error = pp.Error.Override()
	}
	return p
}

// Resolved is one level resolved to bytes on one volume.
type Resolved struct {
	Amount setting.Amount
	Bytes  int64
	// Own: the volume's own level, not the global one.
	Own bool
}

// VolumeLevels is a volume's two levels, resolved.
type VolumeLevels struct{ Warn, Error Resolved }

// Judge is Fail below the error level, else Warn below the warning level,
// else None. Strictly below, with no hysteresis.
func (v VolumeLevels) Judge(free int64) Level {
	switch {
	case free < v.Error.Bytes:
		return Fail
	case free < v.Warn.Bytes:
		return Warn
	}
	return None
}

// SkipsWarning reports that the error level is at or above a warning level
// that can fire, so the volume goes straight to error, as mixed units can.
func (v VolumeLevels) SkipsWarning() bool {
	return v.Warn.Bytes > 0 && v.Error.Bytes >= v.Warn.Bytes
}

// resolve is the levels a volume follows: its own where set, else the
// global ones, else the defaults, each in bytes against total.
func resolve(global, own Pair, total int64) VolumeLevels {
	level := func(own, global *setting.Amount, def setting.Amount) Resolved {
		a := def
		switch {
		case own != nil:
			a = *own
		case global != nil:
			a = *global
		}
		return Resolved{Amount: a, Bytes: a.Bytes(total), Own: own != nil}
	}
	return VolumeLevels{
		Warn:  level(own.Warn, global.Warn, StorageWarnBelow.Default),
		Error: level(own.Error, global.Error, StorageErrorBelow.Default),
	}
}

// ordered refuses levels that share a unit and put the error above the
// warning. Mixed units can't be compared without a volume.
func ordered(v VolumeLevels) error {
	w, e := v.Warn.Amount, v.Error.Amount
	if w.Unit == e.Unit && e.N > w.N {
		return &setting.RangeError{Key: StorageErrorBelow.Key, Sentence: fmt.Sprintf("No more than the warning level, %s.", w.Words())}
	}
	return nil
}

type volumeAt struct {
	node  role.NodeID
	mount string
}

// Levels is every stored level, read once.
type Levels struct {
	global Pair
	own    map[volumeAt]Pair
}

// LevelsIn reads the levels in rd.
func LevelsIn(ctx context.Context, rd storage.Reader[file]) (Levels, error) {
	q := db.New(rd)
	s, err := q.GetSettings(ctx)
	if err != nil {
		return Levels{}, fmt.Errorf("read the storage levels: %w", err)
	}
	rows, err := q.ListVolumeLevels(ctx)
	if err != nil {
		return Levels{}, fmt.Errorf("read the volumes' own levels: %w", err)
	}
	l := Levels{global: pairOf(s.WarnAmount, s.WarnUnit, s.ErrorAmount, s.ErrorUnit), own: make(map[volumeAt]Pair, len(rows))}
	for _, r := range rows {
		l.own[volumeAt{r.NodeID, r.Mount}] = pairOf(r.WarnAmount, r.WarnUnit, r.ErrorAmount, r.ErrorUnit)
	}
	return l, nil
}

// For is the levels the volume of total bytes on node at mount follows.
func (l Levels) For(node role.NodeID, mount string, total int64) VolumeLevels {
	return resolve(l.global, l.own[volumeAt{node, mount}], total)
}

// fields is a volume's two levels as settings that default from the
// global ones.
func (l Levels) fields(node role.NodeID, mount string) (warn, errorBelow setting.SpaceField) {
	g := resolve(l.global, Pair{}, 0)
	own := l.own[volumeAt{node, mount}]
	w, e := StorageWarnBelow, StorageErrorBelow
	w.Default, w.DefaultFrom = g.Warn.Amount, globalLevels
	e.Default, e.DefaultFrom = g.Error.Amount, globalLevels
	return w.Field(own.Warn), e.Field(own.Error)
}

func pairOf(warnN *int64, warnUnit *setting.Unit, errN *int64, errUnit *setting.Unit) Pair {
	amount := func(n *int64, u *setting.Unit) *setting.Amount {
		if n == nil || u == nil {
			return nil
		}
		return &setting.Amount{N: *n, Unit: *u}
	}
	return Pair{Warn: amount(warnN, warnUnit), Error: amount(errN, errUnit)}
}

func columns(a *setting.Amount) (*int64, *setting.Unit) {
	if a == nil {
		return nil, nil
	}
	return &a.N, &a.Unit
}

// Levels is every stored level, kept since Open and through each levels
// Write, so judging volumes reads no storage.
func (h *Health) Levels() Levels { return *h.levels.Load() }

// writeLevels reads the levels inside a Write, lets change write and
// apply its change to them, and keeps them once the Write commits.
func (h *Health) writeLevels(ctx context.Context, change func(ctx context.Context, tx *storage.Tx[file], l *Levels) error) error {
	h.levelsMu.Lock()
	defer h.levelsMu.Unlock()
	var l Levels
	err := h.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		var err error
		if l, err = LevelsIn(ctx, tx); err != nil {
			return err
		}
		return change(ctx, tx, &l)
	})
	if err != nil {
		return err
	}
	h.levels.Store(&l)
	return nil
}

// GlobalLevels is the two global levels as settings.
func (h *Health) GlobalLevels() (warn, errorBelow setting.SpaceField) {
	l := h.Levels()
	return StorageWarnBelow.Field(l.global.Warn), StorageErrorBelow.Field(l.global.Error)
}

// SetGlobalLevels applies p unless it leaves the levels out of order, then
// evaluates Storage and publishes health.changed{storage}, so other tabs
// read the levels again even when no item moved.
func (h *Health) SetGlobalLevels(ctx context.Context, p PairPatch) error {
	err := h.writeLevels(ctx, func(ctx context.Context, tx *storage.Tx[file], l *Levels) error {
		l.global = l.global.patched(p)
		if err := ordered(resolve(l.global, Pair{}, 0)); err != nil {
			return err
		}
		warnN, warnUnit := columns(l.global.Warn)
		errN, errUnit := columns(l.global.Error)
		return db.New(tx).SetSettings(ctx, db.SetSettingsParams{WarnAmount: warnN, WarnUnit: warnUnit, ErrorAmount: errN, ErrorUnit: errUnit})
	})
	if err != nil {
		return fmt.Errorf("set the storage levels: %w", err)
	}
	return h.levelsChanged(ctx)
}

// ListedVolume is a volume a holder reports, with the levels it follows.
type ListedVolume struct {
	Volume
	Levels VolumeLevels
	// WarnBelow and ErrorBelow are IsDefault while they follow the global
	// levels.
	WarnBelow, ErrorBelow setting.SpaceField
	// Level is its Storage item's level, None when it has none.
	Level Level
}

// Override is a volume's own levels whose volume no holder reports.
type Override struct {
	Node     role.NodeID
	NodeName string
	Mount    string
	Own      Pair
}

type VolumeList struct {
	Volumes      []ListedVolume
	NotReporting []Override
}

// VolumeList is every volume any holder reports, with its levels and its
// item's level, and every override whose volume no longer reports.
func (h *Health) VolumeList(ctx context.Context) (VolumeList, error) {
	vs, err := h.volumes(ctx)
	if err != nil {
		return VolumeList{}, err
	}
	rep, err := h.Report(ctx)
	if err != nil {
		return VolumeList{}, err
	}
	l := h.Levels()
	var names map[role.NodeID]string
	err = h.cfg.Library.Read(ctx, func(ctx context.Context, rd storage.Reader[file]) error {
		var err error
		names, err = h.cfg.Names(ctx, rd)
		return err
	})
	if err != nil {
		return VolumeList{}, err
	}
	var out VolumeList
	reporting := map[volumeAt]bool{}
	for _, v := range vs.list {
		reporting[volumeAt{v.Node, v.Mount}] = true
		lv := ListedVolume{Volume: v, Levels: l.For(v.Node, v.Mount, v.Total)}
		lv.WarnBelow, lv.ErrorBelow = l.fields(v.Node, v.Mount)
		if it, ok := rep.item(StorageID, volumeKey(v.Node, v.Mount)); ok {
			lv.Level = it.Level
		}
		out.Volumes = append(out.Volumes, lv)
	}
	for at, own := range l.own {
		if !reporting[at] {
			out.NotReporting = append(out.NotReporting, Override{Node: at.node, NodeName: names[at.node], Mount: at.mount, Own: own})
		}
	}
	slices.SortFunc(out.NotReporting, func(a, b Override) int {
		return cmp.Or(a.Node.Compare(b.Node.UUID), cmp.Compare(a.Mount, b.Mount))
	})
	return out, nil
}

// SetVolumeLevels sets or resets a reporting volume's own levels unless
// that leaves its levels out of order. A row left with nothing of its own
// is deleted.
func (h *Health) SetVolumeLevels(ctx context.Context, node role.NodeID, mount string, p PairPatch) error {
	vs, err := h.volumes(ctx)
	if err != nil {
		return err
	}
	if _, ok := vs.byKey[volumeKey(node, mount)]; !ok {
		return ErrNoVolume
	}
	err = h.writeLevels(ctx, func(ctx context.Context, tx *storage.Tx[file], l *Levels) error {
		at := volumeAt{node, mount}
		own := l.own[at].patched(p)
		if err := ordered(resolve(l.global, own, 0)); err != nil {
			return err
		}
		q := db.New(tx)
		if own.Warn == nil && own.Error == nil {
			delete(l.own, at)
			return q.DeleteVolumeLevels(ctx, db.DeleteVolumeLevelsParams{NodeID: node, Mount: mount})
		}
		l.own[at] = own
		warnN, warnUnit := columns(own.Warn)
		errN, errUnit := columns(own.Error)
		return q.SetVolumeLevels(ctx, db.SetVolumeLevelsParams{
			NodeID: node, Mount: mount, WarnAmount: warnN, WarnUnit: warnUnit, ErrorAmount: errN, ErrorUnit: errUnit,
		})
	})
	if err != nil {
		return fmt.Errorf("set the volume's levels: %w", err)
	}
	return h.levelsChanged(ctx)
}

// RemoveVolumeLevels deletes a volume's own levels; removing none is no
// error.
func (h *Health) RemoveVolumeLevels(ctx context.Context, node role.NodeID, mount string) error {
	err := h.writeLevels(ctx, func(ctx context.Context, tx *storage.Tx[file], l *Levels) error {
		delete(l.own, volumeAt{node, mount})
		return db.New(tx).DeleteVolumeLevels(ctx, db.DeleteVolumeLevelsParams{NodeID: node, Mount: mount})
	})
	if err != nil {
		return fmt.Errorf("remove the volume's levels: %w", err)
	}
	return h.levelsChanged(ctx)
}

func (h *Health) levelsChanged(ctx context.Context) error {
	h.mayPrune.Store(true)
	_, err := h.ask(ctx, ask{ids: []ID{StorageID}, publish: true})
	return err
}
