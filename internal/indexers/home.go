package indexers

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"

	searchv1 "github.com/jatassi/slipstream/internal/pb/slipstream/search/v1"
	"github.com/jatassi/slipstream/internal/rpc"
)

const (
	searchLimit  = 8 << 20
	torrentLimit = 10 << 20
)

// Home is the Searching service an Indexer's home Node serves. It sends
// each request and returns the answer unparsed, within the request
// timing and a body cap, and never leaves the Indexer's own host.
type Home struct {
	client  *http.Client
	timings Timings
}

func NewHome(client *http.Client, t Timings) *Home {
	own := *client
	own.CheckRedirect = sameHost
	return &Home{client: &own, timings: t}
}

func sameHost(req *http.Request, via []*http.Request) error {
	if !onHost(req.URL, via[0].URL) {
		return errors.New("the Indexer redirected off its own host")
	}
	if len(via) >= 10 {
		return errors.New("the Indexer redirected 10 times")
	}
	return nil
}

func onHost(u, home *url.URL) bool { return u.Scheme == home.Scheme && u.Host == home.Host }

func (h *Home) Caps(ctx context.Context, req *searchv1.CapsRequest) (*searchv1.CapsResponse, error) {
	body, status, err := h.api(ctx, req.GetIndexer(), map[string]string{"t": "caps"})
	if err != nil {
		return nil, err
	}
	return &searchv1.CapsResponse{Body: body, Status: status}, nil
}

func (h *Home) Search(ctx context.Context, req *searchv1.SearchRequest) (*searchv1.SearchResponse, error) {
	body, status, err := h.api(ctx, req.GetIndexer(), req.GetParams())
	if err != nil {
		return nil, err
	}
	return &searchv1.SearchResponse{Body: body, Status: status}, nil
}

func (h *Home) FetchTorrent(ctx context.Context, req *searchv1.FetchTorrentRequest) (*searchv1.FetchTorrentResponse, error) {
	home, err := indexerURL(req.GetIndexer())
	if err != nil {
		return nil, err
	}
	link, err := url.Parse(req.GetUrl())
	if err != nil || !onHost(link, home) {
		return nil, rpc.Errorf(rpc.CodeInvalidArgument, "the torrent's URL is not on the Indexer's host")
	}
	body, status, err := h.get(ctx, link, torrentLimit)
	if err != nil {
		return nil, err
	}
	if status < http.StatusOK || status >= http.StatusMultipleChoices {
		return nil, rpc.Errorf(rpc.CodeAborted, "the Indexer answered status %d", status)
	}
	return &searchv1.FetchTorrentResponse{Torrent: body}, nil
}

func indexerURL(idx *searchv1.Indexer) (*url.URL, error) {
	u, err := url.Parse(idx.GetUrl())
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, rpc.Errorf(rpc.CodeInvalidArgument, "the Indexer's URL is not an http or https URL")
	}
	return u, nil
}

// api sends params, with the Indexer's API key, to its Torznab api path.
func (h *Home) api(ctx context.Context, idx *searchv1.Indexer, params map[string]string) (body []byte, status int32, err error) {
	base, err := indexerURL(idx)
	if err != nil {
		return nil, 0, err
	}
	u := base.JoinPath("api")
	q := url.Values{}
	for k, v := range params {
		q.Set(k, v)
	}
	q.Set("apikey", idx.GetApiKey())
	u.RawQuery = q.Encode()
	return h.get(ctx, u, searchLimit)
}

func (h *Home) get(ctx context.Context, u *url.URL, limit int64) (body []byte, status int32, err error) {
	ctx, cancel := context.WithTimeout(ctx, h.timings.Request)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), http.NoBody)
	if err != nil {
		return nil, 0, rpc.Errorf(rpc.CodeInvalidArgument, "build the request: %v", err)
	}
	res, err := h.client.Do(req)
	if err != nil {
		return nil, 0, rpc.Errorf(rpc.CodeAborted, "reach the Indexer: %v", withoutURL(err))
	}
	defer func() { _ = res.Body.Close() }()
	body, err = io.ReadAll(io.LimitReader(res.Body, limit+1))
	if err != nil {
		return nil, 0, rpc.Errorf(rpc.CodeAborted, "read the Indexer's answer: %v", err)
	}
	if int64(len(body)) > limit {
		return nil, 0, rpc.Errorf(rpc.CodeResourceExhausted, "the Indexer's answer is over %d bytes", limit)
	}
	return body, int32(res.StatusCode), nil //nolint:gosec // an HTTP status fits
}

// withoutURL drops the request URL from err, since it carries the API key.
func withoutURL(err error) error {
	if ue, ok := errors.AsType[*url.Error](err); ok {
		return ue.Err
	}
	return err
}
