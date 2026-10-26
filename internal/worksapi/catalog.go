package worksapi

import (
	"context"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

type candidate struct {
	Key       string                  `json:"key"`
	Label     string                  `json:"label"`
	Original  string                  `json:"original" doc:"The label in its original language; empty when it is the label"`
	Overview  string                  `json:"overview" doc:"One line; empty when the source has none"`
	Fields    []api.OneOf[fieldValue] `json:"fields" nullable:"false" doc:"The Work's fields"`
	PosterURL *string                 `json:"posterUrl" doc:"The artwork route for the result's poster, without a size; null when the source has none"`
	WorkID    *string                 `json:"workId" format:"uuid" doc:"The Work, once it is in the library"`
}

type catalogInput struct {
	ContentType string `path:"contentType" pattern:"^[a-z][a-z0-9]{1,15}$"`
	Query       string `query:"query" required:"true" minLength:"1"`
}

type candidateList struct {
	Candidates []candidate `json:"candidates" nullable:"false" doc:"Best first"`
}

type candidateListOutput struct{ Body candidateList }

type recordInput struct {
	ContentType string `path:"contentType" pattern:"^[a-z][a-z0-9]{1,15}$"`
	Key         string `path:"key" minLength:"1" maxLength:"200" doc:"The Work's key at its metadata source"`
}

type previewed struct {
	InLibrary bool `json:"inLibrary" doc:"Whether the Work is in the library; while it isn't, the ids in work name nothing"`
	Work      Tree `json:"work"`
}

type previewedOutput struct{ Body previewed }

func (h handlers) searchCatalog(ctx context.Context, by identity.Admin, in *catalogInput) (*candidateListOutput, error) {
	found, err := open(h.shelf).Search(ctx, by, works.TypeID(in.ContentType), in.Query)
	if err != nil {
		return nil, answer(err)
	}
	m := minter(h.shelf)
	out := candidateList{Candidates: []candidate{}}
	for _, c := range found {
		out.Candidates = append(out.Candidates, candidate{
			Key: string(c.Key), Label: c.Label, Original: c.Original, Overview: c.Overview, Fields: fieldsOf(m, c.Fields),
			PosterURL: urlOrNull(m, c.Poster), WorkID: idOrNull(c.WorkID.UUID),
		})
	}
	return &candidateListOutput{Body: out}, nil
}

func (h handlers) previewCatalogRecord(ctx context.Context, by identity.Admin, in *recordInput) (*previewedOutput, error) {
	key, err := works.ParseKey(in.Key)
	if err != nil {
		return nil, api.Invalid
	}
	var timings map[works.ItemID]targets.Timing
	t, stored, err := open(h.shelf).Preview(ctx, by, works.TypeID(in.ContentType), key, func(ctx context.Context, r storage.Reader[storage.Library], t works.Tree) error {
		var err error
		timings, err = goals(h.shelf).TimingsIn(ctx, r, t.ContentType, []works.WorkID{t.ID})
		return err
	})
	if err != nil {
		return nil, answer(err)
	}
	return &previewedOutput{Body: previewed{InLibrary: stored, Work: treeOf(minter(h.shelf), t, timings)}}, nil
}
