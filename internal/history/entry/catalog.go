package entry

import (
	"fmt"
	"slices"
	"sync"
)

// Catalog is every row an entry can be recorded or rendered from, in
// registration order. Owners register while the node wires the Controller,
// and the node seals it before any runner starts.
type Catalog struct {
	mu     sync.Mutex
	rows   []Entry
	sealed bool
}

func NewCatalog() *Catalog { return &Catalog{} }

// Register adds owner's rows. It panics on a kind already registered, a
// row with no History group, and after Seal.
func (c *Catalog) Register(owner string, rows []Entry) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.sealed {
		panic(fmt.Sprintf("entry: %s registered its kinds after the catalog was sealed", owner))
	}
	for _, e := range rows {
		spec := e.Declared()
		if !slices.Contains(Groups(), spec.Group) {
			panic(fmt.Sprintf("entry: %s registers %s with no History group", owner, spec.Kind))
		}
		if slices.ContainsFunc(c.rows, func(r Entry) bool { return r.Declared().Kind == spec.Kind }) {
			panic(fmt.Sprintf("entry: %s registers %s, which is already registered", owner, spec.Kind))
		}
		c.rows = append(c.rows, e)
	}
}

// Seal ends registration.
func (c *Catalog) Seal() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sealed = true
}

// Specs is every registered kind, in registration order.
func (c *Catalog) Specs() []Spec {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]Spec, len(c.rows))
	for i, e := range c.rows {
		out[i] = e.Declared()
	}
	return out
}

// Kinds is g's kinds, in registration order.
func (c *Catalog) Kinds(g Group) []Kind {
	var out []Kind
	for _, s := range c.Specs() {
		if s.Group == g {
			out = append(out, s.Kind)
		}
	}
	return out
}

// Spec is k's declaration, false for a kind no row has, such as one a
// later build recorded.
func (c *Catalog) Spec(k Kind) (Spec, bool) {
	e, ok := c.row(k)
	if !ok {
		return Spec{}, false
	}
	return e.Declared(), true
}

// Render is a stored subject of kind k in words.
func (c *Catalog) Render(k Kind, subject []byte, ctx Context) (Message, error) {
	e, ok := c.row(k)
	if !ok {
		return Message{}, fmt.Errorf("render a %s entry: no catalog row has its kind", k)
	}
	return e.RenderStored(subject, ctx)
}

func (c *Catalog) row(k Kind) (Entry, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	i := slices.IndexFunc(c.rows, func(e Entry) bool { return e.Declared().Kind == k })
	if i < 0 {
		return nil, false
	}
	return c.rows[i], true
}
