package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"maps"
	"net/http"
	"path"
	"reflect"
	"regexp"
	"runtime/debug"
	"slices"
	"strconv"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"

	"github.com/jatassi/slipstream/internal/identity"
)

const (
	// Prefix is where the operations live; Op paths are relative to it.
	Prefix = "/api/v1"
	// version is info.version, constant so a stamped and an unstamped build
	// produce the same document.
	version = "1"

	defaultBodyLimit = 2 << 20
	accessExtension  = "x-slipstream-access"
	schemaPrefix     = "#/components/schemas/"
)

// Services are what serving needs. The document needs none of them.
type Services struct {
	Identity       *identity.Service
	Gate           *identity.Gate
	Web            fs.FS
	TrustedOrigins []Origin
	Log            *slog.Logger
}

// Router holds every registered operation. Owners register through Owner,
// then the Node serves Handler and surface renders Document.
type Router struct {
	services Services
	huma     huma.API
	mux      *http.ServeMux
	owners   map[string]bool
	ops      map[string]string
}

func NewRouter(s Services) *Router {
	mux := http.NewServeMux()
	registry := huma.NewMapRegistry(schemaPrefix, schemaName)
	h := huma.NewAPI(huma.Config{
		OpenAPI: &huma.OpenAPI{
			OpenAPI: "3.1.0",
			Info:    &huma.Info{Title: "SlipStream", Version: version},
			Servers: []*huma.Server{{URL: Prefix}},
			Components: &huma.Components{
				Schemas: registry,
				SecuritySchemes: map[string]*huma.SecurityScheme{
					schemeSession:   {Type: "apiKey", In: "cookie", Name: identity.CookieName},
					schemeAPIKey:    {Type: "apiKey", In: "header", Name: apiKeyHeader},
					schemeSetupCode: {Type: "apiKey", In: "header", Name: setupCodeHeader},
				},
			},
		},
		Formats:       huma.DefaultFormats,
		DefaultFormat: "application/json",
	}, humago.NewAdapter(mux, Prefix))
	registry.Schema(reflect.TypeFor[Problem](), true, "Problem")
	return &Router{services: s, huma: h, mux: mux, owners: map[string]bool{}, ops: map[string]string{}}
}

var tagShape = regexp.MustCompile(`^[a-z][a-z0-9]*$`)

// Owner is an owner's handle on the router. tag names the owner and is the
// tag of its operations; floor is the lowest level they may use.
func (r *Router) Owner(tag string, floor Floor) *Owner {
	if !tagShape.MatchString(tag) {
		panic(fmt.Sprintf("api: owner tag %q must match %s", tag, tagShape))
	}
	if floor.level() == nil {
		panic("api: owner " + tag + ": its floor is " + zeroLevel)
	}
	if r.owners[tag] {
		panic("api: owner tag " + tag + " is used by two owners")
	}
	r.owners[tag] = true
	return &Owner{router: r, tag: tag, floor: floor.level()}
}

type Owner struct {
	router *Router
	tag    string
	floor  *access
}

// Op declares an operation.
type Op struct {
	// ID is lowerCamelCase and verb-first: listNodes, getSystem.
	ID string
	// Method is an http.Method* constant.
	Method string
	// Path is relative to Prefix: /nodes/{id}.
	Path    string
	Summary string
	// Status is the success status.
	Status int
	// Errors are the owner's codes the handler may return, beyond its
	// level's codes, invalid and internal.
	Errors []*Code
	// MaxBody raises the 2 MiB body limit for an operation that needs more.
	MaxBody int64
}

var idShape = regexp.MustCompile(`^[a-z]+[A-Z][A-Za-z0-9]*$`)

type principalKey struct{}

// Register adds an operation. Its handler receives the principal its
// level admits. Register panics at start on an id that is not
// lowerCamelCase or is taken, a level below the owner's floor, an unlimited
// body, or a GET or HEAD with a request body.
func Register[P, I, O any](o *Owner, level Level[P], op Op, handler func(context.Context, P, *I) (*O, error)) {
	r := o.router
	hop, declared := humaOperation(o, level, op, reflect.TypeFor[I]())
	huma.Register(r.huma, hop, func(ctx context.Context, in *I) (out *O, err error) {
		p := principal[P](ctx, op)
		defer func() {
			if v := recover(); v != nil {
				r.logPanic(op, v)
				out, err = nil, Internal.problem()
			}
		}()
		out, err = handler(ctx, p, in)
		if err != nil {
			return nil, r.failure(op, declared, err)
		}
		return out, nil
	})
}

// humaOperation is op as Huma registers it, behind level's access check,
// and every code it may answer.
func humaOperation[P any](o *Owner, level Level[P], op Op, input reflect.Type) (huma.Operation, []*Code) {
	o.check(level.access, op, input)
	declared := slices.Concat(level.codes, op.Errors, []*Code{Internal})
	if input.NumField() > 0 {
		declared = append(declared, Invalid)
	}
	limit := op.MaxBody
	if limit == 0 {
		limit = defaultBodyLimit
	}
	return huma.Operation{
		OperationID:   op.ID,
		Method:        op.Method,
		Path:          op.Path,
		Summary:       op.Summary,
		Tags:          []string{o.tag},
		DefaultStatus: op.Status,
		Security:      level.security,
		Extensions:    map[string]any{accessExtension: level.name},
		Responses:     errorResponses(declared),
		// Huma answers 413 when it reads exactly MaxBodyBytes, so the limit
		// itself is MaxBodyBytes-1.
		MaxBodyBytes: limit + 1,
		Middlewares:  huma.Middlewares{admitMiddleware(o.router, level)},
	}, declared
}

func principal[P any](ctx context.Context, op Op) P {
	p, ok := ctx.Value(principalKey{}).(P)
	if !ok {
		panic("api: " + op.ID + " ran without its access check")
	}
	return p
}

func (r *Router) logPanic(op Op, v any) {
	r.services.Log.Error("operation panicked", "operation", op.ID, "method", op.Method, "path", Prefix+op.Path, "panic", v, "stack", string(debug.Stack()))
}

func (o *Owner) check(level *access, op Op, input reflect.Type) {
	r := o.router
	if level == nil {
		panic(op.ID + ": " + zeroLevel)
	}
	if !idShape.MatchString(op.ID) {
		panic(fmt.Sprintf("api: operation id %q is not lowerCamelCase and verb-first", op.ID))
	}
	route := fmt.Sprintf("%s %s%s by owner %s", op.Method, Prefix, op.Path, o.tag)
	if other, taken := r.ops[op.ID]; taken {
		panic(fmt.Sprintf("api: operation id %s is registered twice: %s and %s", op.ID, other, route))
	}
	r.ops[op.ID] = route
	if level.rank < o.floor.rank {
		var admitted []string
		for _, l := range levels[o.floor.rank:] {
			admitted = append(admitted, l.name)
		}
		panic(fmt.Sprintf("%s: owner %s admits %s operations only; level is %s", op.ID, o.tag, strings.Join(admitted, " and "), level.name))
	}
	if op.MaxBody < 0 {
		panic(op.ID + ": an operation's body limit is never unlimited")
	}
	_, body := input.FieldByName("Body")
	_, raw := input.FieldByName("RawBody")
	if (op.Method == http.MethodGet || op.Method == http.MethodHead) && (body || raw) {
		panic(op.ID + ": a " + op.Method + " operation takes no request body")
	}
}

func admitMiddleware[P any](r *Router, level Level[P]) func(huma.Context, func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		c, ok := ctx.Context().Value(callerKey{}).(identity.Caller)
		if !ok {
			panic("api: an operation ran without authentication")
		}
		err := level.admit(ctx.Context(), r.services.Gate, c, ctx.Header(setupCodeHeader), func(p P) {
			next(huma.WithValue(ctx, principalKey{}, p))
		})
		if err == nil {
			return
		}
		code, ok := identityCode(err)
		if !ok {
			panic(fmt.Sprintf("api: %s refused with %v, which maps to no code", ctx.Operation().OperationID, err))
		}
		_, w := humago.Unwrap(ctx)
		writeProblem(w, code.problem())
	}
}

// failure turns a handler's error into what the client sees. A code the
// operation declares answers as itself; anything else answers internal
// and logs the real error.
func (r *Router) failure(op Op, declared []*Code, err error) error {
	attrs := []any{"method", op.Method, "path", Prefix + op.Path}
	code, isCode := err.(*Code) //nolint:errorlint // a handler returns its code itself, never wrapped
	switch {
	case isCode && slices.Contains(declared, code):
		return code.problem()
	case isCode:
		r.services.Log.Error("operation "+op.ID+" returned undeclared code "+code.name, attrs...)
	default:
		r.services.Log.Error("operation failed", append(attrs, "operation", op.ID, "error", err)...)
	}
	return Internal.problem()
}

// errorResponses documents each status's codes, so a client's generated
// types know exactly which codes an operation answers.
func errorResponses(codes []*Code) map[string]*huma.Response {
	byStatus := map[int][]any{}
	for _, c := range codes {
		if !slices.Contains(byStatus[c.status], any(c.name)) {
			byStatus[c.status] = append(byStatus[c.status], c.name)
		}
	}
	responses := map[string]*huma.Response{}
	for _, status := range slices.Sorted(maps.Keys(byStatus)) {
		names := byStatus[status]
		slices.SortFunc(names, func(a, b any) int { return strings.Compare(fmt.Sprint(a), fmt.Sprint(b)) })
		responses[strconv.Itoa(status)] = &huma.Response{
			Description: http.StatusText(status),
			Content: map[string]*huma.MediaType{problemType: {Schema: &huma.Schema{AllOf: []*huma.Schema{
				{Ref: schemaPrefix + "Problem"},
				{Type: "object", Properties: map[string]*huma.Schema{"code": {Type: "string", Enum: names}}},
			}}}},
		}
	}
	return responses
}

// schemaName prefixes Huma's name with the owner a type belongs to, read
// from its package with any api suffix dropped: identityapi.Me becomes
// IdentityMe. Two owners' types never share a name.
func schemaName(t reflect.Type, hint string) string {
	name := huma.DefaultSchemaNamer(t, hint)
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Name() == "" || t.PkgPath() == "" {
		return name
	}
	owner := strings.TrimSuffix(path.Base(t.PkgPath()), "api")
	if owner == "" {
		return name
	}
	return strings.ToUpper(owner[:1]) + owner[1:] + name
}

// Document is the OpenAPI document: OpenAPI 3.1 as indented JSON with a
// trailing newline, the same bytes on every call.
func (r *Router) Document() []byte {
	b, err := json.MarshalIndent(r.huma.OpenAPI(), "", "  ")
	if err != nil {
		panic(fmt.Sprintf("api: render the OpenAPI document: %v", err))
	}
	return append(b, '\n')
}
