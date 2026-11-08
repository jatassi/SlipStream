package filelist

import (
	"encoding/base64"
	"net/url"
	"testing"

	"github.com/jatassi/slipstream/internal/ports/port"
	"github.com/jatassi/slipstream/internal/prowlarr/stringutil"
)

func TestEveryRequestSignsWithTheUsernameAndPasskey(t *testing.T) {
	base, _ := url.Parse("https://filelist.fixture-site.fake.localhost/")
	ix := &indexer{
		Settings:   fileListSettings{Username: " fixture-user ", PasskeyField: port.PasskeyField{Passkey: "fixture-passkey "}, FreeleechOnly: true},
		BaseURL:    base,
		Categories: Port.Entry().Capabilities.Categories,
	}
	r := getPagedRequests(ix, port.SearchCriteria{}, &stringutil.NameValueCollection{})
	want := "Basic " + base64.StdEncoding.EncodeToString([]byte("fixture-user:fixture-passkey"))
	if got := r.Header.Get("Authorization"); got != want {
		t.Errorf("Authorization = %q, want %q", got, want)
	}
	if q := r.URL.Query(); q.Get("action") != "latest-torrents" || q.Get("freeleech") != "1" {
		t.Errorf("query = %v, want action=latest-torrents and freeleech=1", q)
	}
}
