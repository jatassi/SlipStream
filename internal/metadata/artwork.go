package metadata

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"github.com/jatassi/slipstream/internal/metadata/artwork"
	metadatav1 "github.com/jatassi/slipstream/internal/pb/slipstream/metadata/v1"
	"github.com/jatassi/slipstream/internal/pb/slipstream/metadata/v1/metadatav1connect"
	"github.com/jatassi/slipstream/internal/rpc"
)

// imageHost is one image host a holder fetches from, over a Line of its
// own.
type imageHost struct {
	src  artwork.Source
	line *Line
}

// imageProvider is an image host as the call layer knows it: keyless, at
// the rate the host's source declares.
func imageProvider(src artwork.Source) Provider {
	perSecond, burst := src.Rate()
	return Provider{ID: ProviderID(src.ID()), Label: string(src.ID()), Rate: rate.Limit(perSecond), Burst: burst}
}

// Image fetches one image from the host its reference names. It answers no
// outcomes: what an image host answers says nothing of a provider's key.
func (h *Holder) Image(ctx context.Context, req *metadatav1.ImageRequest) (*metadatav1.ImageResponse, error) {
	ref := req.GetRef()
	host, ok := h.images[artwork.SourceID(ref.GetSource())]
	if !ok {
		return nil, rpc.Errorf(rpc.CodeInvalidArgument, "this Node fetches from no image host %q", ref.GetSource())
	}
	shape, shapeErr := artwork.ParseShape(ref.GetShape())
	size, sizeErr := artwork.ParseSize(req.GetSize())
	if err := errors.Join(shapeErr, sizeErr); err != nil {
		return nil, rpc.Errorf(rpc.CodeInvalidArgument, "%v", err)
	}
	u, err := host.src.URL(shape, ref.GetPath(), size)
	if err != nil {
		return nil, rpc.Errorf(rpc.CodeInvalidArgument, "%v", err)
	}
	img, err := host.line.Bytes(ctx, askOf(req.GetAsk()), u)
	if f, failed := failureToWire(err); failed {
		return &metadatav1.ImageResponse{Failure: f}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("image %s: %w", ref.GetPath(), err)
	}
	return &metadatav1.ImageResponse{Body: img.Body, ContentType: img.Type}, nil
}

// gallery is the Controller's artwork: the store, the fetches in flight
// and the images waiting to be prefetched. Every fetch ends with Run.
type gallery struct {
	store *artwork.Store
	life  context.Context
	end   context.CancelFunc

	mu      sync.Mutex
	flights map[artwork.Want]*flight
	flying  sync.WaitGroup
	queued  []artwork.Want
	wake    chan struct{}
}

// flight is one fetch that every request for its image waits on.
type flight struct {
	done chan struct{}
	img  artwork.Image
	err  error
}

func newGallery() *gallery {
	life, end := context.WithCancel(context.Background())
	return &gallery{life: life, end: end, flights: map[artwork.Want]*flight{}, wake: make(chan struct{}, 1)}
}

// close ends every fetch and waits for them.
func (g *gallery) close() {
	g.mu.Lock()
	g.end()
	g.mu.Unlock()
	g.flying.Wait()
}

// openStore opens the artwork store bounded by the artwork store size
// setting as the Controller holds it.
func (m *Metadata) openStore(ctx context.Context) (*artwork.Store, error) {
	return artwork.OpenStore(ctx, m.cfg.ArtworkDir, m.cfg.Local, func(context.Context) (int64, error) {
		return ArtworkStore.Value(m.stored().ArtworkStoreMb), nil
	})
}

// Artwork is the image token names, at size: the stored one, or else one
// fetched through a holder, checked whole and stored first. A token
// SlipStream didn't issue is artwork.ErrForged. An image that can't be
// fetched is artwork.ErrMissing, and is answered so without another
// fetch until timings.metadata.artwork_retry has passed.
func (m *Metadata) Artwork(ctx context.Context, token string, size artwork.Size) (artwork.Image, error) {
	ref, err := m.cfg.Minter.Open(token)
	if err != nil {
		return artwork.Image{}, err
	}
	return m.image(ctx, artwork.Want{Ref: ref, Size: size})
}

// Prefetch fetches wants in the background, ahead of the first request
// for each.
func (m *Metadata) Prefetch(wants []artwork.Want) {
	g := m.gallery
	g.mu.Lock()
	g.queued = append(g.queued, wants...)
	g.mu.Unlock()
	select {
	case g.wake <- struct{}{}:
	default:
	}
}

func (m *Metadata) prefetching(ctx context.Context) {
	g := m.gallery
	for {
		select {
		case <-ctx.Done():
			return
		case <-g.wake:
		}
		g.mu.Lock()
		wants := g.queued
		g.queued = nil
		g.mu.Unlock()
		for _, w := range wants {
			_, _ = m.image(ctx, w)
		}
	}
}

// image is w from the store, or fetched once however many ask for it at
// a time.
func (m *Metadata) image(ctx context.Context, w artwork.Want) (artwork.Image, error) {
	g := m.gallery
	img, err := g.store.Open(ctx, w)
	if !errors.Is(err, artwork.ErrNotHeld) {
		return img, err
	}
	f, err := m.fly(w)
	if err != nil {
		return artwork.Image{}, err
	}
	select {
	case <-f.done:
		return f.img, f.err
	case <-ctx.Done():
		return artwork.Image{}, fmt.Errorf("waiting on %s: %w", w.Ref.Path, ctx.Err())
	}
}

// fly is w's fetch in flight, started when none is.
func (m *Metadata) fly(w artwork.Want) (*flight, error) {
	g := m.gallery
	g.mu.Lock()
	defer g.mu.Unlock()
	if err := g.life.Err(); err != nil {
		return nil, fmt.Errorf("fetching %s: %w", w.Ref.Path, err)
	}
	if f, ok := g.flights[w]; ok {
		return f, nil
	}
	f := &flight{done: make(chan struct{})}
	g.flights[w] = f
	g.flying.Go(func() {
		f.img, f.err = m.fetch(g.life, w)
		g.mu.Lock()
		delete(g.flights, w)
		g.mu.Unlock()
		close(f.done)
	})
	return f, nil
}

// fetch reads w through a holder and stores it. An image that doesn't
// arrive whole is marked missing until the retry time.
func (m *Metadata) fetch(ctx context.Context, w artwork.Want) (artwork.Image, error) {
	img, err := m.fromHolder(ctx, w)
	if err == nil {
		img, err = m.gallery.store.Put(ctx, w, img)
		if err == nil || !errors.Is(err, artwork.ErrUnfit) {
			return img, err
		}
	}
	retry := time.Now().Add(m.cfg.Timings.ArtworkRetry)
	return artwork.Image{}, errors.Join(fmt.Errorf("%w: %w", artwork.ErrMissing, err), m.gallery.store.MarkMissing(ctx, w, retry))
}

func (m *Metadata) fromHolder(ctx context.Context, w artwork.Want) (artwork.Image, error) {
	req := &metadatav1.ImageRequest{
		Ask:  askToWire(Ask{}),
		Ref:  &metadatav1.ArtRef{Source: string(w.Ref.Source), Shape: string(w.Ref.Shape), Path: w.Ref.Path},
		Size: string(w.Size),
	}
	res, _, err := call(ctx, m, func(ctx context.Context, s metadatav1connect.MetadataServiceHandler) (*metadatav1.ImageResponse, error) {
		return s.Image(ctx, req)
	})
	if err != nil {
		return artwork.Image{}, err
	}
	if f := res.GetFailure(); f != nil {
		return artwork.Image{}, failureOf(f)
	}
	return artwork.Image{Type: res.GetContentType(), Body: res.GetBody()}, nil
}
