package nodesystem

import (
	"context"
	"crypto/rand"
	"errors"
	"os"
	"sync"
	"time"

	systemv1 "github.com/jatassi/slipstream/internal/pb/slipstream/system/v1"
	"github.com/jatassi/slipstream/internal/pb/slipstream/system/v1/systemv1connect"
	"github.com/jatassi/slipstream/internal/rpc"
)

// archiveChunk is the most one ReadLogArchive answers.
const archiveChunk = 512 << 10

// ServeTimings are the serving Node's timings.link: the longest a
// ReadLogs waits, and how long an archive waits for its reader.
type ServeTimings struct {
	LogPoll       time.Duration
	LogArchiveTTL time.Duration
}

// Serve is NodeSystemService over h. It keeps each archive it opened until
// its last chunk is read or it goes unread for its TTL, then deletes it.
func Serve(h Host, t ServeTimings) systemv1connect.NodeSystemServiceHandler {
	return &server{host: h, timings: t, archives: map[string]*openArchive{}}
}

type server struct {
	host    Host
	timings ServeTimings

	mu       sync.Mutex
	archives map[string]*openArchive
}

type openArchive struct {
	Archive
	expire *time.Timer
}

// wire is err as a call answers it: a code the Controller maps back to the
// domain error. A starting or stopping Node answers FailedPrecondition, so
// its answer never reads as a call the link lost.
func wire(err error) error {
	switch {
	case errors.Is(err, ErrAlreadyRestarting):
		return rpc.Errorf(rpc.CodeAborted, "%v", err)
	case errors.Is(err, ErrOffline):
		return rpc.Errorf(rpc.CodeFailedPrecondition, "%v", err)
	}
	return err
}

func (s *server) GetFacts(context.Context, *systemv1.GetFactsRequest) (*systemv1.GetFactsResponse, error) {
	f, err := s.host.Facts()
	if err != nil {
		return nil, wire(err)
	}
	return factsToWire(f), nil
}

func (s *server) GetConfig(context.Context, *systemv1.GetConfigRequest) (*systemv1.GetConfigResponse, error) {
	c, err := s.host.Config()
	if err != nil {
		return nil, wire(err)
	}
	return &systemv1.GetConfigResponse{Config: configToWire(c)}, nil
}

func (s *server) SetConfig(ctx context.Context, req *systemv1.SetConfigRequest) (*systemv1.SetConfigResponse, error) {
	changes := make([]Change, len(req.GetChanges()))
	for i, c := range req.GetChanges() {
		changes[i] = changeFromWire(c)
	}
	c, err := s.host.SetConfig(ctx, changes)
	if refused, ok := errors.AsType[*Refusal](err); ok {
		return &systemv1.SetConfigResponse{Result: &systemv1.SetConfigResponse_Refused{Refused: refusalToWire(refused)}}, nil
	}
	if err != nil {
		return nil, wire(err)
	}
	return &systemv1.SetConfigResponse{Result: &systemv1.SetConfigResponse_Config{Config: configToWire(c)}}, nil
}

func (s *server) Restart(context.Context, *systemv1.RestartRequest) (*systemv1.RestartResponse, error) {
	r, err := s.host.Restart()
	if err != nil {
		return nil, wire(err)
	}
	return &systemv1.RestartResponse{StartedAtUnixNano: r.StartedAt.UnixNano(), BoundPort: uint32(r.BoundPort), NextPort: uint32(r.NextPort)}, nil
}

func (s *server) ReadLogs(ctx context.Context, req *systemv1.ReadLogsRequest) (*systemv1.ReadLogsResponse, error) {
	b, err := s.host.ReadLogs(ctx, req.GetAfter(), s.timings.LogPoll)
	if err != nil {
		return nil, wire(err)
	}
	return batchToWire(b), nil
}

func (s *server) OpenLogArchive(context.Context, *systemv1.OpenLogArchiveRequest) (*systemv1.OpenLogArchiveResponse, error) {
	a, err := s.host.Archive()
	if err != nil {
		return nil, wire(err)
	}
	id := rand.Text()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.archives[id] = &openArchive{Archive: a, expire: time.AfterFunc(s.timings.LogArchiveTTL, func() { s.drop(id) })}
	return &systemv1.OpenLogArchiveResponse{Id: id, Size: a.Size, Stamp: a.Stamp}, nil
}

func (s *server) ReadLogArchive(_ context.Context, req *systemv1.ReadLogArchiveRequest) (*systemv1.ReadLogArchiveResponse, error) {
	s.mu.Lock()
	a, ok := s.archives[req.GetId()]
	s.mu.Unlock()
	if !ok {
		return nil, rpc.Errorf(rpc.CodeNotFound, "no log archive %s is open", req.GetId())
	}
	offset := req.GetOffset()
	if offset < 0 || offset >= a.Size {
		return nil, rpc.Errorf(rpc.CodeOutOfRange, "offset %d is outside the archive's %d bytes", offset, a.Size)
	}
	data := make([]byte, min(archiveChunk, a.Size-offset))
	if _, err := a.File.ReadAt(data, offset); err != nil {
		return nil, rpc.Errorf(rpc.CodeInternal, "read the log archive: %v", err)
	}
	if offset+int64(len(data)) == a.Size {
		s.drop(req.GetId())
	} else {
		a.expire.Reset(s.timings.LogArchiveTTL)
	}
	return &systemv1.ReadLogArchiveResponse{Data: data}, nil
}

func (s *server) Reach(ctx context.Context, req *systemv1.ReachRequest) (*systemv1.ReachResponse, error) {
	t := Target{Host: req.GetHost(), Port: uint16(req.GetPort()), TLS: req.GetTls()} //nolint:gosec // a port the Controller sent from a uint16
	r := s.host.Reach(ctx, t)
	return &systemv1.ReachResponse{Ok: r.OK, Millis: r.Took.Milliseconds(), Error: r.Err}, nil
}

func (s *server) drop(id string) {
	s.mu.Lock()
	a, ok := s.archives[id]
	delete(s.archives, id)
	s.mu.Unlock()
	if !ok {
		return
	}
	a.expire.Stop()
	_ = a.File.Close()
	_ = os.Remove(a.File.Name())
}
