package formatsapi

import (
	"context"
	"strconv"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/formats"
	"github.com/jatassi/slipstream/internal/identity"
)

type kindValue struct {
	Value api.Bare `json:"value" doc:"As the value field holds it: a number in the radarr and sonarr dialects, a name in slipstream's"`
	Words string   `json:"words"`
}

type offeredKind struct {
	Kind    string      `json:"kind"`
	Words   string      `json:"words"`
	Fields  []string    `json:"fields" nullable:"false" doc:"The fields its conditions take, in upstream's order"`
	Pattern bool        `json:"pattern" doc:"Its value is a .NET regular expression, matched ignoring case"`
	Values  []kindValue `json:"values" nullable:"false" doc:"What its value field takes, in upstream's order; empty for a kind whose value is a pattern or a number"`
}

type dialectSchema struct {
	Dialect string        `json:"dialect" enum:"slipstream,radarr,sonarr"`
	Kinds   []offeredKind `json:"kinds" nullable:"false" doc:"In upstream's order; a kind only the other app has is left out"`
}

type schema struct {
	Dialects []dialectSchema `json:"dialects" nullable:"false" doc:"SlipStream's own, then Radarr's and Sonarr's"`
}

type schemaOutput struct{ Body schema }

func (handlers) customFormatSchema(context.Context, identity.Admin, *struct{}) (*schemaOutput, error) {
	out := schema{Dialects: []dialectSchema{}}
	for _, d := range []formats.Dialect{formats.SlipStream, formats.Radarr, formats.Sonarr} {
		ds := dialectSchema{Dialect: string(d), Kinds: []offeredKind{}}
		for _, k := range formats.KindsIn(d) {
			ok := offeredKind{Kind: string(k.Kind), Words: k.Words, Fields: k.Fields, Pattern: k.Regex, Values: make([]kindValue, len(k.Values))}
			for i, v := range k.Values {
				ok.Values[i] = kindValue{Value: valueOf(d, v.Key), Words: v.Words}
			}
			ds.Kinds = append(ds.Kinds, ok)
		}
		out.Dialects = append(out.Dialects, ds)
	}
	return &schemaOutput{Body: out}, nil
}

func valueOf(d formats.Dialect, key string) api.Bare {
	if d == formats.SlipStream {
		return api.BareString(key)
	}
	n, err := strconv.Atoi(key)
	if err != nil {
		panic("formatsapi: " + string(d) + " value " + key + " is no number")
	}
	return api.BareNumber(float64(n))
}
