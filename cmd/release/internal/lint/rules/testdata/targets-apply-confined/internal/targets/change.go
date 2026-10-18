package targets

type Change interface{ apply() error }

func Apply(c Change) error { return c.apply() }

func again(c Change) error { return Apply(c) }
