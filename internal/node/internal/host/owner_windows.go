package host

// ForeignOwner is nil on Windows: the per-user folders a Windows install
// uses are private to their user already.
func ForeignOwner(string) (*Owner, error) { return nil, nil }
