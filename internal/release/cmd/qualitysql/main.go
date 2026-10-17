// Command qualitysql prints release.QualitySQL for a migration to paste.
package main

import (
	"flag"
	"fmt"

	"github.com/jatassi/slipstream/internal/release"
)

func main() {
	title := flag.String("title", "title", "the SQL expression of the release title")
	width := flag.String("width", "width", "the SQL expression of the probed width")
	height := flag.String("height", "height", "the SQL expression of the probed height")
	flag.Parse()
	fmt.Println(release.QualitySQL(*title, *width, *height))
}
