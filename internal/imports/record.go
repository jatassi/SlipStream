package imports

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"slices"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/imports/internal/db"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/works"
)

// importOf reads the Import id with its files, ErrNoImport when none has
// it.
func importOf(ctx context.Context, r storage.Reader[file], id ImportID) (Import, error) {
	q := db.New(r)
	row, err := q.GetImport(ctx, id)
	if errors.Is(err, storage.ErrNoRows) {
		return Import{}, ErrNoImport
	}
	if err != nil {
		return Import{}, err
	}
	fs, err := q.ListFilesOf(ctx, id)
	if err != nil {
		return Import{}, err
	}
	items, err := q.ListFileItemsOf(ctx, id)
	if err != nil {
		return Import{}, err
	}
	fills, err := q.ListFillsOf(ctx, id)
	if err != nil {
		return Import{}, err
	}
	return recordOf(row, fs, items, fills)
}

// rootImportAt reads the unfinished root Import of root's folder,
// storage.ErrNoRows when there is none.
func rootImportAt(ctx context.Context, r storage.Reader[file], p db.GetRootImportAtParams) (Import, error) {
	row, err := db.New(r).GetRootImportAt(ctx, p)
	if err != nil {
		return Import{}, err
	}
	return importOf(ctx, r, row.ID)
}

// allImports is every Import rows lists, with its files.
func allImports(ctx context.Context, r storage.Reader[file], rows []db.ImportsImport) ([]Import, error) {
	q := db.New(r)
	fileRows, err := q.ListFiles(ctx)
	if err != nil {
		return nil, err
	}
	itemRows, err := q.ListFileItems(ctx)
	if err != nil {
		return nil, err
	}
	fillRows, err := q.ListFills(ctx)
	if err != nil {
		return nil, err
	}
	fills := map[ImportID][]db.ImportsFileFill{}
	for _, f := range fillRows {
		fills[f.ImportID] = append(fills[f.ImportID], f)
	}
	fs := map[ImportID][]db.ImportsFile{}
	for _, f := range fileRows {
		fs[f.ImportID] = append(fs[f.ImportID], f)
	}
	items := map[ImportID][]db.ImportsFileItem{}
	for _, it := range itemRows {
		items[it.ImportID] = append(items[it.ImportID], it)
	}
	out := make([]Import, len(rows))
	for i, row := range rows {
		if out[i], err = recordOf(row, fs[row.ID], items[row.ID], fills[row.ID]); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func recordOf(row db.ImportsImport, fs []db.ImportsFile, items []db.ImportsFileItem, fills []db.ImportsFileFill) (Import, error) {
	im := Import{
		ID: row.ID, Source: Source(row.Source), Grab: row.GrabID, Node: row.NodeID, Root: row.RootID, ContentType: row.ContentType,
		Work: row.WorkID, Edition: row.Edition, MoveFiles: row.MoveFiles == 1, Created: row.CreatedAt, Ignored: row.IgnoredAt,
		Finished: row.FinishedAt,
	}
	if row.Folder != nil {
		im.Folder = *row.Folder
	}
	if row.InfoHash != nil {
		h := bencode.InfoHash(row.InfoHash)
		im.InfoHash = &h
	}
	if row.FailureCode != nil {
		im.Failure = new(FailureCode(*row.FailureCode))
	}
	if err := json.Unmarshal([]byte(row.Hint), &im.Hint); err != nil {
		return Import{}, fmt.Errorf("import %s: its hint: %w", row.ID, err)
	}
	var err error
	if im.Suggestion, err = fromJSON[Suggestion](row.Suggestion); err != nil {
		return Import{}, fmt.Errorf("import %s: its suggestion: %w", row.ID, err)
	}
	if im.Ended, err = fromJSON[Ending](row.Ended); err != nil {
		return Import{}, fmt.Errorf("import %s: how it ended: %w", row.ID, err)
	}
	of := map[int64][]works.ItemID{}
	for _, it := range items {
		of[it.Position] = append(of[it.Position], it.ItemID)
	}
	filled := map[int64][]Fill{}
	for _, f := range fills {
		filled[f.Position] = append(filled[f.Position], Fill{Cell: files.CellKey{Item: f.ItemID, Edition: f.Edition, Target: f.TargetID}, Forced: f.Forced == 1})
	}
	im.Files = make([]File, len(fs))
	for i, f := range fs {
		if im.Files[i], err = fileOf(f, of[f.Position]); err != nil {
			return Import{}, fmt.Errorf("import %s: %w", row.ID, err)
		}
		im.Files[i].Fills = filled[f.Position]
	}
	return im, nil
}

func fileOf(row db.ImportsFile, items []works.ItemID) (File, error) {
	f := File{
		Position: int(row.Position), Path: row.Path, Size: row.Size, Probe: row.Probe, Cut: row.Cut, ReleaseName: row.ReleaseName,
		SourceName: row.SourceName, Items: items, Outcome: Outcome(row.Outcome), Confirmed: row.Confirmed == 1, FileID: row.FileID,
		SourceRemoved: row.SourceRemovedAt, Seeded: row.Seeded == 1,
	}
	if row.SourceIndex != nil {
		f.SourceIndex = new(int(*row.SourceIndex))
	}
	if row.Quality != nil {
		f.Quality = *row.Quality
	}
	if !row.ModifiedAt.IsZero() {
		f.Seen = &storing.Seen{Modified: row.ModifiedAt.Time()}
		if row.NodeFileID != nil {
			f.Seen.ID = storing.FileID{Volume: *row.NodeVolumeID, File: *row.NodeFileID}
		}
	}
	var err error
	if f.Reason, err = fromJSON[Reason](row.Reason); err != nil {
		return File{}, fmt.Errorf("the reason of %s: %w", row.Path, err)
	}
	if f.Hold, err = fromJSON[Hold](row.Hold); err != nil {
		return File{}, fmt.Errorf("the hold of %s: %w", row.Path, err)
	}
	if row.Dest != nil {
		holding, err := storing.ParseRel(*row.Dest)
		if err != nil {
			return File{}, fmt.Errorf("the destination of %s: %w", row.Path, err)
		}
		f.Dest = &Dest{Root: row.RootID, Holding: holding}
	}
	if row.SentOrder != nil {
		f.Sent = &Sent{Transfer: row.SentTransfer, Order: int(*row.SentOrder)}
	}
	if row.PlacedBy != nil {
		f.PlacedBy = PlacedBy(*row.PlacedBy)
	}
	return f, nil
}

func fileRow(id ImportID, f File) (db.ImportsFile, error) {
	row := db.ImportsFile{
		ImportID: id, Position: int64(f.Position), Path: f.Path, Size: f.Size, Probe: f.Probe, Cut: f.Cut, ReleaseName: f.ReleaseName,
		SourceName: f.SourceName, Outcome: string(f.Outcome), FileID: f.FileID, SourceRemovedAt: f.SourceRemoved,
	}
	if f.SourceIndex != nil {
		row.SourceIndex = new(int64(*f.SourceIndex))
	}
	if f.Probe != nil {
		row.Quality = &f.Quality
	}
	if f.Seen != nil {
		row.ModifiedAt = storage.FromTime(f.Seen.Modified)
		if !f.Seen.ID.IsZero() {
			row.NodeFileID, row.NodeVolumeID = &f.Seen.ID.File, &f.Seen.ID.Volume
		}
	}
	if f.Confirmed {
		row.Confirmed = 1
	}
	if f.Seeded {
		row.Seeded = 1
	}
	var err error
	if row.Reason, err = jsonOf(f.Reason); err != nil {
		return db.ImportsFile{}, fmt.Errorf("the reason of %s: %w", f.Path, err)
	}
	if row.Hold, err = jsonOf(f.Hold); err != nil {
		return db.ImportsFile{}, fmt.Errorf("the hold of %s: %w", f.Path, err)
	}
	if f.Dest != nil {
		row.RootID, row.Dest = f.Dest.Root, new(f.Dest.Holding.String())
	}
	if f.Sent != nil {
		row.SentTransfer, row.SentOrder = f.Sent.Transfer, new(int64(f.Sent.Order))
	}
	if f.PlacedBy != "" {
		row.PlacedBy = new(string(f.PlacedBy))
	}
	return row, nil
}

func insertFile(ctx context.Context, q *db.Queries, id ImportID, f File) error {
	row, err := fileRow(id, f)
	if err != nil {
		return err
	}
	err = q.InsertFile(ctx, db.InsertFileParams{
		ImportID: id, Position: row.Position, Path: row.Path, SourceIndex: row.SourceIndex, Size: row.Size, ModifiedAt: row.ModifiedAt,
		NodeFileID: row.NodeFileID, NodeVolumeID: row.NodeVolumeID, Probe: row.Probe, Quality: row.Quality, Cut: row.Cut,
		ReleaseName: row.ReleaseName, SourceName: row.SourceName, Outcome: row.Outcome, Reason: row.Reason, Hold: row.Hold,
		Confirmed: row.Confirmed, RootID: row.RootID, Dest: row.Dest, SentTransfer: row.SentTransfer, SentOrder: row.SentOrder,
		PlacedBy: row.PlacedBy, FileID: row.FileID, SourceRemovedAt: row.SourceRemovedAt, Seeded: row.Seeded,
	})
	if err != nil {
		return err
	}
	if err := insertFills(ctx, q, id, f); err != nil {
		return err
	}
	return insertItems(ctx, q, id, f)
}

func insertFills(ctx context.Context, q *db.Queries, id ImportID, f File) error {
	for _, fl := range f.Fills {
		p := db.InsertFillParams{ImportID: id, Position: int64(f.Position), ItemID: fl.Cell.Item, Edition: fl.Cell.Edition, TargetID: fl.Cell.Target}
		if fl.Forced {
			p.Forced = 1
		}
		if err := q.InsertFill(ctx, p); err != nil {
			return err
		}
	}
	return nil
}

func insertItems(ctx context.Context, q *db.Queries, id ImportID, f File) error {
	for i, it := range f.Items {
		if err := q.InsertFileItem(ctx, db.InsertFileItemParams{ImportID: id, Position: int64(f.Position), ItemID: it, Order: int64(i)}); err != nil {
			return err
		}
	}
	return nil
}

// insertImport stores a new Import and its files.
func insertImport(ctx context.Context, tx *storage.Tx[file], im Import) error {
	q := db.New(tx)
	p := db.InsertImportParams{
		ID: im.ID, Source: string(im.Source), GrabID: im.Grab, NodeID: im.Node, RootID: im.Root, ContentType: im.ContentType,
		WorkID: im.Work, Edition: im.Edition, CreatedAt: im.Created, IgnoredAt: im.Ignored, FinishedAt: im.Finished,
	}
	if im.Folder != "" {
		p.Folder = &im.Folder
	}
	if im.InfoHash != nil {
		p.InfoHash = im.InfoHash[:]
	}
	if im.MoveFiles {
		p.MoveFiles = 1
	}
	var err error
	if p.Hint, p.Suggestion, p.FailureCode, p.Ended, err = columnsOf(im); err != nil {
		return err
	}
	if err := q.InsertImport(ctx, p); err != nil {
		return err
	}
	for _, f := range im.Files {
		if err := insertFile(ctx, q, im.ID, f); err != nil {
			return err
		}
	}
	return nil
}

func columnsOf(im Import) (hint string, suggestion, failure, ended *string, err error) {
	b, err := json.Marshal(im.Hint)
	if err != nil {
		return "", nil, nil, nil, fmt.Errorf("import %s: its hint: %w", im.ID, err)
	}
	if suggestion, err = jsonOf(im.Suggestion); err != nil {
		return "", nil, nil, nil, fmt.Errorf("import %s: its suggestion: %w", im.ID, err)
	}
	if ended, err = jsonOf(im.Ended); err != nil {
		return "", nil, nil, nil, fmt.Errorf("import %s: how it ended: %w", im.ID, err)
	}
	if im.Failure != nil {
		failure = new(string(*im.Failure))
	}
	return string(b), suggestion, failure, ended, nil
}

// save writes im over what tx holds for its id: each file through setFile
// from its stored outcome, a new position inserted, a position im no
// longer lists deleted. A root or folder Import left with no files is
// deleted, and a root Import whose files are all final is finished.
func save(ctx context.Context, tx *storage.Tx[file], im *Import) error {
	q := db.New(tx)
	stored, err := importOf(ctx, tx, im.ID)
	if err != nil {
		return err
	}
	if im.Source != FromGrab && len(im.Files) == 0 {
		return q.DeleteImport(ctx, im.ID)
	}
	if im.Source == FromRoot && im.Finished.IsZero() && !slices.ContainsFunc(im.Files, func(f File) bool { return !f.Outcome.Final() }) {
		im.Finished = storage.Now()
	}
	if err := saveFiles(ctx, q, im.ID, stored.Files, im.Files); err != nil {
		return err
	}
	p := db.UpdateImportParams{ID: im.ID, WorkID: im.Work, Edition: im.Edition, IgnoredAt: im.Ignored, FinishedAt: im.Finished}
	if im.MoveFiles {
		p.MoveFiles = 1
	}
	if p.Hint, p.Suggestion, p.FailureCode, p.Ended, err = columnsOf(*im); err != nil {
		return err
	}
	return q.UpdateImport(ctx, p)
}

// saveFiles writes fs over stored, an Import's files as tx holds them.
func saveFiles(ctx context.Context, q *db.Queries, id ImportID, stored, fs []File) error {
	at := map[int]File{}
	for _, f := range stored {
		at[f.Position] = f
	}
	for _, f := range fs {
		was, ok := at[f.Position]
		delete(at, f.Position)
		if !ok {
			if err := insertFile(ctx, q, id, f); err != nil {
				return err
			}
			continue
		}
		if err := setFile(ctx, q, id, was.Outcome, f); err != nil {
			return err
		}
		if err := saveLinks(ctx, q, id, was, f); err != nil {
			return err
		}
	}
	for _, f := range at {
		if err := q.DeleteFile(ctx, db.DeleteFileParams{ImportID: id, Position: int64(f.Position)}); err != nil {
			return err
		}
	}
	return nil
}

// appendWaiting appends fs to im at fresh positions, waiting with the
// reasons they carry, named by their base names.
func appendWaiting(im *Import, fs ...File) {
	next := 0
	for _, f := range im.Files {
		next = max(next, f.Position+1)
	}
	for _, f := range fs {
		f.Position, f.Outcome = next, Waiting
		f.ReleaseName = path.Base(f.Path)
		f.SourceName = f.ReleaseName
		im.Files = append(im.Files, f)
		next++
	}
}

// waiting is im's files that wait.
func (im Import) waiting() []File {
	var out []File
	for _, f := range im.Files {
		if f.Outcome == Waiting {
			out = append(out, f)
		}
	}
	return out
}

// saveLinks rewrites f's fills and Items where they changed from was.
func saveLinks(ctx context.Context, q *db.Queries, id ImportID, was, f File) error {
	at := int64(f.Position)
	if !slices.Equal(was.Fills, f.Fills) {
		if err := q.DeleteFills(ctx, db.DeleteFillsParams{ImportID: id, Position: at}); err != nil {
			return err
		}
		if err := insertFills(ctx, q, id, f); err != nil {
			return err
		}
	}
	if slices.Equal(was.Items, f.Items) {
		return nil
	}
	if err := q.DeleteFileItems(ctx, db.DeleteFileItemsParams{ImportID: id, Position: at}); err != nil {
		return err
	}
	return insertItems(ctx, q, id, f)
}
