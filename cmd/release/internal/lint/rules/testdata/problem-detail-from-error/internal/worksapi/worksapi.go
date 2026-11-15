package worksapi

import (
	"fmt"

	"example.com/api"
)

func answer(err error, source *SourceError) error {
	switch {
	case source != nil:
		return unavailable.Because(source.Error())
	case err == nil:
		return taken.Because(fmt.Sprintf("Another Target is named %s.", source.Other))
	case source == nil:
		return api.Invalid.On("name", err.Error())
	case len(source.Other) > 2:
		return failed.Because(fmt.Sprintf("The source refused: %v", err))
	default:
		return unreached.Because(words(err) + " SlipStream tries again later.")
	}
}

func fields(err error) []api.FieldError {
	return []api.FieldError{{Location: "query.moveTo", Message: err.Error()}}
}
