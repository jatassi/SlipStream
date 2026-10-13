// Package rpc is call dispatch: a Role calls a service through the
// descriptor protoc-gen-slipstream emits from the service's served_by
// option, and gets the local implementation or a link client of the same
// handler interface H.
package rpc

import (
	"connectrpc.com/connect/v2"

	"github.com/jatassi/slipstream/internal/role"
)

type service[H any] struct {
	name     string
	client   func(*connect.Client) H
	register func(*connect.Server, H)
}

// ControllerService is a service the Controller serves and every claimed
// Node calls.
type ControllerService[H any] struct{ service[H] }

// NodeService is a service every Node serves and only the Controller calls.
type NodeService[H any] struct{ service[H] }

// RoleService is a service the holders of one Role serve and only the
// Controller calls.
type RoleService[H any] struct {
	service[H]
	role role.Name
}

func NewControllerService[H any](name string, client func(*connect.Client) H, register func(*connect.Server, H)) ControllerService[H] {
	return ControllerService[H]{service[H]{name, client, register}}
}

func NewNodeService[H any](name string, client func(*connect.Client) H, register func(*connect.Server, H)) NodeService[H] {
	return NodeService[H]{service[H]{name, client, register}}
}

func NewRoleService[H any](r role.Name, name string, client func(*connect.Client) H, register func(*connect.Server, H)) RoleService[H] {
	return RoleService[H]{service[H]{name, client, register}, r}
}

// Call is the Controller's implementation: local on the Controller, a link
// client elsewhere.
func (s ControllerService[H]) Call(d *Dispatch) H {
	return s.client(d.client(d.controller))
}

// Mount serves impl on the Controller.
func (s ControllerService[H]) Mount(c *Cluster, impl H) {
	c.d.mount(s.name, "", func(srv *connect.Server) { s.register(srv, impl) })
}

// On is node's implementation. A node the roster does not know fails every
// call with NotHolder.
func (s NodeService[H]) On(c *Cluster, node role.NodeID) H {
	return s.client(c.client(node, ""))
}

// Mount serves impl on this Node.
func (s NodeService[H]) Mount(d *Dispatch, impl H) {
	d.mount(s.name, "", func(srv *connect.Server) { s.register(srv, impl) })
}

// Holders is every Node the roster says holds the Role.
func (s RoleService[H]) Holders(c *Cluster) []role.NodeID {
	return c.roster.Holders(s.role)
}

// On is node's implementation. A node the roster says does not hold the
// Role fails every call with NotHolder, before sending.
func (s RoleService[H]) On(c *Cluster, node role.NodeID) H {
	return s.client(c.client(node, s.role))
}

// Mount serves impl on this Node. A call arriving while this Node does not
// hold the Role is refused with NotHolder before impl runs.
func (s RoleService[H]) Mount(d *Dispatch, impl H) {
	d.mount(s.name, s.role, func(srv *connect.Server) { s.register(srv, impl) })
}
