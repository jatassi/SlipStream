package grabsapi

import (
	"reflect"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/targets"
)

func TestTheReasonEnumListsEveryReasonCode(t *testing.T) {
	field, _ := reflect.TypeFor[reason]().FieldByName("Code")
	codes := make([]string, 0, len(targets.ReasonCodes()))
	for _, c := range targets.ReasonCodes() {
		codes = append(codes, string(c))
	}
	if got, want := field.Tag.Get("enum"), strings.Join(codes, ","); got != want {
		t.Fatalf("the API's reason codes are %s, want every ReasonCode in order: %s", got, want)
	}
}
