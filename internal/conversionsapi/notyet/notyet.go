// Package notyet answers what S44 (Conversion) declares on the wire before
// its build serves it: an operation, and a request field set to anything
// but its default.
package notyet

import (
	"net/http"

	"github.com/jatassi/slipstream/internal/api"
)

var (
	Op    = api.NewCode(http.StatusNotImplemented, "not_built_yet", "SlipStream doesn't do this yet")
	Field = api.NewCode(http.StatusUnprocessableEntity, "not_built_yet", "SlipStream can't save this setting yet")
)
