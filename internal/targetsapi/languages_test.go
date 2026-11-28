package targetsapi_test

import (
	"net/http"
	"strings"
	"testing"
)

// withLanguages is profileBody of Profile name holding the Targets with
// ids, with languages as its JSON array.
func withLanguages(name, languages string, ids ...string) string {
	return strings.Replace(profileBody(name, ids...), `"languages":[]`, `"languages":`+languages, 1)
}

const (
	english       = `{"language":"en","forced":false,"hearingImpaired":"allow","audio":"always"}`
	forcedFrench  = `{"language":"fra","forced":true,"hearingImpaired":"avoid","audio":"no_audio"}`
	noLanguages   = `"subtitles":{"gained":[],"dropped":[],"days":0}`
	untouched     = `{"works":0,"ownList":0,"unwanted":{"files":0,"bytes":0},"renamed":0,"renames":[],"newlyWanted":0,"downloading":0,` + noLanguages + `,"made":{"files":0,"works":0,"seconds":0,"bytes":0,"nodes":[],"unconvertible":[],"matching":0,"rows":[]},"remake":{"older":0,"rows":[]}}`
	savedFixture  = `{"id":"id1","name":"Fixture","contentType":"fixture","default":true,"targets":[{"targetId":"id2","sourcing":"download"}],"editions":{"kind":"any"},"languages":[`
	frenchOnWire  = `{"language":"fr","forced":true,"hearingImpaired":"avoid","audio":"no_audio"}`
	invalidLocale = `{"language":"en-GB","forced":false,"hearingImpaired":"allow","audio":"always"}`
)

func TestAProfileSavesItsLanguagesAndTouchesNoWork(t *testing.T) {
	s := serve(t)
	path := "/api/v1/profiles/" + s.fixture + "?confirm=0&confirmSubtitles=0"
	s.Expect(t, http.MethodPut, path, withLanguages("Fixture", "["+english+","+forcedFrench+"]", s.starter), http.StatusOK,
		`{"impact":`+untouched+`,"profile":`+savedFixture+english+`,`+frenchOnWire+`],"works":0}}`)
	code, body := s.Call(t, http.MethodGet, "/api/v1/profiles", "")
	if want := `"languages":[` + english + `,` + frenchOnWire + `]`; code != http.StatusOK || !strings.Contains(body, want) {
		t.Fatalf("GET /profiles = %d %s, want the Fixture Profile with %s", code, body, want)
	}
}

func TestAProfileRefusesALanguageTwiceAnUnknownOneAndATypeWithoutSubtitles(t *testing.T) {
	s := serve(t)
	path := "/api/v1/profiles/" + s.fixture
	twice := strings.Replace(english, `"allow"`, `"prefer"`, 1)
	s.Expect(t, http.MethodPut, path, withLanguages("Fixture", "["+forcedFrench+","+english+","+twice+"]"), http.StatusUnprocessableEntity,
		`{"status":422,"code":"duplicate_language","title":"The Profile lists a subtitle language twice","detail":"Full English subtitles are already on this list.",`+
			`"errors":[{"location":"body.languages[2]","message":"Full English subtitles are already on this list."}]}`)
	s.Expect(t, http.MethodPut, path, withLanguages("Fixture", "["+invalidLocale+"]"), http.StatusUnprocessableEntity,
		`{"status":422,"code":"invalid","title":"Some fields have values SlipStream can't accept","detail":"en-GB isn't a supported language code. Use an ISO 639 code, such as en, or pt-BR for Brazilian Portuguese.",`+
			`"errors":[{"location":"body.languages[0].language","message":"en-GB isn't a supported language code. Use an ISO 639 code, such as en, or pt-BR for Brazilian Portuguese."}]}`)

	*s.subtitled = false
	refused := `{"status":422,"code":"no_subtitles_for_type","title":"The Content Type has no subtitles","detail":"Fixture has no subtitles, so its Profiles can't list subtitle languages.",` +
		`"errors":[{"location":"body.languages","message":"Fixture has no subtitles, so its Profiles can't list subtitle languages."}]}`
	s.Expect(t, http.MethodPut, path, withLanguages("Fixture", "["+english+"]"), http.StatusUnprocessableEntity, refused)
	s.Expect(t, http.MethodPost, "/api/v1/profiles", withLanguages("Other", "["+english+"]"), http.StatusUnprocessableEntity, refused)
	s.Expect(t, http.MethodPut, path, withLanguages("Fixture", "[]", s.starter), http.StatusOK, `{"impact":`+untouched+`,"profile":`+savedFixture+`],"works":0}}`)
}
