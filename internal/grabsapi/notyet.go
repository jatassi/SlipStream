package grabsapi

import (
	"net/http"

	"github.com/jatassi/slipstream/internal/api"
)

// notYet answers an operation whose wire is settled and whose handler
// isn't built. TestNotYetLedger lists each one.
var notYet = api.NewCode(http.StatusNotImplemented, "not_yet", "SlipStream doesn't do this yet")
