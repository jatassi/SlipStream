package works

import "slices"

// Among is the Works or Items a batch read covers, by id. When they are
// every one in the reader's snapshot, a read scans the table; otherwise it
// binds them Batch at a time.
type Among[ID any] struct {
	ids   []ID
	every bool
}

// Every is ids, which are every Work or every Item in the snapshot.
func Every[ID any](ids []ID) Among[ID] { return Among[ID]{ids: ids, every: true} }

func Only[ID any](ids []ID) Among[ID] { return Among[ID]{ids: ids} }

// Like is ids, which are every one of their kind, or of a Content Type's,
// when a is every one of its.
func Like[ID, J any](a Among[ID], ids []J) Among[J] { return Among[J]{ids: ids, every: a.every} }

func (a Among[ID]) IDs() []ID { return a.ids }

func (a Among[ID]) Every() bool { return a.every }

// Read answers all's rows when a is every one, else of's rows for each
// Batch of a's ids in turn.
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
