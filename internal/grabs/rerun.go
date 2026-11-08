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
	Source         *keptSource       `json:"source,omitempty"`
	Flags          []searching.Flag  `json:"flags,omitempty"`
}

// keptSource is the site a Torznab feed said the Release came from, which
// picks its Tracker when a rerun takes it.
type keptSource struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Privacy string `json:"privacy"`
	Origin  string `json:"origin"`
}

// encodeKept is r as grabs_release keeps it for a rerun, its link apart.
func encodeKept(r indexers.Release) (string, error) {
	k := keptRelease{
		Priority: r.Priority, Title: r.Title, Size: r.Size, Seeders: r.Seeders, IDs: r.IDs,
		DownloadFactor: r.Factors.Download, UploadFactor: r.Factors.Upload, MinRatio: r.Floor.MinRatio, MinSeedSeconds: int64(r.Floor.MinSeed.Seconds()),
		Flags: r.Flags,
	}
	if r.InfoHash != nil {
		k.InfoHash = r.InfoHash.String()
	}
	if s := r.Source; s != nil {
		k.Source = &keptSource{ID: s.ID, Name: s.Name, Privacy: s.Privacy, Origin: s.Origin}
	}
	b, err := json.Marshal(k)
	if err != nil {
		return "", fmt.Errorf("encode a kept Release: %w", err)
	}
	return string(b), nil
}

// kept is the Release row keeps, opened, with no link when it is gone.
func (gr *Grabs) kept(row db.GrabsRelease) (indexers.Release, error) {
	var k keptRelease
	if err := json.Unmarshal([]byte(*row.Release), &k); err != nil {
		return indexers.Release{}, fmt.Errorf("search %s: decode kept Release %d: %w", row.SearchID, row.Position, err)
	}
	r := indexers.Release{Indexer: row.IndexerID, Priority: k.Priority, Release: searching.Release{
		Title: k.Title, Size: k.Size, Seeders: k.Seeders, IDs: k.IDs,
		Factors: searching.Factors{Download: k.DownloadFactor, Upload: k.UploadFactor},
		Floor:   searching.Floor{MinRatio: k.MinRatio, MinSeed: time.Duration(k.MinSeedSeconds) * time.Second},
		Flags:   k.Flags,
	}}
	if k.InfoHash != "" {
		h, err := bencode.ParseInfoHash(k.InfoHash)
		if err != nil {
			return indexers.Release{}, fmt.Errorf("search %s: kept Release %d: %w", row.SearchID, row.Position, err)
		}
		r.InfoHash = &h
	}
	if s := k.Source; s != nil {
		r.Source = &searching.Source{ID: s.ID, Name: s.Name, Privacy: s.Privacy, Origin: s.Origin}
	}
	if !row.LinkSealed.Set() {
		return r, nil
	}
	raw, err := gr.cfg.Sealer.Open(row.LinkSealed, keptLinkAD(row.SearchID, int(row.Position)))
	if err != nil {
		return indexers.Release{}, fmt.Errorf("search %s: open kept Release %d's link: %w", row.SearchID, row.Position, err)
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

// redo is what a dropped Grab's rerun reads before its Write: the run,
// the Releases it kept, assessed, the positions of those gone, taken by a
// Grab of the run, and each one's Indexer's name by position.
type redo struct {
	run      Search
	assessed []assessed
	gone     map[int]bool
	names    map[int]string
}

// rerunOf reads, in one Read, the rerun of g's run, nil when no run
// holds g.
func (gr *Grabs) rerunOf(ctx context.Context, g Grab) (*redo, error) {
	var out *redo
	err := gr.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		q := db.New(r)
		row, err := q.GetSearchOfGrab(ctx, g.ID)
		if errors.Is(err, storage.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		run, err := searchOf(row, nil)
		if err != nil {
			return err
		}
		rows, err := q.ListKeptReleasesOfSearch(ctx, run.ID)
		if err != nil {
			return err
		}
		ls := make([]listed, len(rows))
		gone, names := map[int]bool{}, map[int]string{}
		for i, row := range rows {
			if ls[i].Release, err = gr.kept(row); err != nil {
				return err
			}
			ls[i].at = int(row.Position)
			gone[ls[i].at], names[ls[i].at] = !row.LinkSealed.Set(), row.IndexerName
		}
		tree, err := gr.cfg.Works.TreeIn(ctx, r, run.WorkID)
		if err != nil {
			return err
		}
		terms, err := gr.cfg.Terms[tree.ContentType].SearchTerms(ctx, r, run.ItemID)
		if err != nil {
			return err
		}
		j, err := gr.judgingOf(ctx, r, tree.ContentType, run, terms.Expect)
		if err != nil {
			return err
		}
		out = &redo{run: run, assessed: assessAll(j, dedupe(ls)), gone: gone, names: names}
		return nil
	})
	return out, err
}

// left settles the run of g, a Grab leaving fetching by o: a drop reruns
// its decision over re, returning the rerun's Grab and run, and any other
// way out ends what the run's Releases keep for a rerun, since none
// follows.
func (gr *Grabs) left(ctx context.Context, tx *storage.Tx[file], g Grab, o outcome, re *redo) (*Grab, *Search, error) {
	if _, drop := o.(dropped); !drop {
		return nil, nil, db.New(tx).EndReleasesOfGrab(ctx, g.ID)
	}
	if re == nil {
		return nil, nil, nil
	}
	grab, err := gr.redecide(ctx, tx, *re)
	return grab, &re.run, err
}

// redecide is re's run's decision again over the Releases it kept: the
// first its Indexer takes, of those no Grab of the run took, becomes the
// run's Grab, else the run found nothing acceptable. The run's Releases
// are stored again with their new outcomes.
func (gr *Grabs) redecide(ctx context.Context, tx *storage.Tx[file], re redo) (*Grab, error) {
	q := db.New(tx)
	run := re.run
	d, c, err := gr.decision(ctx, tx, run, re.assessed)
	if err != nil {
		return nil, err
	}
	grab, pick, err := gr.take(ctx, tx, run, d.picks(c), re.gone, storage.Now())
	if err != nil {
		return nil, err
	}
	if err := q.DeleteReleasesOfSearch(ctx, run.ID); err != nil {
		return nil, err
	}
	if err := gr.storeReleases(ctx, q, run.ID, outcomes(d, c, pick.at), re.names, grab != nil, re.gone); err != nil {
		return nil, err
	}
	if grab == nil {
		return nil, q.Redecide(ctx, db.RedecideParams{ID: run.ID, Outcome: new(NothingAcceptable)})
	}
	notes, err := encodeReasons(pick.Notes)
	if err != nil {
		return nil, err
	}
	return grab, q.Redecide(ctx, db.RedecideParams{ID: run.ID, Outcome: new(Grabbed), GrabID: grab.ID, Notes: &notes})
}
