package serializer

import (
	"encoding/json"
	"testing"
)

func TestNewtonsoftTypesReadWhatJsonReads(t *testing.T) {
	var got struct {
		Text, Number, Yes, Kept String
		Count, FromText, Empty  Int
		On, OnText, OnNumber    Bool
		Ptr                     *Int
	}
	got.Kept, got.Empty = "before", 7
	err := json.Unmarshal([]byte(`{"text":"Harbor","number":12.5,"yes":true,"kept":null,
		"count":42,"fromText":"43","empty":"",
		"on":true,"onText":"FALSE","onNumber":2,"ptr":null}`), &got)
	if err != nil {
		t.Fatal(err)
	}
	if got.Text != "Harbor" || got.Number != "12.5" || got.Yes != "True" || got.Kept != "before" {
		t.Errorf("strings = %q %q %q %q", got.Text, got.Number, got.Yes, got.Kept)
	}
	if got.Count != 42 || got.FromText != 43 || got.Empty != 7 {
		t.Errorf("ints = %d %d %d", got.Count, got.FromText, got.Empty)
	}
	if !got.On || got.OnText || !got.OnNumber || got.Ptr != nil {
		t.Errorf("bools, pointer = %v %v %v %v", got.On, got.OnText, got.OnNumber, got.Ptr)
	}
	for _, in := range []string{`{"count":"4x"}`, `{"on":"yes"}`, `{"on":"1"}`} {
		var v struct {
			Count Int
			On    Bool
		}
		if err := json.Unmarshal([]byte(in), &v); err == nil {
			t.Errorf("read %s as %+v; want an error", in, v)
		}
	}
}

func TestSTJsonTypesReadWhatSTJsonReads(t *testing.T) {
	var got struct {
		Count, FromText STJInt
		Ratio, RatioTxt STJFloat
		On, One, Zero   STJBool
		Year            *STJInt
	}
	err := json.Unmarshal([]byte(`{"count":42,"fromText":"43","ratio":0.5,"ratioTxt":"1.5",
		"on":true,"one":1,"zero":0,"year":null}`), &got)
	if err != nil {
		t.Fatal(err)
	}
	if got.Count != 42 || got.FromText != 43 || got.Ratio != 0.5 || got.RatioTxt != 1.5 {
		t.Errorf("numbers = %d %d %v %v", got.Count, got.FromText, got.Ratio, got.RatioTxt)
	}
	if !got.On || !got.One || got.Zero || got.Year != nil {
		t.Errorf("bools, pointer = %v %v %v %v", got.On, got.One, got.Zero, got.Year)
	}
	for _, in := range []string{`{"count":null}`, `{"count":""}`, `{"count":"0x10"}`, `{"count":1.5}`,
		`{"ratio":"NaN"}`, `{"on":null}`, `{"on":"true"}`, `{"on":2}`} {
		var v struct {
			Count STJInt
			Ratio STJFloat
			On    STJBool
		}
		if err := json.Unmarshal([]byte(in), &v); err == nil {
			t.Errorf("read %s as %+v; want an error", in, v)
		}
	}
}
