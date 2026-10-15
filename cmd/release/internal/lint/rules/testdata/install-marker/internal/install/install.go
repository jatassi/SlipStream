package install

type Kind uint8

type Layout struct{}

func Read(Layout) (Kind, error) { return 0, nil }

func Write(string, Kind) error { return nil }

func MarkerPath(Layout) string { return "" }

func again() { _, _ = Read(Layout{}) }
