package indexerfields

import (
	"errors"
	"maps"
	"testing"
)

func TestAValueFitsOnlyWhatItsFieldTakes(t *testing.T) {
	one, nine := 1, 9
	sort := Field{Name: "sort", Label: "Sort", Type: Select, Options: []Option{{Value: "1", Label: "Newest"}, {Value: "2"}}}
	codecs := Field{Name: "codecs", Label: "Codecs", Type: Multiselect, Options: []Option{{Value: "h264", Label: "H.264"}, {Value: "hevc", Label: "HEVC"}}}
	pages := Field{Name: "pages", Label: "Pages", Type: Number, Min: &one, Max: &nine}
	free := Field{Name: "freeleech", Label: "Freeleech only", Type: Checkbox}
	for _, tc := range []struct {
		f        Field
		v        string
		sentence string
	}{
		{sort, "2", ""},
		{sort, "3", "Choose one of the options for Sort."},
		{codecs, "", ""},
		{codecs, "h264,hevc", ""},
		{codecs, "h264,av1", "Choose from the options for Codecs."},
		{pages, "9", ""},
		{pages, "0", "Pages is a whole number from 1 to 9."},
		{pages, "2.5", "Pages is a whole number from 1 to 9."},
		{Field{Name: "n", Label: "Limit", Type: Number}, "-4", ""},
		{Field{Name: "n", Label: "Limit", Type: Number, Min: &one}, "x", "Limit is a whole number of 1 or more."},
		{free, "true", ""},
		{free, "yes", "Freeleech only is true or false."},
		{Field{Name: "apiKey", Type: Text}, "anything", ""},
	} {
		err := Fits(tc.f, tc.v)
		var ve *ValueError
		switch {
		case tc.sentence == "" && err != nil:
			t.Errorf("Fits(%s, %q) = %v, want it to fit", tc.f.Name, tc.v, err)
		case tc.sentence != "" && (!errors.As(err, &ve) || ve.Sentence != tc.sentence || ve.Field != tc.f.Name):
			t.Errorf("Fits(%s, %q) = %v, want %q", tc.f.Name, tc.v, err, tc.sentence)
		}
	}
}

func TestSplitSealsEveryTextButTheUsernameAndKeepsWhatWasSaved(t *testing.T) {
	fields := []Field{
		{Name: "username", Type: Text}, {Name: "password", Type: Password}, {Name: "apiKey", Type: Text},
		{Name: "pages", Type: Number}, {Name: "codecs", Type: Multiselect, Options: []Option{{Value: "hevc"}}},
		{Name: "about", Type: Info},
	}
	plain, sealed, err := Split(fields,
		map[string]string{"username": "sam", "password": "", "pages": "3", "codecs": "hevc"},
		map[string]string{"pages": "1", "about": "x"}, map[string]string{"password": "old", "apiKey": "k"})
	if err != nil {
		t.Fatal(err)
	}
	if want := map[string]string{"username": "sam", "pages": "3", "codecs": "hevc"}; !maps.Equal(plain, want) {
		t.Errorf("plain = %v, want %v", plain, want)
	}
	if want := map[string]string{"password": "old", "apiKey": "k"}; !maps.Equal(sealed, want) {
		t.Errorf("sealed = %v, want %v", sealed, want)
	}
	var ve *ValueError
	if _, _, err := Split(fields, map[string]string{"about": "x"}, nil, nil); !errors.As(err, &ve) || ve.Field != "about" {
		t.Errorf("Split took a value for an info field: %v", err)
	}
}

func TestARequiredValueCantBeSavedEmptyButASealedOneSentEmptyIsKept(t *testing.T) {
	fields := []Field{
		{Name: "username", Label: "Username", Type: Text, Required: true},
		{Name: "password", Label: "Password", Type: Password, Required: true},
		{Name: "cookie", Label: "Cookie", Type: Text},
	}
	for _, tc := range []struct {
		name                string
		values, plain, seal map[string]string
		refused             string
	}{
		{"a new Indexer without a password", map[string]string{"username": "sam"}, nil, nil, "password"},
		{"a new Indexer with an empty password", map[string]string{"username": "sam", "password": ""}, nil, nil, "password"},
		{"a new Indexer with an empty username", map[string]string{"username": "", "password": "pw"}, nil, nil, "username"},
		{"a new Indexer with both", map[string]string{"username": "sam", "password": "pw", "cookie": ""}, nil, nil, ""},
		{"an edit that leaves the password empty", map[string]string{"username": "sam", "password": ""}, map[string]string{"username": "sam"}, map[string]string{"password": "old"}, ""},
		{"an edit that leaves both out", nil, map[string]string{"username": "sam"}, map[string]string{"password": "old"}, ""},
		{"an edit that clears the username", map[string]string{"username": ""}, map[string]string{"username": "sam"}, map[string]string{"password": "old"}, "username"},
	} {
		_, sealed, err := Split(fields, tc.values, tc.plain, tc.seal)
		var ve *ValueError
		switch {
		case tc.refused == "" && err != nil:
			t.Errorf("%s: Split = %v, want it saved", tc.name, err)
		case tc.refused == "" && sealed["password"] == "":
			t.Errorf("%s: sealed = %v, want a password", tc.name, sealed)
		case tc.refused != "" && (!errors.As(err, &ve) || ve.Field != tc.refused):
			t.Errorf("%s: Split = %v, want %s refused", tc.name, err, tc.refused)
		}
	}
}
