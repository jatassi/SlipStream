//go:build unix

package worldtest

func heldBriefly(error) bool { return false }

func withHolders(err error) error { return err }
