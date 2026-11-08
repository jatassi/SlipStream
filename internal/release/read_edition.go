package release

// readEdition is the edition text tail names: unknown until Radarr's
// edition pattern is ported.
func readEdition(string) Fact[string] { return Fact[string]{} }
