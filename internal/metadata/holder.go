package metadata

import (
	"context"
	"fmt"

	"github.com/jatassi/slipstream/internal/metadata/artwork"
	metadatav1 "github.com/jatassi/slipstream/internal/pb/slipstream/metadata/v1"
	"github.com/jatassi/slipstream/internal/pb/slipstream/metadata/v1/metadatav1connect"
	"github.com/jatassi/slipstream/internal/rpc"
	"github.com/jatassi/slipstream/internal/works"
)

// Holder is MetadataService on a Node. It runs a Content Type's provider
// half for the Ask each call carries and answers with every outcome the
// call produced, a provider failure included. It keeps nothing but its
// Lines' in-memory state.
type Holder struct {
	metadatav1connect.UnimplementedMetadataServiceHandler
	types   map[works.TypeID]Source
	testers map[ProviderID]Tester
	images  map[artwork.SourceID]imageHost
}

// Tester is a provider client's cheap call that says whether the provider
// answers with an Ask's key.
type Tester interface {
	Provider() Provider
	Test(ctx context.Context, ask Ask) error
}

type HolderConfig struct {
	Types   map[works.TypeID]Source
	Testers []Tester
	// Lines carry the image hosts the types' Sources name, each on a Line
	// of its own.
	Lines *Lines
}

func NewHolder(cfg HolderConfig) *Holder {
	h := &Holder{types: cfg.Types, testers: map[ProviderID]Tester{}, images: map[artwork.SourceID]imageHost{}}
	for _, t := range cfg.Testers {
		h.testers[t.Provider().ID] = t
	}
	var lists [][]artwork.Source
	for _, s := range cfg.Types {
		lists = append(lists, s.Sources())
	}
	for id, src := range artwork.NewSources(lists...) {
		h.images[id] = imageHost{src: src, line: cfg.Lines.Line(imageProvider(src))}
	}
	return h
}

var _ metadatav1connect.MetadataServiceHandler = (*Holder)(nil)

func (h *Holder) Search(ctx context.Context, req *metadatav1.SearchRequest) (*metadatav1.SearchResponse, error) {
	found, failure, outcomes, err := run(ctx, h, req.GetAsk(), req.GetContentType(), func(s Source, ask Ask) ([]works.Candidate, error) {
		return s.Search(ctx, ask, req.GetQuery())
	})
	if err != nil || failure != nil {
		return &metadatav1.SearchResponse{Failure: failure, Outcomes: outcomes}, err
	}
	candidates, err := works.EncodeCandidates(found)
	return &metadatav1.SearchResponse{Candidates: candidates, Outcomes: outcomes}, err
}

func (h *Holder) Fetch(ctx context.Context, req *metadatav1.FetchRequest) (*metadatav1.FetchResponse, error) {
	f, failure, outcomes, err := run(ctx, h, req.GetAsk(), req.GetContentType(), func(s Source, ask Ask) (works.Fetched, error) {
		return s.Fetch(ctx, ask, works.Key(req.GetKey()))
	})
	if err != nil || failure != nil {
		return &metadatav1.FetchResponse{Failure: failure, Outcomes: outcomes}, err
	}
	fetched, err := works.EncodeFetched(f)
	return &metadatav1.FetchResponse{Fetched: fetched, Outcomes: outcomes}, err
}

// Test runs the provider's test call with the key the Ask carries and
// answers its outcomes, a refusal included.
func (h *Holder) Test(ctx context.Context, req *metadatav1.TestRequest) (*metadatav1.TestResponse, error) {
	t, ok := h.testers[ProviderID(req.GetProvider())]
	if !ok {
		return nil, rpc.Errorf(rpc.CodeInvalidArgument, "this Node tests no provider %q", req.GetProvider())
	}
	ask, read := tallied(askOf(req.GetAsk()))
	if err := t.Test(ctx, ask); err != nil {
		if _, failed := failureToWire(err); !failed {
			return nil, fmt.Errorf("test %s: %w", req.GetProvider(), err)
		}
	}
	return &metadatav1.TestResponse{Outcomes: outcomesToWire(read())}, nil
}

// run calls the provider half of type t with the Ask w carries. A provider
// failure is answered with the outcomes; any other error fails the call.
func run[T any](ctx context.Context, h *Holder, w *metadatav1.Ask, t string, call func(Source, Ask) (T, error)) (T, *metadatav1.Failure, []*metadatav1.Outcome, error) {
	var none T
	s, ok := h.types[works.TypeID(t)]
	if !ok {
		return none, nil, nil, rpc.Errorf(rpc.CodeInvalidArgument, "this Node opens no Content Type %q", t)
	}
	ask, read := tallied(askOf(w))
	v, err := call(s, ask)
	outcomes := outcomesToWire(read())
	if err == nil {
		return v, nil, outcomes, nil
	}
	if f, ok := failureToWire(err); ok {
		return none, f, outcomes, nil
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		return none, nil, nil, fmt.Errorf("%s: %w", t, ctxErr)
	}
	return none, nil, nil, rpc.Errorf(rpc.CodeInternal, "%v", err)
}
