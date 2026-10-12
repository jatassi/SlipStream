// Package api is SlipStream's HTTP mechanism and the only package that
// imports Huma. It registers no product operation.
//
// # Adding an operation
//
// An owner keeps its operations in a sibling package <owner>api, in one
// function named Routes:
//
//	func Routes(r *api.Router, nodes *nodes.Service) {
//		o := r.Owner("nodes", api.Admin)
//		api.Register(o, api.Admin, api.Op{
//			ID: "listNodes", Method: http.MethodGet, Path: "/nodes", Summary: "List the Nodes",
//		}, func(ctx context.Context, a identity.Admin, _ *struct{}) (*listOutput, error) {
//			...
//		})
//	}
//
// The level decides the principal the handler receives: api.Public gives
// identity.Anyone, api.Setup identity.Claim, api.Account identity.Account
// and api.Admin identity.Admin. A domain method only an Admin may run takes
// the identity.Admin too.
//
// # GET and HEAD have no side effects
//
// A GET or HEAD operation changes nothing but a session's last_seen_at
// slide. CSRF protection admits safe methods from any origin and relies on
// this; Register refuses a GET with a request body, but no check can see a
// side effect.
//
// # Errors
//
// An operation answers its errors as Codes it lists in its Op. Any other
// error, an unlisted Code or a panic answers 500 internal and logs the real
// error with the operation id.
package api
