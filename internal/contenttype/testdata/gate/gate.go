// Package gate grows contenttype.Type by a second facet, as a later spec
// does, over a type that lacks it.
package gate

import (
	"github.com/jatassi/slipstream/internal/contenttype"
	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
)

type Naming interface {
	Name() string
}

type Type interface {
	contenttype.Type
	Naming
}

var _ Type = contenttypetest.New()
