package grabsapi

import (
	"context"
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/identity"
)

type workRef struct {
	ID    string `json:"id" format:"uuid"`
	Title string `json:"title"`
}

type blockEntry struct {
	InfoHash string                `json:"infoHash" pattern:"^[0-9a-f]{40}$"`
	Title    string                `json:"title"`
	Words    string                `json:"words" doc:"What SlipStream reads from the title, in words"`
	Cause    string                `json:"cause" enum:"unregistered,no_payload,engine_error,marked_failed,by_hand"`
	Because  string                `json:"because" doc:"The cause in a sentence, such as TorrentLeech reported it unregistered."`
	Indexer  *string               `json:"indexer" doc:"The name of the Indexer that listed it; null when none is known"`
	Work     api.Nullable[workRef] `json:"work" doc:"The Work it was grabbed for; null once the Work is deleted"`
	AddedAt  time.Time             `json:"addedAt"`
}

type blocklistInput struct {
	Offset int    `query:"offset" minimum:"0" doc:"How many of the newest to skip"`
	Q      string `query:"q" maxLength:"200" doc:"Only entries whose title holds this text"`
}

type blocklist struct {
	Entries []blockEntry `json:"entries" nullable:"false" doc:"Up to 50, newest first"`
	Total   int          `json:"total" minimum:"0" doc:"How many entries match in all"`
}

type blocklistOutput struct{ Body blocklist }

type unblockInput struct {
	Body struct {
		InfoHashes []string `json:"infoHashes" pattern:"^[0-9a-f]{40}$" minItems:"1" maxItems:"1000" nullable:"false"`
	}
}

func (h handlers) listBlocklist(ctx context.Context, by identity.Admin, in *blocklistInput) (*blocklistOutput, error) {
	page, err := open(h.p).Blocklist(ctx, by, grabs.BlocklistQuery{Title: in.Q, Offset: in.Offset})
	if err != nil {
		return nil, answer(err)
	}
	out := blocklist{Entries: []blockEntry{}, Total: page.Total}
	for _, e := range page.Entries {
		out.Entries = append(out.Entries, blockEntryOf(e))
	}
	return &blocklistOutput{Body: out}, nil
}

func blockEntryOf(e grabs.BlockEntry) blockEntry {
	out := blockEntry{
		InfoHash: e.InfoHash.String(), Title: e.Title, Words: e.Words, Cause: string(e.Cause), Because: e.Because, AddedAt: e.Added,
	}
	if e.IndexerName != "" {
		out.Indexer = new(e.IndexerName)
	}
	if !e.Work.IsZero() {
		out.Work = api.Some(workRef{ID: e.Work.String(), Title: e.WorkTitle})
	}
	return out
}

func (h handlers) removeFromBlocklist(ctx context.Context, by identity.Admin, in *unblockInput) (*struct{}, error) {
	hashes := make([]bencode.InfoHash, len(in.Body.InfoHashes))
	for i, s := range in.Body.InfoHashes {
		hash, err := bencode.ParseInfoHash(s)
		if err != nil {
			return nil, api.Invalid
		}
		hashes[i] = hash
	}
	return nil, answer(open(h.p).Unblock(ctx, by, hashes))
}
