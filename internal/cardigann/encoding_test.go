package cardigann

import (
	"testing"

	"github.com/jatassi/slipstream/internal/prowlarr/encoding"
)

func TestQueryString(t *testing.T) {
	got := queryString(encoding.UTF8, []pair{{"q", "a b"}, {"cat[]", "1"}, {"cat[]", "2"}}, "&")
	if want := "q=a+b&cat[]=1&cat[]=2"; got != want {
		t.Errorf("QueryString = %q, want %q", got, want)
	}
	got = queryString(encoding.UTF8, []pair{{"a", "1"}, {"b", ""}}, ";")
	if want := "a=1;b="; got != want {
		t.Errorf("QueryString = %q, want %q", got, want)
	}
}
