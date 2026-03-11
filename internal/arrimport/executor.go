package arrimport

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/module"
	"github.com/slipstream/slipstream/internal/progress"
)

// Executor handles the actual import of media from a source into SlipStream.
type Executor struct {
	db              *sql.DB
	reader          Reader
	sourceType      SourceType
	registry        ModuleRegistry
	progressManager *progress.Manager
	logger          *zerolog.Logger
}

// NewExecutor creates a new import executor.
func NewExecutor(
	db *sql.DB,
	reader Reader,
	sourceType SourceType,
	registry ModuleRegistry,
	progressManager *progress.Manager,
	logger *zerolog.Logger,
) *Executor {
	return &Executor{
		db:              db,
		reader:          reader,
		sourceType:      sourceType,
		registry:        registry,
		progressManager: progressManager,
		logger:          logger,
	}
}

// Run executes the full import process.
func (e *Executor) Run(ctx context.Context, mappings ImportMappings) {
	report := &ImportReport{Errors: []string{}}
	_ = e.progressManager.StartActivity("arrimport", progress.ActivityTypeImport, "Library Import")

	defer func() {
		e.progressManager.UpdateActivityMetadata("arrimport", "report", report)
		if len(report.Errors) > 0 {
			e.progressManager.FailActivity("arrimport", fmt.Sprintf("Completed with %d errors", len(report.Errors)))
		} else {
			e.progressManager.CompleteActivity("arrimport", fmt.Sprintf("Import complete: %d movies, %d series imported", report.MoviesCreated, report.SeriesCreated))
		}
	}()

	if e.sourceType == SourceTypeRadarr {
		e.importMovies(ctx, mappings, report)
	}

	if e.sourceType == SourceTypeSonarr {
		e.importAllSeries(ctx, mappings, report)
	}

	e.logger.Info().
		Int("moviesCreated", report.MoviesCreated).
		Int("moviesSkipped", report.MoviesSkipped).
		Int("seriesCreated", report.SeriesCreated).
		Int("seriesSkipped", report.SeriesSkipped).
		Int("filesImported", report.FilesImported).
		Int("errors", len(report.Errors)).
		Msg("import finished")
}

func (e *Executor) importMovies(ctx context.Context, mappings ImportMappings, report *ImportReport) {
	adapter := e.registry.GetMovieArrAdapter()
	if adapter == nil {
		report.Errors = append(report.Errors, "no movie arr import adapter registered")
		return
	}

	adapted := &readerAdapter{inner: e.reader}
	sourceMovies, err := adapted.ReadMovies(ctx)
	if err != nil {
		report.Errors = append(report.Errors, fmt.Sprintf("failed to read movies: %v", err))
		return
	}

	moduleMappings := convertToModuleMappings(mappings, e.sourceType)
	total := len(sourceMovies)

	for i := range sourceMovies {
		pct := 0
		if total > 0 {
			pct = (i * 100) / total
		}
		e.progressManager.UpdateActivity("arrimport", fmt.Sprintf("Importing: %s", sourceMovies[i].Title), pct)

		entity, err := adapter.ImportMovie(ctx, sourceMovies[i], moduleMappings)
		if err != nil {
			report.MoviesErrored++
			report.Errors = append(report.Errors, fmt.Sprintf("failed to import movie %q: %v", sourceMovies[i].Title, err))
			continue
		}
		if entity == nil {
			report.MoviesSkipped++
			continue
		}

		report.MoviesCreated++
		report.FilesImported += entity.FilesImported
		report.TotalFiles += entity.FilesImported
		report.Errors = append(report.Errors, entity.Errors...)
	}
}

func (e *Executor) importAllSeries(ctx context.Context, mappings ImportMappings, report *ImportReport) {
	adapter := e.registry.GetTVArrAdapter()
	if adapter == nil {
		report.Errors = append(report.Errors, "no TV arr import adapter registered")
		return
	}

	adapted := &readerAdapter{inner: e.reader}
	seriesList, err := adapted.ReadSeries(ctx)
	if err != nil {
		report.Errors = append(report.Errors, fmt.Sprintf("failed to read series: %v", err))
		return
	}

	moduleMappings := convertToModuleMappings(mappings, e.sourceType)
	total := len(seriesList)

	for i := range seriesList {
		pct := 0
		if total > 0 {
			pct = (i * 100) / total
		}
		e.progressManager.UpdateActivity("arrimport", fmt.Sprintf("Importing: %s", seriesList[i].Title), pct)

		entity, err := adapter.ImportSeries(ctx, seriesList[i], adapted, moduleMappings)
		if err != nil {
			report.SeriesErrored++
			report.Errors = append(report.Errors, fmt.Sprintf("failed to import series %q: %v", seriesList[i].Title, err))
			continue
		}
		if entity == nil {
			report.SeriesSkipped++
			continue
		}

		report.SeriesCreated++
		report.FilesImported += entity.FilesImported
		report.TotalFiles += entity.FilesImported
		report.Errors = append(report.Errors, entity.Errors...)
	}
}

func convertToModuleMappings(mappings ImportMappings, sourceType SourceType) module.ArrImportMappings {
	result := module.ArrImportMappings{
		RootFolderMapping:     mappings.RootFolderMapping,
		QualityProfileMapping: mappings.QualityProfileMapping,
	}
	if sourceType == SourceTypeRadarr {
		result.SelectedIDs = mappings.SelectedMovieTmdbIDs
	} else {
		result.SelectedIDs = mappings.SelectedSeriesTvdbIDs
	}
	return result
}
