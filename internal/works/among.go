package works

import "slices"

// Among is the Works or Items a batch read covers: every one in the
// snapshot, which a plain scan reads, or the listed ones, which it binds
// Batch at a time.
type Among[ID any] struct {
	ids   []ID
	every bool
}

func Every[ID any]() Among[ID] { return Among[ID]{every: true} }

func Only[ID any](ids []ID) Among[ID] { return Among[ID]{ids: ids} }

func (a Among[ID]) Every() bool { return a.every }

// Read answers all's rows for every one, else of's rows for each Batch of
// the listed ids in turn.
func Read[ID, T any](a Among[ID], all func() ([]T, error), of func([]ID) ([]T, error)) ([]T, error) {
	if a.every {
		return all()
	}
	var out []T
	for chunk := range slices.Chunk(a.ids, Batch) {
		rows, err := of(chunk)
		if err != nil {
			return nil, err
		}
		out = append(out, rows...)
	}
	return out, nil
}
