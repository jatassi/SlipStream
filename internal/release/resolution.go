package release

// Resolution ranks by value: a lower one is the higher resolution.
type Resolution uint8

const (
	R2160p Resolution = iota + 1
	R1080p
	R720p
	R576p
	R540p
	R480p
	R360p
)

var resolutionNames = []string{R2160p: "2160p", R1080p: "1080p", R720p: "720p", R576p: "576p", R540p: "540p", R480p: "480p", R360p: "360p"}

func (r Resolution) String() string               { return resolutionNames[r] }
func (r Resolution) MarshalText() ([]byte, error) { return []byte(r.String()), nil }
func (r *Resolution) UnmarshalText(b []byte) error {
	return parseKey(r, "a resolution", resolutionNames, b)
}

// classes are the smallest frame of each resolution but the lowest. The
// width counts too, so a cropped video keeps its class.
var classes = []struct {
	res           Resolution
	width, height int
}{
	{R2160p, 3200, 2000},
	{R1080p, 1800, 1000},
	{R720p, 1200, 700},
	{R576p, 1000, 560},
}

// ResolutionOf is the resolution class of a width by height video.
func ResolutionOf(width, height int) Resolution {
	for _, c := range classes {
		if width >= c.width || height >= c.height {
			return c.res
		}
	}
	return R480p
}
