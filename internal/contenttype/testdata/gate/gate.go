package gate

import (
	"github.com/jatassi/slipstream/internal/contenttype"
	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
)

type Naming interface {
	Name() string
}

type Library interface {
	contenttype.Library
	Naming
}

var _ Library = contenttypetest.New()
