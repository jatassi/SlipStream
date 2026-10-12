package tmdb

const api = "https://api.themoviedb.org/3"

var hosts = []string{
	"udp://tracker.opentrackr.org:1337/announce",
	"stun:stun.l.google.com:19302",
	"TURNS:relay.slipstream.app",
	"router.bittorrent.com:6881",
	"8.8.8.8:53",
	"[2001:db8::1]:443",
	"see https://user:pw@Docs.SlipStream.app/help for help",
	`ws://tracker/announce`,
}

var exempt = []string{
	"http://localhost:8080",
	"http://app.localhost/",
	"127.0.0.2:80",
	"[::1]:9000",
	"0.0.0.0:6881",
	"[::]:0",
	"[::ffff:127.0.0.1]:80",
	"https://example.com/x",
	"api.example.org:443",
	"svc.test:80",
	"stun:stun.example:3478",
	"http://host.invalid",
	"turn:%s",
	"github.com/jatassi/slipstream",
	"https://%s/api",
	"file:///etc/hosts",
	"15:04:05",
	"1.2:3",
	"return: fmt.Errorf",
	"tracker:6969",
}
