package worksapi

import (
	"context"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/works"
)

type searchInput struct {
	Q           string `query:"q" required:"true" minLength:"1" maxLength:"200" doc:"Words each match's titles hold as word prefixes, case and accents ignored"`
	Limit       int    `query:"limit" minimum:"0" maximum:"20000" default:"8" doc:"How many rows to return; 0 returns ids only"`
	ContentType string `query:"contentType" pattern:"^[a-z][a-z0-9]{1,15}$" doc:"Only this Content Type's Works"`
}

type searchResult struct {
	WorkIDs []string     `json:"workIds" nullable:"false" doc:"Every matching Work, best first"`
	Rows    []libraryRow `json:"rows" nullable:"false" doc:"The rows of the first limit of them, in the same order"`
}

type searchResultOutput struct{ Body searchResult }

func (h handlers) searchWorks(ctx context.Context, by identity.Admin, in *searchInput) (*searchResultOutput, error) {
	var only *works.TypeID
	if in.ContentType != "" {
		t := works.TypeID(in.ContentType)
		only = &t
	}
	ids, rows, err := composer(h.shelf).Search(ctx, by, in.Q, in.Limit, only)
	if err != nil {
		return nil, answer(err)
	}
	out := searchResult{WorkIDs: make([]string, len(ids)), Rows: rowsOf(minter(h.shelf), rows)}
	for i, id := range ids {
		out.WorkIDs[i] = id.String()
	}
	return &searchResultOutput{Body: out}, nil
}
