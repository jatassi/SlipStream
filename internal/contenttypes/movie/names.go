package movie

import "github.com/jatassi/slipstream/internal/release"

// ReadName reads no title yet, so every core fact reads the whole name.
func (*Movie) ReadName(release.Prepared) release.Name { return release.Name{} }

// CleanTitle is the title as given.
func (*Movie) CleanTitle(title string) string { return title }
