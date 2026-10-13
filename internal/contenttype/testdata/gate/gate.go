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
