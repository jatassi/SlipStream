package release

// Prepared is a name made ready for reading, as Sonarr's ParseTitle
// prepares it before any pattern runs. Prepare is deterministic.
type Prepared struct {
	// Original is the name as given. REAL is counted in it, case-sensitively.
	Original string
	// Title is Original reversed when written backwards, without its
	// extension, site prefix and suffix, torrent suffix and trailing
	// quality bracket, with 【】 read as [].
	Title string
	// Simple is Title with resolution, codec, audio, bit depth and frame
	// sizes blanked, so no number in them reads as an episode.
	Simple string
	// Obfuscated is a hashed or obfuscated name. Its Name is zero.
	Obfuscated bool
}

func Prepare(name string) Prepared {
	return Prepared{Original: name, Title: name, Simple: name}
}
