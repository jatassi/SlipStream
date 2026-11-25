package mediaserversapi_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"testing/synctest"
)

// TestNoMemberSecretLeaves runs each Member path that carries a secret
// with a sentinel in place of it, and reads every response, event payload
// and log line, at debug, of the Controller, its Nodes and the API: none
// holds the sentinel.
func TestNoMemberSecretLeaves(t *testing.T) {
	const sentinel = "sentinel-7f3a9c"
	for _, c := range []struct {
		name string
		// run answers the response bodies the path got.
		run func(t *testing.T, d denRig) []string
	}{
		{"a password made through seedbox", func(t *testing.T, d denRig) []string {
			_, body := d.makeAs(t, d.sam, d.row, "sam", sentinel)
			return []string{body}
		}},
		{"a password Den refused with the name", func(t *testing.T, d denRig) []string {
			_, body := d.makeAs(t, d.sam, d.row, "ada", sentinel)
			return []string{body}
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				d := newDenRigOn(t, newRig(t), "http://192.168.1.99:8096")
				tap := d.w.Tap(t.Context())
				bodies := c.run(t, d)
				synctest.Wait()
				bodies = append(bodies, d.expectOK(t, servers+"/access"))
				_, mine := d.o.CallAs(t, http.MethodGet, "/api/v1/account/media-servers", "", d.sam.Session)
				bodies = append(bodies, mine)
				msgs, _ := tap.Take()
				for _, m := range msgs {
					payload, err := json.Marshal(m.Data)
					if err != nil {
						t.Fatal(err)
					}
					bodies = append(bodies, m.Name+" "+string(payload))
				}
				for _, where := range append(bodies, d.w.Logged(), d.o.Log.String()) {
					if strings.Contains(where, sentinel) {
						t.Errorf("the sentinel leaked in %s", where)
					}
				}
			})
		})
	}
}
