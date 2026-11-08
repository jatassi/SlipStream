package release

// Describe is the facts in words, joined with " · ": Quality, HDR, audio
// and channels, group, size. "WEBDL-1080p · Dolby Vision · DD+ 5.1 · FLUX · 6.2 GB".
// Only the Quality is described until the stream facts are read.
func Describe(r Read) string {
	if q, ok := r.Facts.Quality.Get(); ok {
		return q.Info().Name
	}
	return ""
}
