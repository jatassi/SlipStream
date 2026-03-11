package movie

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/library/movies"
	"github.com/slipstream/slipstream/internal/library/rootfolder"
	"github.com/slipstream/slipstream/internal/module"
)

var _ module.ImportHandler = (*importHandler)(nil)

type importHandler struct {
	movieSvc      *movies.Service
	rootFolderSvc *rootfolder.Service
	logger        zerolog.Logger
}

func newImportHandler(movieSvc *movies.Service, rootFolderSvc *rootfolder.Service, logger *zerolog.Logger) *importHandler {
	return &importHandler{
		movieSvc:      movieSvc,
		rootFolderSvc: rootFolderSvc,
		logger:        logger.With().Str("component", "movie-import").Logger(),
	}
}

func (h *importHandler) MatchDownload(ctx context.Context, download *module.CompletedDownload) ([]module.MatchedEntity, error) {
	movie, err := h.movieSvc.Get(ctx, download.EntityID)
	if err != nil {
		return nil, fmt.Errorf("movie %d not found: %w", download.EntityID, err)
	}

	rf, err := h.rootFolderSvc.Get(ctx, movie.RootFolderID)
	if err != nil {
		return nil, fmt.Errorf("root folder %d not found: %w", movie.RootFolderID, err)
	}

	entity := module.MatchedEntity{
		ModuleType:       module.TypeMovie,
		EntityType:       module.EntityMovie,
		EntityID:         movie.ID,
		Title:            movie.Title,
		RootFolder:       rf.Path,
		Confidence:       1.0,
		Source:           "queue",
		QualityProfileID: movie.QualityProfileID,
		TokenData: map[string]any{
			"MovieTitle": movie.Title,
			"MovieYear":  movie.Year,
			"ImdbID":     movie.ImdbID,
			"TmdbID":     movie.TmdbID,
		},
	}

	if download.TargetSlotID != nil {
		entity.TokenData["TargetSlotID"] = *download.TargetSlotID
	}

	return []module.MatchedEntity{entity}, nil
}

func (h *importHandler) ImportFile(ctx context.Context, filePath string, entity *module.MatchedEntity, qi *module.QualityInfo) (*module.ImportResult, error) {
	qid := int64(qi.QualityID)
	originalPath, _ := entity.TokenData["OriginalPath"].(string)
	movieFile, err := h.movieSvc.AddFile(ctx, entity.EntityID, &movies.CreateMovieFileInput{
		Path:         filePath,
		QualityID:    &qid,
		Quality:      qi.Quality,
		OriginalPath: originalPath,
	})
	if err != nil {
		return nil, fmt.Errorf("create movie file record: %w", err)
	}

	return &module.ImportResult{
		FileID:          movieFile.ID,
		DestinationPath: filePath,
		QualityID:       qi.QualityID,
	}, nil
}

func (h *importHandler) SupportsMultiFileDownload() bool { return false }

func (h *importHandler) MatchIndividualFile(_ context.Context, _ string, _ *module.MatchedEntity) (*module.MatchedEntity, error) {
	return nil, fmt.Errorf("movie module does not support multi-file downloads")
}

func (h *importHandler) IsGroupImportReady(_ context.Context, _ *module.MatchedEntity, _ []module.MatchedEntity) bool {
	return false
}

func (h *importHandler) MediaInfoFields() []module.MediaInfoFieldDecl {
	return []module.MediaInfoFieldDecl{
		{Name: "video_codec", Required: false},
		{Name: "audio_codec", Required: false},
		{Name: "audio_channels", Required: false},
		{Name: "resolution", Required: false},
		{Name: "dynamic_range", Required: false},
	}
}
