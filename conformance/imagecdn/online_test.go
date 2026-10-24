//go:build online

package imagecdn_test

import (
	"net/http"
	"testing"
	"time"

	conformance "github.com/jatassi/slipstream/conformance/imagecdn"
)

// fightClub is Fight Club's poster; netflix is Netflix's network logo,
// which TMDB keeps as both PNG and SVG.
var (
	fightClub  = conformance.Known{Path: "/pB8BM7pdSp6B6Ih7QZ4DrQ3PmJK.jpg", Type: "image/jpeg"}
	netflixPNG = conformance.Known{Path: "/wwemzKWzjKYJFfCeiB57q3r4Bcm.png", Type: "image/png"}
	netflixSVG = conformance.Known{Path: "/wwemzKWzjKYJFfCeiB57q3r4Bcm.svg", Type: "image/svg+xml"}
)

func TestImageCDNConforms(t *testing.T) {
	conformance.Run(t, conformance.Target{
		Images: "https://image.tmdb.org/t/p",
		Client: &http.Client{Timeout: time.Minute},
		Known:  []conformance.Known{fightClub, netflixPNG, netflixSVG},
	})
}
