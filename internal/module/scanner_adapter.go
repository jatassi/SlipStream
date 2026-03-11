package module

import (
	"github.com/slipstream/slipstream/internal/library/scanner"
)

// scannerModuleParser adapts a Module to the scanner.ModuleFileParser interface.
type scannerModuleParser struct {
	mod Module
}

func (a *scannerModuleParser) ID() string {
	return string(a.mod.ID())
}

func (a *scannerModuleParser) TryMatch(filename string) (confidence float64, result *scanner.ParseResultData) {
	conf, parseResult := a.mod.TryMatch(filename)
	if parseResult == nil {
		return 0, nil
	}

	isTV := a.mod.ID() == TypeTV

	return conf, &scanner.ParseResultData{
		Title:         parseResult.Title,
		Year:          parseResult.Year,
		Quality:       parseResult.Quality,
		Source:        parseResult.Source,
		Codec:         parseResult.Codec,
		HDRFormats:    parseResult.HDRFormats,
		AudioCodecs:   parseResult.AudioCodecs,
		AudioChannels: parseResult.AudioChannels,
		ReleaseGroup:  parseResult.ReleaseGroup,
		Revision:      parseResult.Revision,
		Edition:       parseResult.Edition,
		Languages:     parseResult.Languages,
		IsTV:          isTV,
		Extra:         parseResult.Extra,
	}
}

// scannerRegistryAdapter adapts a module.Registry to scanner.ModuleParserRegistry.
type scannerRegistryAdapter struct {
	registry *Registry
}

func (a *scannerRegistryAdapter) AllFileParsers() []scanner.ModuleFileParser {
	modules := a.registry.All()
	parsers := make([]scanner.ModuleFileParser, len(modules))
	for i, mod := range modules {
		parsers[i] = &scannerModuleParser{mod: mod}
	}
	return parsers
}

// NewScannerRegistryAdapter creates a scanner.ModuleParserRegistry that delegates
// to the given module registry. Call scanner.SetGlobalRegistry with the result.
func NewScannerRegistryAdapter(reg *Registry) scanner.ModuleParserRegistry {
	return &scannerRegistryAdapter{registry: reg}
}
