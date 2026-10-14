package shape_test

import (
	"reflect"
	"slices"
	"testing"

	"github.com/jatassi/slipstream/conformance/internal/shape"
)

var answers = []shape.Body{
	{Key: "search 200", JSON: []byte(`{"page":1,"results":[{"id":550,"title":"Fight Club","poster_path":null},{"id":551,"title":"Other","poster_path":"/a.jpg"}]}`)},
	{Key: "search 200", JSON: []byte(`{"page":1,"results":[]}`)},
	{Key: "movie 404", JSON: []byte(`{"success":false,"status_code":34}`)},
}

func TestOfKeepsKeyNamesAndTypesNotValues(t *testing.T) {
	got, err := shape.Of(answers)
	if err != nil {
		t.Fatal(err)
	}
	want := shape.Shape{
		"search 200": {
			"$":                       "object",
			"$.page":                  "number",
			"$.results":               "array",
			"$.results[]":             "object",
			"$.results[].id":          "number",
			"$.results[].title":       "string",
			"$.results[].poster_path": "null|string",
		},
		"movie 404": {"$": "object", "$.success": "boolean", "$.status_code": "number"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Of = %v\nwant %v", got, want)
	}
}

func TestCheckNamesWhatTheShapeLacks(t *testing.T) {
	s, err := shape.Of(answers)
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.Check([]shape.Body{
		{Key: "search 200", JSON: []byte(`{"page":"1","results":[{"id":9000001,"title":"The Quiet Orchard","adult":false}]}`)},
		{Key: "movie 404", JSON: []byte(`{"success":false}`)},
		{Key: "movie 200", JSON: []byte(`{}`)},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"movie 200: not in shape.json",
		"search 200: $.page is string, shape.json has number",
		"search 200: $.results[].adult (boolean) is not in shape.json",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("Check = %q\nwant %q", got, want)
	}
}

func TestCheckRefusesABodyThatIsNotJSON(t *testing.T) {
	_, err := shape.Shape{}.Check([]shape.Body{{Key: "search 503", JSON: []byte("<html>")}})
	if want := "search 503: invalid character '<' looking for beginning of value"; err == nil || err.Error() != want {
		t.Fatalf("Check = %v, want %s", err, want)
	}
}
