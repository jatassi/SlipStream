package release

// Fact is a value that may be unknown. Its zero value is unknown. An
// unknown fact passes every hard filter, so a title that says nothing
// never blocks a grab.
type Fact[T any] struct {
	value T
	known bool
}

func Known[T any](v T) Fact[T] { return Fact[T]{value: v, known: true} }

func (f Fact[T]) Get() (v T, known bool) { return f.value, f.known }
