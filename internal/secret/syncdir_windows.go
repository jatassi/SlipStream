package secret

// syncDir does nothing: Windows cannot flush a directory handle.
func syncDir(string) error { return nil }
