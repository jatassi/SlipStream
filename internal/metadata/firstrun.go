package metadata

import "github.com/jatassi/slipstream/internal/firstrun"

// LanguageStep is first setup's language and region step, done only by
// Continue.
func (m *Metadata) LanguageStep() firstrun.Step { return firstrun.Step{ID: firstrun.LanguageID} }
