package identity

import "strings"

type uaMark struct{ marks, name string }

// browsers and systems are checked in order, so a browser that also says
// "Chrome" or "Safari" in its User-Agent comes before them.
var (
	browsers = []uaMark{
		{"Edg/|EdgiOS/|EdgA/", "Edge"},
		{"OPR/|OPiOS/", "Opera"},
		{"SamsungBrowser/", "Samsung Internet"},
		{"Firefox/|FxiOS/", "Firefox"},
		{"Chrome/|CriOS/|Chromium/", "Chrome"},
		{"Safari/", "Safari"},
		{"curl/", "curl"},
	}
	systems = []uaMark{
		{"iPhone", "iPhone"},
		{"iPad", "iPad"},
		{"Android", "Android"},
		{"Windows", "Windows"},
		{"CrOS", "ChromeOS"},
		{"Macintosh|Mac OS X", "Mac"},
		{"Linux", "Linux"},
	}
)

func (m uaMark) in(ua string) bool {
	for mark := range strings.SplitSeq(m.marks, "|") {
		if strings.Contains(ua, mark) {
			return true
		}
	}
	return false
}

func firstIn(table []uaMark, ua string) string {
	for _, m := range table {
		if m.in(ua) {
			return m.name
		}
	}
	return ""
}

// deviceLabel names the device a session signed in from, such as "Safari
// on iPhone", from its User-Agent at sign-in.
func deviceLabel(userAgent string) string {
	browser, system := firstIn(browsers, userAgent), firstIn(systems, userAgent)
	switch {
	case browser == "" && system == "":
		return "A browser"
	case system == "":
		return browser
	case browser == "":
		return "A browser on " + system
	}
	return browser + " on " + system
}
