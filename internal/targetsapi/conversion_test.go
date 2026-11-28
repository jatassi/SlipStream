package targetsapi_test

import (
	"encoding/json"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/api/ownertest"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/targetsapi"
)

func TestTheProblemCodesAreTargetsProblemCodes(t *testing.T) {
	f, _ := reflect.TypeFor[targetsapi.Problem]().FieldByName("Code")
	var want []string
	for _, c := range targets.ProblemCodes() {
		want = append(want, string(c))
	}
	if got := f.Tag.Get("enum"); got != strings.Join(want, ",") {
		t.Errorf("Problem.Code enum = %q, want %q", got, strings.Join(want, ","))
	}
}

// mobile is a 720p Target body whose Conversion settings are settings.
func mobile(settings string) string {
	return `{"name":"720p mobile","tiers":[["bluray-720p","webdl-720p"]],"cutoff":"webdl-720p","conversion":{` + settings + `}}`
}

const mobileSettings = `"codec":"h264","maxHeight":720,"kbps":2000,"depth":8,"range":"sdr","keepDolbyVision":false,"curve":"bt2390",` +
	`"audio":"stereo_aac","dropCommentary":true,"container":"mkv","runOnNodeId":null`

type problem struct {
	Code     string `json:"code"`
	Errors   []struct{ Location string }
	Problems []targetsapi.Problem `json:"problems"`
	Profiles []string             `json:"profiles"`
	Impact   *struct {
		Made struct{ Files int } `json:"made"`
	} `json:"impact"`
}

func problemOf(t *testing.T, reply string) problem {
	t.Helper()
	var p problem
	if err := json.Unmarshal([]byte(reply), &p); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestConversionSettingsAreSavedOrRefusedWithTheirProblem(t *testing.T) {
	s := serve(t)
	for _, c := range []struct {
		name, settings string
		code, location string
		problems       []string
	}{
		{"a Run on Node without the Converting Role", strings.Replace(mobileSettings, `"runOnNodeId":null`, `"runOnNodeId":"`+storage.NewUUID().String()+`"`, 1), "invalid", "body.conversion.runOnNodeId", nil},
		{"10 bit H.264", strings.Replace(mobileSettings, `"depth":8`, `"depth":10`, 1), "invalid", "body.conversion.depth", nil},
		{"a bitrate below every 720p size", strings.Replace(mobileSettings, `"kbps":2000`, `"kbps":200`, 1), "conversion_refused", "", []string{"outside_size kbps"}},
	} {
		status, reply := s.Call(t, http.MethodPost, "/api/v1/targets", mobile(c.settings))
		p := problemOf(t, reply)
		if status/100 != 4 || p.Code != c.code {
			t.Errorf("%s = %d %s, want %s", c.name, status, reply, c.code)
			continue
		}
		if c.location != "" && (len(p.Errors) != 1 || p.Errors[0].Location != c.location) {
			t.Errorf("%s refused at %+v, want %s", c.name, p.Errors, c.location)
		}
		var got []string
		for _, pr := range p.Problems {
			got = append(got, pr.Code+" "+pr.Field)
		}
		if strings.Join(got, ";") != strings.Join(c.problems, ";") {
			t.Errorf("%s has problems %v, want %v", c.name, got, c.problems)
		}
	}

	status, reply := s.Call(t, http.MethodPost, "/api/v1/targets", mobile(mobileSettings))
	var made struct {
		Conversion *struct {
			Codec       string `json:"codec"`
			Kbps        int    `json:"kbps"`
			Older       int    `json:"older"`
			DefaultKbps int    `json:"defaultKbps"`
		} `json:"conversion"`
	}
	if err := json.Unmarshal([]byte(reply), &made); status != http.StatusCreated || err != nil || made.Conversion == nil ||
		made.Conversion.Codec != "h264" || made.Conversion.Kbps != 2000 || made.Conversion.Older != 0 {
		t.Fatalf("saving the settings = %d %s, want them back", status, reply)
	}
	// webdl-720p's floor is 12.5 MiB per minute with no maximum: 12.5 ×
	// 1.25 × 139.81 kbps.
	if made.Conversion.DefaultKbps != 2185 {
		t.Errorf("defaultKbps = %d, want 2185", made.Conversion.DefaultKbps)
	}
}

// A Profile that adds a made-from Target previews the files it makes,
// refuses a save past the count confirmed with the Impact it has now, and
// keeps the Target's Conversion settings on while it makes from it.
func TestAMadeFromRowMakesFilesAndHoldsItsSettings(t *testing.T) {
	s, ids := shelved(t)
	_, reply := s.Call(t, http.MethodPost, "/api/v1/targets", mobile(mobileSettings))
	ids["mobile"] = ownertest.FirstID(reply)
	path := "/api/v1/profiles/" + ids["fixture"]
	body := fill(`{"name":"Fixture","contentType":"fixture","targets":[{"targetId":"@starter","sourcing":"download"},`+
		`{"targetId":"@mobile","sourcing":"derive","fromTargetId":"@starter"}],"editions":{"kind":"any"},"languages":[]}`, ids)

	var preview struct {
		Impact struct {
			Made struct{ Files int } `json:"made"`
		} `json:"impact"`
	}
	status, reply := s.Call(t, http.MethodPut, path+"?preview=true", body)
	if err := json.Unmarshal([]byte(reply), &preview); status != http.StatusOK || err != nil || preview.Impact.Made.Files != 1 {
		t.Fatalf("the preview = %d %s, want 1 file made", status, reply)
	}
	status, reply = s.Call(t, http.MethodPut, path+"?confirmMade=0", body)
	if p := problemOf(t, reply); status != http.StatusConflict || p.Code != "impact_changed" || p.Impact == nil || p.Impact.Made.Files != 1 {
		t.Errorf("a save past confirmMade=0 = %d %s, want 409 impact_changed carrying 1 made file", status, reply)
	}
	if status, reply := s.Call(t, http.MethodPut, path+"?confirmMade=1", body); status != http.StatusOK {
		t.Fatalf("the confirmed save = %d %s, want 200", status, reply)
	}

	status, reply = s.Call(t, http.MethodGet, path, "")
	if want := fill(`{"targetId":"@mobile","sourcing":"derive","fromTargetId":"@starter"}`, ids); status != http.StatusOK || !strings.Contains(reply, want) {
		t.Errorf("the Profile = %s, want its made-from row %s", reply, want)
	}
	status, reply = s.Call(t, http.MethodPut, "/api/v1/targets/"+ids["mobile"], `{"name":"720p mobile","tiers":[["bluray-720p","webdl-720p"]],"cutoff":"webdl-720p"}`)
	if p := problemOf(t, reply); status != http.StatusConflict || p.Code != "target_is_a_source" || strings.Join(p.Profiles, ",") != "Fixture" {
		t.Errorf("turning off the settings = %d %s, want 409 target_is_a_source naming Fixture", status, reply)
	}
}
