package files

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"slices"

	"github.com/jatassi/slipstream/internal/files/internal/db"
	"github.com/jatassi/slipstream/internal/naming"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// Block is why a row can't rename now.
type Block uint8

const (
	// Occupied is a file SlipStream doesn't track at To.
	Occupied Block = iota + 1
	// TooLong is a path that doesn't fit after every title is cut.
	TooLong
	// Clash is another file that ends at To: another Work's, or with
	// renaming off a file of its own Work that keeps the same name.
	Clash
)

// Blocked is why a row can't rename now. The zero Blocked is none.
type Blocked struct {
	Reason Block
	// Other is the Work whose file a Clash meets.
	Other works.WorkID
	// Long is a TooLong row's lengths.
	Long *naming.TooLongError
}

// Row is one live file whose path from the templates differs from its
// stored one.
type Row struct {
	Work works.WorkID
	File ID
	// From is where the file belongs now, and To where the templates put
	// it, "" when it doesn't fit.
	From string
	To   string
	// Blocked is why it can't rename now.
	Blocked Blocked
	// Group is 0 for a row joined to no other, else the same number, from
	// 1, for every row of its Work joined by one's To being another's
	// From without case: rows that trade or chain paths, ticked together.
	Group int
	// Companions are the files beside From that move with it, such as
	// its subtitles: only a one-Work plan lists them.
	Companions []string
	root       roots.ID
}

// WorkPlan is one Work's rows, in file id order.
type WorkPlan struct {
	Work  works.WorkID
	Type  works.TypeID
	Title string
	Rows  []Row
	// Moving counts its files the disk queue still moves, which no row
	// lists.
	Moving int
	Digest Digest
}

// Digest is the first 8 bytes of SHA-256 over a Work's rows in file id
// order: each row's file, From, To, Blocked.Reason, Blocked.Other and
// Group. A confirm renames a Work only while its digest is the one shown.
type Digest [8]byte

func (d Digest) String() string { return hex.EncodeToString(d[:]) }

// ParseDigest reads a Digest from its String.
func ParseDigest(s string) (Digest, error) {
	var d Digest
	if len(s) != hex.EncodedLen(len(d)) {
		return Digest{}, fmt.Errorf("the digest %q isn't 16 hexadecimal characters", s)
	}
	if _, err := hex.Decode(d[:], []byte(s)); err != nil {
		return Digest{}, fmt.Errorf("the digest %q isn't 16 hexadecimal characters", s)
	}
	return d, nil
}

func digest(rows []Row) Digest {
	var b []byte
	for _, r := range rows {
		b = append(b, r.File.Bytes()...)
		b = binary.AppendUvarint(b, uint64(len(r.From)))
		b = append(b, r.From...)
		b = binary.AppendUvarint(b, uint64(len(r.To)))
		b = append(b, r.To...)
		b = append(b, byte(r.Blocked.Reason))
		b = append(b, r.Blocked.Other.Bytes()...)
		b = binary.AppendVarint(b, int64(r.Group))
	}
	sum := sha256.Sum256(b)
	return Digest(sum[:8])
}

// plans is each Work of trees's plan under its Content Type's stored
// scheme, or under draft for draft's type, before any Node is asked, read
// in r with the values of each type's Works in one read, with the world
// and names each was made from. among names trees' Works.
func (f *Files) plans(ctx context.Context, r storage.Reader[file], trees []works.Bare, among works.Among[works.WorkID], draft *Draft) ([]worldPlan, error) {
	var scheme *naming.Scheme
	if draft != nil {
		var err error
		if scheme, err = f.cfg.Naming.DraftIn(ctx, r, draft.Type, draft.Patch); err != nil {
			return nil, err
		}
	}
	ws, err := f.worlds(ctx, r, trees, among, wholeFiles)
	if err != nil {
		return nil, err
	}
	of := make([]*world, len(ws))
	for i := range ws {
		of[i] = &ws[i]
	}
	if err := f.withNaming(ctx, r, of, among); err != nil {
		return nil, err
	}
	moving, err := movingIn(ctx, db.New(r), among)
	if err != nil {
		return nil, err
	}
	out := make([]worldPlan, len(ws))
	for i, w := range ws {
		w.moving = moving
		if draft != nil && w.tree.ContentType == draft.Type {
			w.scheme = scheme
		}
		out[i].w = w
		if out[i].WorkPlan, out[i].names, err = plan(w); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// worldPlan is a Work's plan with the world and the names it was made from.
type worldPlan struct {
	WorkPlan
	w     world
	names map[ID]name
}

// movingIn is the files of among's Works the disk queue still moves.
func movingIn(ctx context.Context, q *db.Queries, among works.Among[works.WorkID]) (map[ID]bool, error) {
	ids, err := works.Read(among, func() ([]ID, error) { return q.ListMovingFiles(ctx) }, func(ids []works.WorkID) ([]ID, error) {
		return q.ListMovingFilesOfWorks(ctx, ids)
	})
	out := make(map[ID]bool, len(ids))
	for _, id := range ids {
		out[id] = true
	}
	return out, err
}

// plan is w's rows under its stored fills: every live file names puts
// somewhere else in everyFile mode, blocked as names blocks it, grouped.
// A file the disk queue still moves is counted, not listed.
func plan(w world) (WorkPlan, map[ID]name, error) {
	w.mode = everyFile
	nm, blocked, err := names(w, w.fills)
	if err != nil {
		return WorkPlan{}, nil, err
	}
	p := WorkPlan{Work: w.tree.ID, Type: w.tree.ContentType, Title: w.tree.Label}
	for _, f := range w.files {
		switch b, ok := blocked[f.ID]; {
		case f.Gone():
		case w.moving[f.ID]:
			p.Moving++
		case ok || nm[f.ID].Path != f.Path:
			p.Rows = append(p.Rows, Row{Work: w.tree.ID, File: f.ID, From: f.Path, To: nm[f.ID].Path, Blocked: b, root: f.RootID})
		}
	}
	group(p.Rows)
	return p, nm, nil
}

// group numbers each set of rows joined by one's To being another's From
// without case, in the order of each set's first row.
func group(rows []Row) {
	from := make(map[place]int, len(rows))
	for i, r := range rows {
		from[folded(r.root, r.From)] = i
	}
	up := make([]int, len(rows))
	for i := range up {
		up[i] = i
	}
	top := func(i int) int {
		for up[i] != i {
			up[i], i = up[up[i]], up[i]
		}
		return i
	}
	joined := make([]bool, len(rows))
	for i, r := range rows {
		if j, ok := from[folded(r.root, r.To)]; ok && r.To != "" && j != i {
			up[top(i)] = top(j)
			joined[i], joined[j] = true, true
		}
	}
	number := map[int]int{}
	for i := range rows {
		if !joined[i] {
			continue
		}
		n, ok := number[top(i)]
		if !ok {
			n = len(number) + 1
			number[top(i)] = n
		}
		rows[i].Group = n
	}
}

// settle blocks each row of ps that can't rename now, then seals each
// plan's digest. tracked is each place SlipStream tracks a file in, or a
// Work outside ps renames a file to, and those Works; found is what the
// Node found at each row's To, by the row's file: the paths whose names
// are To's without case.
//
// A row is a Clash when its To is, without case, another Work's tracked
// file or another Work's row's To, naming the lowest such Work. It
// is Occupied when the Node found a file there that isn't a From of its
// own Work's rows: its own source in another case is a case-only rename,
// and another row's From is vacated.
func settle(ps []WorkPlan, tracked map[place][]works.WorkID, found map[ID][]string) {
	owners := map[place][]works.WorkID{}
	for at, ws := range tracked {
		owners[folded(at.root, at.path)] = append(owners[folded(at.root, at.path)], ws...)
	}
	vacated := map[place]works.WorkID{}
	for _, p := range ps {
		for _, r := range p.Rows {
			vacated[place{r.root, r.From}] = p.Work
			if r.Blocked == (Blocked{}) {
				owners[folded(r.root, r.To)] = append(owners[folded(r.root, r.To)], p.Work)
			}
		}
	}
	for i := range ps {
		p := &ps[i]
		for j := range p.Rows {
			r := &p.Rows[j]
			if r.Blocked != (Blocked{}) {
				continue
			}
			others := slices.DeleteFunc(slices.Clone(owners[folded(r.root, r.To)]), func(w works.WorkID) bool { return w == p.Work })
			if len(others) > 0 {
				r.Blocked = Blocked{Reason: Clash, Other: slices.MinFunc(others, func(a, b works.WorkID) int { return a.Compare(b.UUID) })}
				continue
			}
			if slices.ContainsFunc(found[r.File], func(at string) bool { return vacated[place{r.root, at}] != p.Work }) {
				r.Blocked = Blocked{Reason: Occupied}
			}
		}
		p.Digest = digest(p.Rows)
	}
}
