package downloadclients_test

import (
	"testing"

	dc "github.com/jatassi/slipstream/internal/downloadclients"
)

func TestRedact(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"https://tracker.example/announce/0123456789abcdef0123456789abcdef", "https://tracker.example/announce/•••"},
		{"https://tracker.example/announce/s3cret/extra", "https://tracker.example/announce/•••/•••"},
		{"https://tracker.example:2053/0123456789ABCDEF0123456789ABCDEF/announce", "https://tracker.example:2053/•••/announce"},
		{"https://tracker.example/announce.php?passkey=s3cret&info=1", "https://tracker.example/announce.php?passkey=•••&info=1"},
		{"https://tracker.example/announce?pk=a&authkey=b&torrent_pass=c&uid=7", "https://tracker.example/announce?pk=•••&authkey=•••&torrent_pass=•••&uid=7"},
		{"https://tracker.example/announce?PassKey=s3cret", "https://tracker.example/announce?PassKey=•••"},
		{"https://user:s3cret@tracker.example/announce", "https://user:•••@tracker.example/announce"},
		{"udp://tracker.opentrackr.org:1337/announce", "udp://tracker.opentrackr.org:1337/announce"},
		{"http://tracker.example/announce", "http://tracker.example/announce"},
		{"not a url", "•••"},
		{"https://tracker.example/%zz", "•••"},
	} {
		if got := dc.Redact(c.in); got != c.want {
			t.Errorf("Redact(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
