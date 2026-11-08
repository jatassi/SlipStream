package stringutil

import (
	"testing"

	"github.com/jatassi/slipstream/internal/prowlarr/encoding"
)

func TestGetQueryStringJoinsARepeatedNamesValuesInTheOrderFirstAdded(t *testing.T) {
	var qc NameValueCollection
	qc.Add("72", "")
	qc.Add("q", "+(tt0000001)")
	qc.Add("o", "seeders")
	qc.Add("Q", "+(The Quiet Orchard)")
	if got, want := qc.GetQueryString(), "72=&q=%2B(tt0000001)%2C%2B(The+Quiet+Orchard)&o=seeders"; got != want {
		t.Errorf("GetQueryString() = %q, want %q", got, want)
	}
	if qc.Count() != 3 || qc.Get("q") != "+(tt0000001),+(The Quiet Orchard)" {
		t.Errorf("Count, Get(q) = %d, %q; want 3 and both values", qc.Count(), qc.Get("q"))
	}
}

func TestSetReplacesANamesValuesWhereItStands(t *testing.T) {
	var qc NameValueCollection
	qc.Add("page", "1")
	qc.Add("search", "Juniper Falls")
	qc.Set("PAGE", "2")
	qc.Set("sort", "size")
	if got, want := qc.GetQueryString(), "page=2&search=Juniper+Falls&sort=size"; got != want {
		t.Errorf("GetQueryString() = %q, want %q", got, want)
	}
}

func TestGetQueryStringWithItsOptionalParameters(t *testing.T) {
	var qc NameValueCollection
	qc.Add("categories[]", "1")
	qc.Add("categories[]", "2")
	qc.Add("search", "Mørk Skog")
	latin1, err := encoding.GetEncoding("iso-8859-1")
	if err != nil {
		t.Fatal(err)
	}
	got := qc.GetQueryStringWith(QueryString{Encoding: latin1, DuplicateKeysIfMulti: true, Separator: ";"})
	if want := "categories[]=1;categories[]=2;search=M%F8rk+Skog"; got != want {
		t.Errorf("GetQueryStringWith = %q, want %q", got, want)
	}
}

func TestGetQueryStringOverPairsKeepsEachRepeatedKey(t *testing.T) {
	var parameters KeyValuePairs
	parameters.Add("st", "1")
	parameters.Add("cat[]", "19")
	parameters.Add("cat[]", "6")
	parameters.Add("search", "Mørk Skog+")
	if got, want := parameters.GetQueryString(), "st=1&cat[]=19&cat[]=6&search=M%C3%B8rk+Skog%2B"; got != want {
		t.Errorf("GetQueryString() = %q, want %q", got, want)
	}
}
