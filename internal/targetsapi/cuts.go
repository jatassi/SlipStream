package targetsapi

import (
	"context"

	"github.com/jatassi/slipstream/internal/identity"
)

type cut struct {
	Key         string   `json:"key"`
	Name        string   `json:"name"`
	Aliases     []string `json:"aliases" nullable:"false" doc:"The phrases that name it in a title or file name; Standard is read where none is named"`
	Origin      string   `json:"origin" enum:"builtin,type" doc:"Built into SlipStream, or added by a Content Type"`
	ContentType *string  `json:"contentType,omitempty" pattern:"^[a-z][a-z0-9]{1,15}$" doc:"The Content Type that adds it, for origin type"`
}

type cutList struct {
	Cuts []cut `json:"cuts" nullable:"false" doc:"The built-in cuts, Standard first, then each Content Type's by type"`
}

type cutListOutput struct{ Body cutList }

func (h handlers) listCuts(_ context.Context, by identity.Admin, _ *struct{}) (*cutListOutput, error) {
	out := cutList{Cuts: []cut{}}
	for _, c := range open(h.goals).ListCuts(by) {
		row := cut{Key: string(c.Key), Name: c.Name, Aliases: append([]string{}, c.Aliases...), Origin: "builtin"}
		if c.Type != "" {
			row.Origin, row.ContentType = "type", new(string(c.Type))
		}
		out.Cuts = append(out.Cuts, row)
	}
	return &cutListOutput{Body: out}, nil
}
