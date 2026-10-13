package nodesapi_test

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/nodesapi"
	"github.com/jatassi/slipstream/internal/role"
)

func TestTheRoleEnumIsRoleNames(t *testing.T) {
	r := api.NewRouter(api.Services{})
	nodesapi.Routes(r, nil)
	var doc struct {
		Components struct {
			Schemas map[string]struct {
				Properties map[string]struct {
					Items struct {
						Enum []string `json:"enum"`
					} `json:"items"`
				} `json:"properties"`
			} `json:"schemas"`
		} `json:"components"`
	}
	if err := json.Unmarshal(r.Document(), &doc); err != nil {
		t.Fatal(err)
	}
	var want []string
	for _, n := range role.Names() {
		want = append(want, string(n))
	}
	for _, schema := range []string{"NodesNode", "NodesRoles"} {
		if got := doc.Components.Schemas[schema].Properties["roles"].Items.Enum; !slices.Equal(got, want) {
			t.Errorf("%s.roles enum = %v, want role.Names() %v", schema, got, want)
		}
	}
}
