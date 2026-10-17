package mail

import "embed"

//go:embed test.html.tmpl test.txt.tmpl
var testFiles embed.FS

var testMail = MustTemplate[TestEmail](testFiles, "test")

// TestEmail is Send test email's message. ServerName is the address's
// host and URL the address, each empty while no address is set.
type TestEmail struct {
	ServerName string
	URL        string
}

func TestMessage(to Address, e TestEmail) (Message, error) { return testMail.Render(to, e) }
