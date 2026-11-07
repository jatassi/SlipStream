package series

import (
	"fmt"
	"strconv"

	"github.com/jatassi/slipstream/internal/providers/tvdb"
)

// code is how the many-Items layout names an episode in order: S01E03, or
// in Absolute order its number in that order, which is its absolute
// number. A special keeps S00E01 in Absolute order, which numbers only
// the main run.
func code(order tvdb.Order, season, episode int64) string {
	if order == tvdb.OrderAbsolute && season > 0 {
		return strconv.FormatInt(episode, 10)
	}
	return fmt.Sprintf("S%02dE%02d", season, episode)
}
