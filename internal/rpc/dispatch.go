package rpc

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"

	"connectrpc.com/connect/v2"
	"connectrpc.com/connect/v2/connecthttp"
	"connectrpc.com/connect/v2/connectinprocess"
	"connectrpc.com/connect/v2/connectproto"

	"github.com/jatassi/slipstream/internal/link"
	nodev1 "github.com/jatassi/slipstream/internal/pb/slipstream/node/v1"
	"github.com/jatassi/slipstream/internal/role"
)

// Dispatch is on every claimed Node. It serves the services mounted on it
// to the link, as the link's lockstep handler, and to this Node's own calls
// through Local.
type Dispatch struct {
	self, controller link.Key
	link             *link.Link
	holds            func(role.Name) bool
	mux              *http.ServeMux

	mu      sync.Mutex
	servers map[string]*connect.Server
}

// NewDispatch dispatches for the Node holding self. controller is the
// Controller's key, self on the Controller. holds reports whether this Node
// holds a Role now; a Role service's calls are refused while it does not.
func NewDispatch(l *link.Link, self, controller link.Key, holds func(role.Name) bool) *Dispatch {
	return &Dispatch{
		self:       self,
		controller: controller,
		link:       l,
		holds:      holds,
		mux:        http.NewServeMux(),
		servers:    map[string]*connect.Server{},
	}
}

// ServeHTTP serves the mounted services to the link's peers.
func (d *Dispatch) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	d.mux.ServeHTTP(w, r)
}

func (d *Dispatch) mount(name string, held role.Name, register func(*connect.Server)) {
	var gates []connect.ServerInterceptor
	if held != "" {
		gates = append(gates, d.gate(held))
	}
	srv := connect.NewServer(gates...)
	register(srv)
	connecthttp.Mount(d.mux, srv)
	d.mu.Lock()
	defer d.mu.Unlock()
	d.servers[name] = srv
}

func (d *Dispatch) gate(held role.Name) connect.ServerInterceptor {
	return func(next connect.ServerFunc) connect.ServerFunc {
		return func(ctx context.Context, spec connect.Spec, stream connect.ServerStream) error {
			if !d.holds(held) {
				return notHolder(held, "this Node does not hold the "+string(held)+" Role")
			}
			return next(ctx, spec, stream)
		}
	}
}

// client calls the Node holding key: Local for this Node, else the link.
func (d *Dispatch) client(key link.Key) *connect.Client {
	if key == d.self {
		return connect.NewClient(Local{d})
	}
	return d.link.Client(key)
}

// Local is the in-process path to this Node's mounted services. Like the
// wire, it hands the handler a clone of the request and the caller a clone
// of the response, names this Node as the caller, and gives the caller only
// the error's code, message and details.
type Local struct{ d *Dispatch }

func (l Local) NewClientStream(ctx context.Context, spec connect.Spec) (connect.ClientStream, error) {
	name, _, _ := strings.Cut(strings.TrimPrefix(spec.Procedure, "/"), "/")
	l.d.mu.Lock()
	srv := l.d.servers[name]
	l.d.mu.Unlock()
	if srv == nil {
		return nil, connect.Errorf(connect.CodeUnimplemented, "%s is not mounted on this Node", name).WithRemote()
	}
	s, err := connectinprocess.New(srv).NewClientStream(context.WithValue(ctx, callerKey{}, l.d.self), spec)
	if err != nil {
		return nil, flatten(ctx, err)
	}
	return flattened{s, ctx}, nil
}

type flattened struct {
	connect.ClientStream
	ctx context.Context //nolint:containedctx // the call's context decides how a bare context error is coded
}

func (s flattened) Receive(msg any) error { return flatten(s.ctx, s.ClientStream.Receive(msg)) }

// flatten keeps what the wire would carry of err.
func flatten(ctx context.Context, err error) error {
	if err == nil || errors.Is(err, io.EOF) {
		return err
	}
	ce, ok := errors.AsType[*connect.Error](err)
	if !ok {
		ce = connect.NewError(codeOfBare(ctx), err.Error())
	}
	out := connect.NewError(ce.Code(), ce.Message())
	for _, detail := range ce.Details() {
		out = out.WithDetail(detail)
	}
	return out.WithRemote()
}

func codeOfBare(ctx context.Context) connect.Code {
	switch {
	case errors.Is(ctx.Err(), context.Canceled):
		return connect.CodeCanceled
	case errors.Is(ctx.Err(), context.DeadlineExceeded):
		return connect.CodeDeadlineExceeded
	}
	return connect.CodeUnknown
}

// Roster is the Controller's view of its Nodes, which the nodes package
// keeps in memory.
type Roster interface {
	// Key is node's link key, false for a node the Controller does not know.
	Key(node role.NodeID) (link.Key, bool)
	// Serves reports whether node holds, drains or covers r.
	Serves(r role.Name, node role.NodeID) bool
}

// Cluster is the Controller's dispatch: the Node services and Role services
// it calls on chosen Nodes. Only the Controller builds one, so a Worker
// calling a Worker or a Role service doesn't compile.
type Cluster struct {
	d      *Dispatch
	roster Roster
}

func NewCluster(d *Dispatch, r Roster) *Cluster {
	return &Cluster{d: d, roster: r}
}

// client calls node, or fails every call with NotHolder when the roster
// does not know node or, for a Role call, does not list it as a holder.
func (c *Cluster) client(node role.NodeID, held role.Name) *connect.Client {
	key, ok := c.roster.Key(node)
	switch {
	case !ok:
		return connect.NewClient(refuse{notHolder(held, "node "+node.String()+" is not in the roster")})
	case held != "" && !c.roster.Serves(held, node):
		return connect.NewClient(refuse{notHolder(held, "node "+node.String()+" does not hold the "+string(held)+" Role")})
	}
	return c.d.client(key)
}

// refuse fails every call before sending. Its error is marked remote, as a
// peer's would be, so a handler returning it unchanged fails its own call
// with CodeInternal instead of passing on NotHolder.
type refuse struct{ err *connect.Error }

func (r refuse) NewClientStream(context.Context, connect.Spec) (connect.ClientStream, error) {
	return nil, r.err.WithRemote()
}

func notHolder(held role.Name, message string) *connect.Error {
	detail, err := connectproto.NewErrorDetail(&nodev1.NotHolder{Role: string(held)})
	if err != nil {
		panic(err)
	}
	return connect.NewError(connect.CodeFailedPrecondition, message).WithDetail(detail)
}
