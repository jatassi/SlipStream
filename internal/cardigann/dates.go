package cardigann

import (
	"strconv"
	"time"

	"github.com/jatassi/slipstream/internal/prowlarr/datetimeutil"
)

// Clock is the current time and the zone a date without an offset is read
// in: Prowlarr's DateTime.Now and its host's local zone.
type Clock = datetimeutil.Clock

// formatRFC1123Z writes t as Prowlarr's Rfc1123ZPattern,
// "ddd, dd MMM yyyy HH':'mm':'ss z", the form dateparse, timeago and
// fuzzytime leave a value in. The offset is whole hours, as .NET's z writes
// it.
func formatRFC1123Z(t time.Time) string {
	_, off := t.Zone()
	sign := "+"
	if off < 0 {
		sign, off = "-", -off
	}
	return t.Format("Mon, 02 Jan 2006 15:04:05") + " " + sign + strconv.Itoa(off/3600)
}
