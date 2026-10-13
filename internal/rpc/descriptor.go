// Package rpc is call dispatch: a Role calls a service through the
// descriptor protoc-gen-slipstream emits from the service's served_by
// option, and gets the local implementation or a link client of the same
// handler interface H.
package rpc

import "connectrpc.com/connect/v2"

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
	role string
}

func NewControllerService[H any](name string, client func(*connect.Client) H, register func(*connect.Server, H)) ControllerService[H] {
	return ControllerService[H]{service[H]{name, client, register}}
}

func NewNodeService[H any](name string, client func(*connect.Client) H, register func(*connect.Server, H)) NodeService[H] {
	return NodeService[H]{service[H]{name, client, register}}
}

func NewRoleService[H any](role, name string, client func(*connect.Client) H, register func(*connect.Server, H)) RoleService[H] {
	return RoleService[H]{service[H]{name, client, register}, role}
}
