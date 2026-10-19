package metadata

import (
	"context"
	"fmt"

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
	types map[works.TypeID]Source
}

func NewHolder(types map[works.TypeID]Source) *Holder { return &Holder{types: types} }

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
