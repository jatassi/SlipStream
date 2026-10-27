package grabs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/grabs/internal/db"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/searching"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage"
)

var keptLinkColumn = secret.NewColumn("grabs_release.link_sealed")

func keptLinkAD(run SearchID, position int) secret.AD {
	return keptLinkColumn.Row(run.String() + "/" + strconv.Itoa(position))
}

// keptRelease is a Release as grabs_release keeps it for a rerun: what the
// decision and a Grab read, with its link sealed apart.
type keptRelease struct {
	Priority       int               `json:"priority"`
	Title          string            `json:"title"`
	Size           int64             `json:"size"`
	Seeders        int               `json:"seeders"`
	InfoHash       string            `json:"infoHash,omitempty"`
	IDs            map[string]string `json:"ids,omitempty"`
	DownloadFactor float64           `json:"downloadFactor"`
	UploadFactor   float64           `json:"uploadFactor"`
	MinRatio       float64           `json:"minRatio,omitempty"`
	MinSeedSeconds int64             `json:"minSeedSeconds,omitempty"`
}

// keep stores c in run's kept Releases, at its position in the run.
func (gr *Grabs) keep(ctx context.Context, q *db.Queries, run SearchID, c candidate) error {
	r := c.Release
	k := keptRelease{
		Priority: r.Priority, Title: r.Title, Size: r.Size, Seeders: r.Seeders, IDs: r.IDs,
		DownloadFactor: r.Factors.Download, UploadFactor: r.Factors.Upload, MinRatio: r.Floor.MinRatio, MinSeedSeconds: int64(r.Floor.MinSeed.Seconds()),
	}
	if r.InfoHash != nil {
		k.InfoHash = r.InfoHash.String()
	}
	b, err := json.Marshal(k)
	if err != nil {
		return fmt.Errorf("encode a kept Release: %w", err)
	}
	return q.InsertRelease(ctx, db.InsertReleaseParams{
		SearchID: run, Position: int64(c.at), IndexerID: r.Indexer, Release: string(b),
		LinkSealed: gr.cfg.Sealer.Seal([]byte(r.Offered().Reveal()), keptLinkAD(run, c.at)),
	})
}

// kept is the Release row keeps, opened.
func (gr *Grabs) kept(row db.GrabsRelease) (indexers.Release, error) {
	var k keptRelease
	if err := json.Unmarshal([]byte(row.Release), &k); err != nil {
		return indexers.Release{}, fmt.Errorf("search %s: decode kept Release %d: %w", row.SearchID, row.Position, err)
	}
	raw, err := gr.cfg.Sealer.Open(row.LinkSealed, keptLinkAD(row.SearchID, int(row.Position)))
	if err != nil {
		return indexers.Release{}, fmt.Errorf("search %s: open kept Release %d's link: %w", row.SearchID, row.Position, err)
	}
	r := indexers.Release{Indexer: row.IndexerID, Priority: k.Priority, Release: searching.Release{
		Title: k.Title, Size: k.Size, Seeders: k.Seeders, IDs: k.IDs,
		Factors: searching.Factors{Download: k.DownloadFactor, Upload: k.UploadFactor},
		Floor:   searching.Floor{MinRatio: k.MinRatio, MinSeed: time.Duration(k.MinSeedSeconds) * time.Second},
	}}
	if k.InfoHash != "" {
		h, err := bencode.ParseInfoHash(k.InfoHash)
		if err != nil {
			return indexers.Release{}, fmt.Errorf("search %s: kept Release %d: %w", row.SearchID, row.Position, err)
		}
		r.InfoHash = &h
	}
	if link := searching.LinkOf(string(raw)); link.IsMagnet() {
		r.Magnet = link
	} else {
		r.Link = link
	}
	return r, nil
}

// dropped fails a Grab at fetching whose Indexer would make it wait for
// the Admin, and reruns its run's decision in the same Write.
type dropped struct {
	code Code
	msg  string
}

func (dropped) outcome() {}

// dropOf is the drop of a fetch its Indexer refused.
var dropOf = map[indexers.DropWhy]dropped{
	indexers.DropStopped:     {IndexerStopped, "The Indexer stopped before the .torrent was fetched"},
	indexers.DropHomeOffline: {HomeOffline, "The Indexer's home Node was offline past the grace period before the .torrent was fetched"},
	indexers.DropRemoved:     {IndexerRemoved, "The Indexer was removed before the .torrent was fetched"},
}

// left settles the run of g, a Grab leaving fetching by o: a drop reruns
// its decision, returning the rerun's Grab and run, and any other way out
// ends the run's kept Releases, since no rerun follows.
func (gr *Grabs) left(ctx context.Context, tx *storage.Tx[file], g Grab, o outcome) (*Grab, *Search, error) {
	q := db.New(tx)
	if _, drop := o.(dropped); !drop {
		return nil, nil, q.DeleteReleasesOfGrab(ctx, g.ID)
	}
	row, err := q.GetSearchOfGrab(ctx, g.ID)
	if errors.Is(err, storage.ErrNoRows) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	run, err := searchOf(row, nil, nil)
	if err != nil {
		return nil, nil, err
	}
	grab, err := gr.redecide(ctx, tx, run)
	return grab, &run, err
}

// redecide is run's decision again over the Releases it kept, which no
// Grab of the run took: the first pick its Indexer takes becomes the
// run's Grab, else the run found nothing acceptable. The run keeps its
// first decision's refusals.
func (gr *Grabs) redecide(ctx context.Context, tx *storage.Tx[file], run Search) (*Grab, error) {
	q := db.New(tx)
	rows, err := q.ListReleasesOfSearch(ctx, run.ID)
	if err != nil {
		return nil, err
	}
	releases := make([]indexers.Release, len(rows))
	for i, row := range rows {
		if releases[i], err = gr.kept(row); err != nil {
			return nil, err
		}
	}
	candidates, err := gr.candidates(ctx, tx, run, releases)
	if err != nil {
		return nil, err
	}
	for i, row := range rows {
		candidates[i].at = int(row.Position)
	}
	tree, err := gr.cfg.Works.TreeIn(ctx, tx, run.WorkID)
	if err != nil {
		return nil, err
	}
	terms, err := gr.cfg.Terms[tree.ContentType].SearchTerms(ctx, tx, run.ItemID)
	if err != nil {
		return nil, err
	}
	picks, _, err := gr.decision(ctx, tx, run, terms, dedupe(candidates))
	if err != nil {
		return nil, err
	}
	grab, pick, err := gr.take(ctx, tx, run, picks, storage.Now())
	if err != nil {
		return nil, err
	}
	if grab == nil {
		if err := q.DeleteReleasesOfSearch(ctx, run.ID); err != nil {
			return nil, err
		}
		return nil, q.Redecide(ctx, db.RedecideParams{ID: run.ID, Outcome: new(NothingAcceptable)})
	}
	notes, err := encodeReasons(pick.Notes)
	if err != nil {
		return nil, err
	}
	if err := q.DeleteRelease(ctx, db.DeleteReleaseParams{SearchID: run.ID, Position: int64(pick.at)}); err != nil {
		return nil, err
	}
	return grab, q.Redecide(ctx, db.RedecideParams{ID: run.ID, Outcome: new(Grabbed), GrabID: grab.ID, Notes: &notes})
}
