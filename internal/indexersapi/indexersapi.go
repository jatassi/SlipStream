package indexersapi

import (
	"context"
	"errors"
	"maps"
	"net/http"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
)

type Registry interface {
	// Indexers is the Indexers, false unless the Node is a Controller.
	Indexers() (*indexers.Indexers, bool)
}

func open(r Registry) *indexers.Indexers {
	x, ok := r.Indexers()
	if !ok {
		panic("indexersapi: an operation ran on a Node that is no Controller")
	}
	return x
}

var (
	indexerNameTaken   = api.NewCode(http.StatusConflict, "indexer_name_taken", "Another Indexer has that name")
	noSearchingNode    = api.NewCode(http.StatusConflict, "no_searching_node", "No Node holds Searching. Give one the Searching Role first.")
	indexerUnavailable = api.NewCode(http.StatusBadGateway, "indexer_unavailable", "The Indexer didn't answer")
	indexerRefused     = api.NewCode(http.StatusUnprocessableEntity, "indexer_refused", "The Indexer refused the request")
)

func answer(err error) error {
	var unavailable *indexers.UnavailableError
	var refused *indexers.RefusedError
	switch {
	case errors.As(err, &refused):
		return indexerRefused.Because("error " + refused.Code + ": " + refused.Description)
	case errors.As(err, &unavailable):
		return indexerUnavailable.Because(unavailable.Err.Error())
	case errors.Is(err, indexers.ErrNameTaken):
		return indexerNameTaken
	case errors.Is(err, indexers.ErrNoSearchingNode):
		return noSearchingNode
	}
	return err
}

type searchMode struct {
	Mode   string   `json:"mode" doc:"The Torznab t= value"`
	Params []string `json:"params" nullable:"false" doc:"In the Indexer's order"`
}

type indexer struct {
	ID          string       `json:"id" format:"uuid"`
	Name        string       `json:"name"`
	Protocol    string       `json:"protocol" enum:"torznab"`
	URL         string       `json:"url"`
	HomeNodeID  string       `json:"homeNodeId" format:"uuid" doc:"The Node that reaches the Indexer"`
	SearchModes []searchMode `json:"searchModes" nullable:"false" doc:"By mode"`
	CreatedAt   time.Time    `json:"createdAt"`
}

func indexerOf(x indexers.Indexer) indexer {
	out := indexer{ID: x.ID.String(), Name: x.Name, Protocol: x.Protocol, URL: x.URL, HomeNodeID: x.HomeNode.String(), SearchModes: []searchMode{}, CreatedAt: x.CreatedAt}
	for _, m := range slices.Sorted(maps.Keys(x.Caps.Modes)) {
		out.SearchModes = append(out.SearchModes, searchMode{Mode: m, Params: x.Caps.Modes[m]})
	}
	return out
}

type newIndexer struct {
	Name   string `json:"name" minLength:"1" maxLength:"100"`
	URL    string `json:"url" maxLength:"2048" pattern:"^https?://[^/?#]+(/[^?#]*)?$" doc:"The Torznab feed's base; SlipStream adds /api"`
	APIKey string `json:"apiKey" maxLength:"256"`
}

type createIndexerInput struct{ Body newIndexer }

type indexerOutput struct{ Body indexer }

type indexerList struct {
	Indexers []indexer `json:"indexers" nullable:"false" doc:"By name"`
}

type indexerListOutput struct{ Body indexerList }

func Routes(r *api.Router, registry Registry) {
	o := r.Owner("indexers", api.Admin)
	h := handlers{registry}

	api.Register(o, api.Admin, api.Op{
		ID:      "createIndexer",
		Method:  http.MethodPost,
		Path:    "/indexers",
		Summary: "Add a Torznab Indexer, once its home Node has read its caps",
		Status:  http.StatusCreated,
		Errors:  []*api.Code{indexerNameTaken, noSearchingNode, indexerUnavailable, indexerRefused},
	}, h.createIndexer)

	api.Register(o, api.Admin, api.Op{
		ID:      "listIndexers",
		Method:  http.MethodGet,
		Path:    "/indexers",
		Summary: "List the Indexers",
		Status:  http.StatusOK,
	}, h.listIndexers)
}

type handlers struct{ registry Registry }

func (h handlers) createIndexer(ctx context.Context, by identity.Admin, in *createIndexerInput) (*indexerOutput, error) {
	x, err := open(h.registry).Create(ctx, by, indexers.Draft{Name: in.Body.Name, URL: in.Body.URL, APIKey: in.Body.APIKey})
	if err != nil {
		return nil, answer(err)
	}
	return &indexerOutput{Body: indexerOf(x)}, nil
}

func (h handlers) listIndexers(ctx context.Context, by identity.Admin, _ *struct{}) (*indexerListOutput, error) {
	list, err := open(h.registry).List(ctx, by)
	if err != nil {
		return nil, err
	}
	out := indexerList{Indexers: []indexer{}}
	for _, x := range list {
		out.Indexers = append(out.Indexers, indexerOf(x))
	}
	return &indexerListOutput{Body: out}, nil
}
