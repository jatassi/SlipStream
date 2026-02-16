package slots

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"time"

	"github.com/slipstream/slipstream/internal/database/sqlc"
	"github.com/slipstream/slipstream/internal/library/quality"
	"github.com/slipstream/slipstream/internal/library/scanner"
)

// Migration Preview Types (Req 14.1.2-14.1.5)

// MigrationPreview contains the complete preview of what would happen during migration.
// Req 14.1.2: Preview organized by type (Movies, TV Shows), then per-item
type MigrationPreview struct {
	Movies  []MovieMigrationPreview  `json:"movies"`
	TVShows []TVShowMigrationPreview `json:"tvShows"`
	Summary MigrationSummary         `json:"summary"`
}

// MovieMigrationPreview shows the proposed migration for a single movie.
type MovieMigrationPreview struct {
	MovieID     int64                  `json:"movieId"`
	Title       string                 `json:"title"`
	Year        int                    `json:"year,omitempty"`
	Files       []FileMigrationPreview `json:"files"`
	HasConflict bool                   `json:"hasConflict"`
	Conflicts   []string               `json:"conflicts,omitempty"`
}

// TVShowMigrationPreview shows the proposed migration for a TV series.
// Req 14.1.3: TV shows show per-series, per-season breakdown with collapsible headers
type TVShowMigrationPreview struct {
	SeriesID    int64                    `json:"seriesId"`
	Title       string                   `json:"title"`
	Seasons     []SeasonMigrationPreview `json:"seasons"`
	TotalFiles  int                      `json:"totalFiles"`
	HasConflict bool                     `json:"hasConflict"`
}

// SeasonMigrationPreview shows the proposed migration for a single season.
type SeasonMigrationPreview struct {
	SeasonNumber int                       `json:"seasonNumber"`
	Episodes     []EpisodeMigrationPreview `json:"episodes"`
	TotalFiles   int                       `json:"totalFiles"`
	HasConflict  bool                      `json:"hasConflict"`
}

// EpisodeMigrationPreview shows the proposed migration for a single episode.
type EpisodeMigrationPreview struct {
	EpisodeID     int64                  `json:"episodeId"`
	EpisodeNumber int                    `json:"episodeNumber"`
	Title         string                 `json:"title,omitempty"`
	Files         []FileMigrationPreview `json:"files"`
	HasConflict   bool                   `json:"hasConflict"`
}

// FileMigrationPreview shows the proposed assignment for a single file.
// Req 14.1.4: Show proposed slot assignment for each file
// Req 14.1.5: Show conflicts and files that can't be matched
type FileMigrationPreview struct {
	FileID           int64           `json:"fileId"`
	Path             string          `json:"path"`
	Quality          string          `json:"quality"`
	Size             int64           `json:"size"`
	ProposedSlotID   *int64          `json:"proposedSlotId"`
	ProposedSlotName string          `json:"proposedSlotName,omitempty"`
	MatchScore       float64         `json:"matchScore"`
	NeedsReview      bool            `json:"needsReview"`
	Conflict         string          `json:"conflict,omitempty"`
	SlotRejections   []SlotRejection `json:"slotRejections,omitempty"`
}

// MigrationSummary provides statistics about the migration.
type MigrationSummary struct {
	TotalMovies        int `json:"totalMovies"`
	TotalTVShows       int `json:"totalTvShows"`
	TotalFiles         int `json:"totalFiles"`
	FilesWithSlots     int `json:"filesWithSlots"`
	FilesNeedingReview int `json:"filesNeedingReview"`
	Conflicts          int `json:"conflicts"`
}

// Migration Execution Types

// MigrationResult contains the results of executing a migration.
type MigrationResult struct {
	Success       bool      `json:"success"`
	FilesAssigned int       `json:"filesAssigned"`
	FilesQueued   int       `json:"filesQueued"`
	Errors        []string  `json:"errors,omitempty"`
	CompletedAt   time.Time `json:"completedAt"`
}

// SlotRejection explains why a file didn't match a specific slot
type SlotRejection struct {
	SlotID   int64    `json:"slotId"`
	SlotName string   `json:"slotName"`
	Reasons  []string `json:"reasons"`
}

// FileOverride represents a manual override for a specific file during migration.
// Users can override the automatic slot assignment by:
// - Ignoring the file (exclude from migration)
// - Assigning to a specific slot (override automatic matching)
// - Unassigning (mark as needing review / no slot)
type FileOverride struct {
	FileID int64  `json:"fileId"`
	Type   string `json:"type"`             // "ignore", "assign", or "unassign"
	SlotID *int64 `json:"slotId,omitempty"` // Required when Type is "assign"
}

// ExecuteMigrationInput contains the optional overrides for migration execution.
type ExecuteMigrationInput struct {
	Overrides []FileOverride `json:"overrides,omitempty"`
}

// FileEvaluation contains the result of evaluating a file against all slots.
// This is the shared evaluation result used by both preview and execution.
type FileEvaluation struct {
	FileID       int64
	MediaID      int64
	MediaType    string // "movie" or "episode"
	Path         string
	Quality      string
	Size         int64
	BestSlotID   *int64
	BestSlotName string
	MatchScore   float64
	CanMatch     bool            // True if at least one slot matches
	Reason       string          // Why it can't match (if applicable)
	Rejections   []SlotRejection // Per-slot rejection reasons
}

// evaluateFileAgainstSlots evaluates a single file against all slots to find the best match.
// This is the core matching logic shared by preview and execution.

// slotMatchResult holds the result of matching a file against a single slot.
type slotMatchResult struct {
	score     float64
	rejection *SlotRejection
}

// matchFileToSlot evaluates a parsed file against a single slot's quality profile.
func (s *Service) matchFileToSlot(ctx context.Context, parsed *scanner.ParsedMedia, slot *Slot) slotMatchResult {
	if slot.QualityProfileID == nil {
		return slotMatchResult{rejection: &SlotRejection{
			SlotID: slot.ID, SlotName: slot.Name,
			Reasons: []string{"No quality profile assigned"},
		}}
	}

	profile, err := s.qualityService.Get(ctx, *slot.QualityProfileID)
	if err != nil {
		return slotMatchResult{rejection: &SlotRejection{
			SlotID: slot.ID, SlotName: slot.Name,
			Reasons: []string{"Failed to load quality profile"},
		}}
	}

	releaseAttrs := parsed.ToReleaseAttributes()
	matchResult := quality.MatchProfileAttributes(&releaseAttrs, profile)
	qualityMatchResult := quality.MatchQuality(parsed.Quality, parsed.Source, profile)

	if !matchResult.AllMatch || !qualityMatchResult.Matches {
		var reasons []string
		if !qualityMatchResult.Matches && qualityMatchResult.Reason != "" {
			reasons = append(reasons, "Quality: "+qualityMatchResult.Reason)
		}
		reasons = append(reasons, matchResult.RejectionReasons()...)
		return slotMatchResult{rejection: &SlotRejection{
			SlotID: slot.ID, SlotName: slot.Name, Reasons: reasons,
		}}
	}

	qualityScore := s.calculateQualityScore(parsed)
	return slotMatchResult{score: qualityScore + matchResult.TotalScore}
}

// evaluateFileAgainstSlots evaluates a single file against all slots to find the best match.
// This is the core matching logic shared by preview and execution.
func (s *Service) evaluateFileAgainstSlots(ctx context.Context, path, qualityStr string, slots []*Slot) FileEvaluation {
	eval := FileEvaluation{
		Path:       path,
		Quality:    qualityStr,
		Rejections: make([]SlotRejection, 0),
	}

	parsed := scanner.ParsePath(path)
	if parsed == nil {
		parsed = &scanner.ParsedMedia{Quality: qualityStr}
	}

	var bestSlot *Slot
	var bestScore float64 = -1

	for _, slot := range slots {
		result := s.matchFileToSlot(ctx, parsed, slot)
		if result.rejection != nil {
			eval.Rejections = append(eval.Rejections, *result.rejection)
			continue
		}
		if result.score > bestScore {
			bestScore = result.score
			bestSlot = slot
		}
	}

	if bestSlot != nil {
		eval.BestSlotID = &bestSlot.ID
		eval.BestSlotName = bestSlot.Name
		eval.MatchScore = bestScore
		eval.CanMatch = true
	}

	return eval
}

// ResolvedAssignment represents the result of resolving a file's slot assignment.
// Contains the file evaluation plus the final assignment decision.
type ResolvedAssignment struct {
	FileEvaluation
	AssignedSlotID   *int64 // Final slot (may differ from BestSlotID if that was taken)
	AssignedSlotName string
	Conflict         string // Non-empty if file couldn't be assigned
}

// resolveOneAssignment resolves the slot assignment for a single file evaluation,
// trying the best slot first, then alternatives if the best is taken.
func (s *Service) resolveOneAssignment(ctx context.Context, eval *FileEvaluation, slots []*Slot, filledSlots map[int64]int64) ResolvedAssignment {
	assignment := ResolvedAssignment{FileEvaluation: *eval}

	if !eval.CanMatch || eval.BestSlotID == nil {
		assignment.Conflict = eval.Reason
		return assignment
	}

	if _, taken := filledSlots[*eval.BestSlotID]; !taken {
		assignment.AssignedSlotID = eval.BestSlotID
		assignment.AssignedSlotName = eval.BestSlotName
		filledSlots[*eval.BestSlotID] = eval.FileID
		return assignment
	}

	// Best slot taken, try alternatives
	for _, slot := range slots {
		if _, used := filledSlots[slot.ID]; used {
			continue
		}
		altEval := s.evaluateFileAgainstSlots(ctx, eval.Path, eval.Quality, []*Slot{slot})
		if altEval.CanMatch {
			assignment.AssignedSlotID = &slot.ID
			assignment.AssignedSlotName = slot.Name
			assignment.MatchScore = altEval.MatchScore
			filledSlots[slot.ID] = eval.FileID
			return assignment
		}
	}

	assignment.Conflict = fmt.Sprintf("%s slot taken by higher-scored file", eval.BestSlotName)
	return assignment
}

// resolveSlotAssignments determines optimal slot assignments for files of a single media item.
// Sorts by score, resolves conflicts, and returns assignment decisions.
// This is the ONLY place where assignment logic lives - used by both preview and execution.
func (s *Service) resolveSlotAssignments(ctx context.Context, evals []FileEvaluation, slots []*Slot) []ResolvedAssignment {
	sort.Slice(evals, func(i, j int) bool {
		return evals[i].MatchScore > evals[j].MatchScore
	})

	assignments := make([]ResolvedAssignment, 0, len(evals))
	filledSlots := make(map[int64]int64)

	for i := range evals {
		assignments = append(assignments, s.resolveOneAssignment(ctx, &evals[i], slots, filledSlots))
	}

	return assignments
}

// toFileMigrationPreview converts a ResolvedAssignment to FileMigrationPreview for dry-run display.
func (a *ResolvedAssignment) toFileMigrationPreview() FileMigrationPreview {
	return FileMigrationPreview{
		FileID:           a.FileID,
		Path:             a.Path,
		Quality:          a.Quality,
		Size:             a.Size,
		ProposedSlotID:   a.AssignedSlotID,
		ProposedSlotName: a.AssignedSlotName,
		MatchScore:       a.MatchScore,
		NeedsReview:      a.AssignedSlotID == nil,
		Conflict:         a.Conflict,
		SlotRejections:   a.Rejections,
	}
}

// executeAssignments performs the actual DB assignments and returns counts.
func (s *Service) executeAssignments(ctx context.Context, assignments []ResolvedAssignment) (assigned, queued int) {
	for i := range assignments {
		a := &assignments[i]
		if a.AssignedSlotID == nil {
			queued++
			continue
		}
		if err := s.AssignFileToSlot(ctx, a.MediaType, a.MediaID, *a.AssignedSlotID, a.FileID); err != nil {
			queued++
		} else {
			assigned++
		}
	}
	return assigned, queued
}

// Profile Change Types (Req 15.1.1-15.1.2)

// ProfileChangeAction represents the action to take when changing a slot's profile.
type ProfileChangeAction string

const (
	// ProfileChangeKeep keeps current file assignments
	ProfileChangeKeep ProfileChangeAction = "keep"
	// ProfileChangeReevaluate re-evaluates files and queues non-matches
	ProfileChangeReevaluate ProfileChangeAction = "reevaluate"
	// ProfileChangeCancel aborts the profile change
	ProfileChangeCancel ProfileChangeAction = "cancel"
)

// ProfileChangeRequest is the request for changing a slot's profile.
type ProfileChangeRequest struct {
	SlotID       int64               `json:"slotId"`
	NewProfileID int64               `json:"newProfileId"`
	Action       ProfileChangeAction `json:"action"`
}

// ProfileChangeResult is the result of checking if a profile change requires action.
type ProfileChangeResult struct {
	RequiresPrompt     bool           `json:"requiresPrompt"`
	AffectedFilesCount int            `json:"affectedFilesCount"`
	AffectedFiles      []SlotFileInfo `json:"affectedFiles,omitempty"`
	IncompatibleCount  int            `json:"incompatibleCount"`
}

// GenerateMigrationPreview generates a preview of what would happen during migration.
// Req 14.1.1: Dry run preview is required before enabling multi-version
// Req 14.1.2-14.1.5: Generate organized preview with slot assignments and conflicts
func (s *Service) GenerateMigrationPreview(ctx context.Context) (*MigrationPreview, error) {
	preview := &MigrationPreview{
		Movies:  make([]MovieMigrationPreview, 0),
		TVShows: make([]TVShowMigrationPreview, 0),
	}

	slots, err := s.ListEnabled(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list enabled slots: %w", err)
	}

	if len(slots) == 0 {
		return nil, fmt.Errorf("no enabled slots configured")
	}

	if err := s.generateMoviePreview(ctx, preview, slots); err != nil {
		s.logger.Warn().Err(err).Msg("Error generating movie preview")
	}

	if err := s.generateTVShowPreview(ctx, preview, slots); err != nil {
		s.logger.Warn().Err(err).Msg("Error generating TV show preview")
	}

	s.calculateMigrationSummary(preview)

	return preview, nil
}

// evaluateAndResolveFiles evaluates files and resolves slot assignments, returning preview entries.
func (s *Service) evaluateAndResolveFiles(ctx context.Context, evals []FileEvaluation, slots []*Slot) ([]FileMigrationPreview, bool) {
	assignments := s.resolveSlotAssignments(ctx, evals, slots)
	files := make([]FileMigrationPreview, 0, len(assignments))
	hasConflict := false
	for i := range assignments {
		files = append(files, assignments[i].toFileMigrationPreview())
		if assignments[i].Conflict != "" {
			hasConflict = true
		}
	}
	return files, hasConflict
}

// generateMoviePreview generates the movie portion of the migration preview.
func (s *Service) generateMoviePreview(ctx context.Context, preview *MigrationPreview, slots []*Slot) error {
	movies, err := s.queries.ListMovies(ctx)
	if err != nil {
		return err
	}

	for _, movie := range movies {
		files, err := s.queries.ListMovieFiles(ctx, movie.ID)
		if err != nil || len(files) == 0 {
			continue
		}

		var evals []FileEvaluation
		for _, file := range files {
			eval := s.evaluateFileAgainstSlots(ctx, file.Path, file.Quality.String, slots)
			eval.FileID = file.ID
			eval.MediaID = movie.ID
			eval.MediaType = mediaTypeMovie
			eval.Size = file.Size
			evals = append(evals, eval)
		}
		preview.Summary.TotalFiles += len(evals)

		filePreviews, hasConflict := s.evaluateAndResolveFiles(ctx, evals, slots)

		moviePreview := MovieMigrationPreview{
			MovieID:     movie.ID,
			Title:       movie.Title,
			Year:        int(movie.Year.Int64),
			Files:       filePreviews,
			HasConflict: hasConflict,
			Conflicts:   make([]string, 0),
		}

		if len(files) > len(slots) {
			moviePreview.HasConflict = true
			moviePreview.Conflicts = append(moviePreview.Conflicts,
				fmt.Sprintf("Movie has %d files but only %d slots enabled", len(files), len(slots)))
		}

		if len(moviePreview.Files) > 0 {
			preview.Movies = append(preview.Movies, moviePreview)
			preview.Summary.TotalMovies++
		}
	}

	return nil
}

// episodeMetadata holds episode identity info for grouping during TV preview.
type episodeMetadata struct {
	seasonNumber  int
	episodeNumber int
	title         string
}

// groupEpisodeFiles collects episode files and metadata for a series.
func (s *Service) groupEpisodeFiles(ctx context.Context, files []*sqlc.EpisodeFile) (
	byEpisode map[int64][]*sqlc.EpisodeFile, info map[int64]episodeMetadata,
) {
	byEpisode = make(map[int64][]*sqlc.EpisodeFile)
	info = make(map[int64]episodeMetadata)

	for _, file := range files {
		episode, err := s.queries.GetEpisode(ctx, file.EpisodeID)
		if err != nil {
			continue
		}
		byEpisode[file.EpisodeID] = append(byEpisode[file.EpisodeID], file)
		info[file.EpisodeID] = episodeMetadata{
			seasonNumber:  int(episode.SeasonNumber),
			episodeNumber: int(episode.EpisodeNumber),
			title:         episode.Title.String,
		}
	}
	return byEpisode, info
}

// buildEpisodePreview builds the migration preview for a single episode.
func (s *Service) buildEpisodePreview(ctx context.Context, episodeID int64, info episodeMetadata, files []*sqlc.EpisodeFile, slots []*Slot) (preview EpisodeMigrationPreview, fileCount int) {
	preview = EpisodeMigrationPreview{
		EpisodeID:     episodeID,
		EpisodeNumber: info.episodeNumber,
		Title:         info.title,
		Files:         make([]FileMigrationPreview, 0),
	}

	var evals []FileEvaluation
	for _, file := range files {
		eval := s.evaluateFileAgainstSlots(ctx, file.Path, file.Quality.String, slots)
		eval.FileID = file.ID
		eval.MediaID = episodeID
		eval.MediaType = mediaTypeEpisode
		eval.Size = file.Size
		evals = append(evals, eval)
	}

	preview.Files, preview.HasConflict = s.evaluateAndResolveFiles(ctx, evals, slots)

	if len(files) > len(slots) {
		preview.HasConflict = true
	}

	return preview, len(evals)
}

// buildSeasonPreview builds the migration preview for a single season.
func (s *Service) buildSeasonPreview(ctx context.Context, seasonNum int, episodeIDs []int64, episodeFiles map[int64][]*sqlc.EpisodeFile, episodeInfo map[int64]episodeMetadata, slots []*Slot) SeasonMigrationPreview {
	seasonPreview := SeasonMigrationPreview{
		SeasonNumber: seasonNum,
		Episodes:     make([]EpisodeMigrationPreview, 0),
	}

	for _, episodeID := range episodeIDs {
		epPreview, fileCount := s.buildEpisodePreview(ctx, episodeID, episodeInfo[episodeID], episodeFiles[episodeID], slots)
		seasonPreview.TotalFiles += fileCount
		if epPreview.HasConflict {
			seasonPreview.HasConflict = true
		}
		seasonPreview.Episodes = append(seasonPreview.Episodes, epPreview)
	}

	sort.Slice(seasonPreview.Episodes, func(i, j int) bool {
		return seasonPreview.Episodes[i].EpisodeNumber < seasonPreview.Episodes[j].EpisodeNumber
	})
	return seasonPreview
}

// generateTVShowPreview generates the TV show portion of the migration preview.
func (s *Service) generateTVShowPreview(ctx context.Context, preview *MigrationPreview, slots []*Slot) error {
	series, err := s.queries.ListSeries(ctx)
	if err != nil {
		return err
	}

	for _, show := range series {
		files, err := s.queries.ListEpisodeFilesBySeries(ctx, show.ID)
		if err != nil || len(files) == 0 {
			continue
		}

		episodeFiles, episodeInfo := s.groupEpisodeFiles(ctx, files)

		seasonEpisodes := make(map[int][]int64)
		for episodeID, info := range episodeInfo {
			seasonEpisodes[info.seasonNumber] = append(seasonEpisodes[info.seasonNumber], episodeID)
		}

		tvPreview := TVShowMigrationPreview{
			SeriesID:   show.ID,
			Title:      show.Title,
			Seasons:    make([]SeasonMigrationPreview, 0),
			TotalFiles: len(files),
		}

		for seasonNum, episodeIDs := range seasonEpisodes {
			sp := s.buildSeasonPreview(ctx, seasonNum, episodeIDs, episodeFiles, episodeInfo, slots)
			preview.Summary.TotalFiles += sp.TotalFiles
			if sp.HasConflict {
				tvPreview.HasConflict = true
			}
			tvPreview.Seasons = append(tvPreview.Seasons, sp)
		}

		sort.Slice(tvPreview.Seasons, func(i, j int) bool {
			return tvPreview.Seasons[i].SeasonNumber < tvPreview.Seasons[j].SeasonNumber
		})

		if tvPreview.TotalFiles > 0 {
			preview.TVShows = append(preview.TVShows, tvPreview)
			preview.Summary.TotalTVShows++
		}
	}

	return nil
}
func tallyFileSummary(file *FileMigrationPreview, summary *MigrationSummary) {
	if file.ProposedSlotID != nil && !file.NeedsReview && file.Conflict == "" {
		summary.FilesWithSlots++
	}
	if file.NeedsReview {
		summary.FilesNeedingReview++
	}
	if file.Conflict != "" {
		summary.Conflicts++
	}
}

// calculateMigrationSummary calculates the summary statistics for a migration preview.
func (s *Service) calculateMigrationSummary(preview *MigrationPreview) {
	for i := range preview.Movies {
		for j := range preview.Movies[i].Files {
			tallyFileSummary(&preview.Movies[i].Files[j], &preview.Summary)
		}
	}
	for i := range preview.TVShows {
		for j := range preview.TVShows[i].Seasons {
			for k := range preview.TVShows[i].Seasons[j].Episodes {
				for l := range preview.TVShows[i].Seasons[j].Episodes[k].Files {
					tallyFileSummary(&preview.TVShows[i].Seasons[j].Episodes[k].Files[l], &preview.Summary)
				}
			}
		}
	}
}

// migrationFileInfo holds common file fields used during override evaluation.
type migrationFileInfo struct {
	id        int64
	mediaID   int64
	mediaType string
	path      string
	quality   string
	size      int64
}

// applyOverride applies a file override and returns the resulting evaluation, or nil if ignored.
func applyOverride(override FileOverride, file migrationFileInfo, slotMap map[int64]*Slot) *FileEvaluation {
	switch override.Type {
	case "ignore":
		return nil
	case "assign":
		if override.SlotID != nil {
			if slot, ok := slotMap[*override.SlotID]; ok {
				return &FileEvaluation{
					FileID: file.id, MediaID: file.mediaID, MediaType: file.mediaType,
					Path: file.path, Quality: file.quality, Size: file.size,
					BestSlotID: override.SlotID, BestSlotName: slot.Name,
					MatchScore: 100, CanMatch: true,
				}
			}
		}
	case "unassign":
		return &FileEvaluation{
			FileID: file.id, MediaID: file.mediaID, MediaType: file.mediaType,
			Path: file.path, Quality: file.quality, Size: file.size,
			CanMatch: false, Reason: "Manually marked for review",
		}
	}
	return nil
}

// migrateMovieFiles evaluates movie files (with overrides), resolves assignments, and executes.
func (s *Service) migrateMovieFiles(ctx context.Context, overrideMap map[int64]FileOverride, slots []*Slot, slotMap map[int64]*Slot) (assigned, queued int, errs []string) {
	movieFiles, err := s.queries.ListMovieFilesWithoutSlot(ctx)
	if err != nil {
		return 0, 0, []string{fmt.Sprintf("Failed to list movie files: %v", err)}
	}

	groups := make(map[int64][]FileEvaluation)
	for _, file := range movieFiles {
		fi := migrationFileInfo{id: file.ID, mediaID: file.MovieID, mediaType: mediaTypeMovie, path: file.Path, quality: file.Quality.String, size: file.Size}

		if override, exists := overrideMap[file.ID]; exists {
			if result := applyOverride(override, fi, slotMap); result != nil {
				groups[file.MovieID] = append(groups[file.MovieID], *result)
			}
			continue
		}

		eval := s.evaluateFileAgainstSlots(ctx, file.Path, file.Quality.String, slots)
		eval.FileID = file.ID
		eval.MediaID = file.MovieID
		eval.MediaType = mediaTypeMovie
		eval.Size = file.Size
		groups[file.MovieID] = append(groups[file.MovieID], eval)
	}

	for _, evals := range groups {
		a, q := s.executeAssignments(ctx, s.resolveSlotAssignments(ctx, evals, slots))
		assigned += a
		queued += q
	}
	return assigned, queued, nil
}

// migrateEpisodeFiles evaluates episode files (with overrides), resolves assignments, and executes.
func (s *Service) migrateEpisodeFiles(ctx context.Context, overrideMap map[int64]FileOverride, slots []*Slot, slotMap map[int64]*Slot) (assigned, queued int, errs []string) {
	episodeFiles, err := s.queries.ListEpisodeFilesWithoutSlot(ctx)
	if err != nil {
		return 0, 0, []string{fmt.Sprintf("Failed to list episode files: %v", err)}
	}

	groups := make(map[int64][]FileEvaluation)
	for _, file := range episodeFiles {
		fi := migrationFileInfo{id: file.ID, mediaID: file.EpisodeID, mediaType: mediaTypeEpisode, path: file.Path, quality: file.Quality.String, size: file.Size}

		if override, exists := overrideMap[file.ID]; exists {
			if result := applyOverride(override, fi, slotMap); result != nil {
				groups[file.EpisodeID] = append(groups[file.EpisodeID], *result)
			}
			continue
		}

		eval := s.evaluateFileAgainstSlots(ctx, file.Path, file.Quality.String, slots)
		eval.FileID = file.ID
		eval.MediaID = file.EpisodeID
		eval.MediaType = mediaTypeEpisode
		eval.Size = file.Size
		groups[file.EpisodeID] = append(groups[file.EpisodeID], eval)
	}

	for _, evals := range groups {
		a, q := s.executeAssignments(ctx, s.resolveSlotAssignments(ctx, evals, slots))
		assigned += a
		queued += q
	}
	return assigned, queued, nil
}

// ExecuteMigration executes the migration, assigning files to slots.
// Req 14.2.1: Intelligently assign existing files to slots based on quality profile matching
// Req 14.2.2: Files that can't be matched to any slot go to review queue
// Req 14.2.3: Quality profile must be assigned to slot before saving configuration
func (s *Service) ExecuteMigration(ctx context.Context, overrides []FileOverride) (*MigrationResult, error) {
	result := &MigrationResult{
		Errors: make([]string, 0),
	}

	overrideMap := make(map[int64]FileOverride)
	for _, o := range overrides {
		overrideMap[o.FileID] = o
	}

	if err := s.ValidateSlotConfiguration(ctx); err != nil {
		return nil, err
	}

	slots, err := s.ListEnabled(ctx)
	if err != nil {
		return nil, err
	}

	slotMap := make(map[int64]*Slot)
	for _, slot := range slots {
		slotMap[slot.ID] = slot
	}

	assigned, queued, errs := s.migrateMovieFiles(ctx, overrideMap, slots, slotMap)
	result.FilesAssigned += assigned
	result.FilesQueued += queued
	result.Errors = append(result.Errors, errs...)

	assigned, queued, errs = s.migrateEpisodeFiles(ctx, overrideMap, slots, slotMap)
	result.FilesAssigned += assigned
	result.FilesQueued += queued
	result.Errors = append(result.Errors, errs...)

	if err := s.SetDryRunCompleted(ctx, true); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to mark dry-run completed: %v", err))
	}

	_, err = s.queries.UpdateMultiVersionSettings(ctx, sqlc.UpdateMultiVersionSettingsParams{
		Enabled:         1,
		DryRunCompleted: 1,
		LastMigrationAt: sql.NullTime{Time: time.Now(), Valid: true},
	})
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to update migration timestamp: %v", err))
	}

	result.Success = len(result.Errors) == 0
	result.CompletedAt = time.Now()

	s.logger.Info().
		Int("assigned", result.FilesAssigned).
		Int("queued", result.FilesQueued).
		Bool("success", result.Success).
		Msg("Migration completed")

	return result, nil
}

// CheckProfileChange checks if changing a slot's profile requires user action.
// Req 15.1.1: When user changes a slot's quality profile after files are assigned, prompt for action
func (s *Service) CheckProfileChange(ctx context.Context, slotID, newProfileID int64) (*ProfileChangeResult, error) {
	result := &ProfileChangeResult{
		AffectedFiles: make([]SlotFileInfo, 0),
	}

	files, err := s.ListFilesInSlot(ctx, slotID)
	if err != nil {
		return nil, err
	}

	result.AffectedFilesCount = len(files)
	result.RequiresPrompt = len(files) > 0

	if len(files) == 0 {
		return result, nil
	}

	result.AffectedFiles = files

	newProfile, err := s.qualityService.Get(ctx, newProfileID)
	if err != nil {
		return nil, fmt.Errorf("failed to get new profile: %w", err)
	}

	for _, file := range files {
		parsed := scanner.ParsePath(file.FilePath)
		if parsed == nil {
			result.IncompatibleCount++
			continue
		}
		releaseAttrs := parsed.ToReleaseAttributes()
		if matchResult := quality.MatchProfileAttributes(&releaseAttrs, newProfile); !matchResult.AllMatch {
			result.IncompatibleCount++
		}
	}

	return result, nil
}

// reevaluateSlotFiles re-evaluates files against a new profile and unassigns non-matching ones.
func (s *Service) reevaluateSlotFiles(ctx context.Context, files []SlotFileInfo, newProfile *quality.Profile) {
	for _, file := range files {
		parsed := scanner.ParsePath(file.FilePath)
		if parsed == nil {
			if err := s.unassignFileByID(ctx, file.MediaType, file.FileID); err != nil {
				s.logger.Warn().Err(err).Int64("fileId", file.FileID).Msg("Failed to unassign file")
			}
			continue
		}

		releaseAttrs := parsed.ToReleaseAttributes()
		if matchResult := quality.MatchProfileAttributes(&releaseAttrs, newProfile); !matchResult.AllMatch {
			if err := s.unassignFileByID(ctx, file.MediaType, file.FileID); err != nil {
				s.logger.Warn().Err(err).Int64("fileId", file.FileID).Msg("Failed to unassign file")
			}
			s.logger.Info().
				Int64("fileId", file.FileID).
				Str("path", file.FilePath).
				Msg("File unassigned due to profile change - moved to review queue")
		}
	}
}

// ChangeSlotProfile changes a slot's profile with the specified action.
// Req 15.1.2: Options: keep current assignments, re-evaluate (may queue non-matches), or cancel
func (s *Service) ChangeSlotProfile(ctx context.Context, req ProfileChangeRequest) (*Slot, error) {
	if req.Action == ProfileChangeCancel {
		slot, _ := s.Get(ctx, req.SlotID)
		return slot, nil
	}

	files, err := s.ListFilesInSlot(ctx, req.SlotID)
	if err != nil {
		return nil, err
	}

	if len(files) > 0 && req.Action == "" {
		return nil, fmt.Errorf("slot has files assigned; action required")
	}

	newProfile, err := s.qualityService.Get(ctx, req.NewProfileID)
	if err != nil {
		return nil, fmt.Errorf("failed to get new profile: %w", err)
	}

	switch req.Action {
	case ProfileChangeKeep:
		s.logger.Info().
			Int64("slotId", req.SlotID).
			Int64("newProfileId", req.NewProfileID).
			Int("filesKept", len(files)).
			Msg("Keeping file assignments despite profile change")
	case ProfileChangeReevaluate:
		s.reevaluateSlotFiles(ctx, files, newProfile)
	}

	return s.SetProfile(ctx, req.SlotID, &req.NewProfileID)
}

// unassignFileByID unassigns a file from its slot by file ID.
func (s *Service) unassignFileByID(ctx context.Context, mediaType string, fileID int64) error {
	switch mediaType {
	case mediaTypeMovie:
		if err := s.queries.ClearMovieSlotFileByFileID(ctx, sql.NullInt64{Int64: fileID, Valid: true}); err != nil {
			return err
		}
		return s.queries.ClearMovieFileSlot(ctx, fileID)
	case mediaTypeEpisode:
		if err := s.queries.ClearEpisodeSlotFileByFileID(ctx, sql.NullInt64{Int64: fileID, Valid: true}); err != nil {
			return err
		}
		return s.queries.ClearEpisodeFileSlot(ctx, fileID)
	}
	return nil
}

// ResolveReviewQueueItem resolves a review queue item by assigning to a slot or deleting.
// Req 14.3.3: Allow assignment to specific slot or deletion
type ReviewItemAction string

const (
	ReviewActionAssign ReviewItemAction = "assign"
	ReviewActionDelete ReviewItemAction = "delete"
	ReviewActionSkip   ReviewItemAction = "skip"
)

type ResolveReviewItemRequest struct {
	MediaType    string           `json:"mediaType"`
	FileID       int64            `json:"fileId"`
	Action       ReviewItemAction `json:"action"`
	TargetSlotID *int64           `json:"targetSlotId,omitempty"`
}

// ResolveReviewQueueItem handles a review queue item.
// Req 14.3.2: Show file details, detected quality, and available slot options
// Req 14.3.3: Allow assignment to specific slot or deletion
func (s *Service) ResolveReviewQueueItem(ctx context.Context, req ResolveReviewItemRequest) error {
	switch req.Action {
	case ReviewActionAssign:
		if req.TargetSlotID == nil {
			return fmt.Errorf("target slot ID required for assign action")
		}
		switch req.MediaType {
		case mediaTypeMovie:
			file, err := s.queries.GetMovieFile(ctx, req.FileID)
			if err != nil {
				return fmt.Errorf("failed to get movie file: %w", err)
			}
			return s.AssignFileToSlot(ctx, mediaTypeMovie, file.MovieID, *req.TargetSlotID, req.FileID)
		case mediaTypeEpisode:
			file, err := s.queries.GetEpisodeFile(ctx, req.FileID)
			if err != nil {
				return fmt.Errorf("failed to get episode file: %w", err)
			}
			return s.AssignFileToSlot(ctx, mediaTypeEpisode, file.EpisodeID, *req.TargetSlotID, req.FileID)
		}

	case ReviewActionDelete:
		if s.fileDeleter == nil {
			return fmt.Errorf("file deleter not configured")
		}
		return s.fileDeleter.DeleteFile(ctx, req.MediaType, req.FileID)

	case ReviewActionSkip:
		return nil
	}

	return fmt.Errorf("invalid action: %s", req.Action)
}

// reviewFileInfo holds file metadata loaded for a review queue item.
type reviewFileInfo struct {
	path      string
	quality   string
	size      int64
	mediaID   int64
	mediaType string
}

// loadReviewFileInfo loads file metadata for a review queue item.
func (s *Service) loadReviewFileInfo(ctx context.Context, details *ReviewQueueItemDetails, mediaType string, fileID int64) (*reviewFileInfo, error) {
	switch mediaType {
	case mediaTypeMovie:
		file, err := s.queries.GetMovieFile(ctx, fileID)
		if err != nil {
			return nil, err
		}
		details.MediaType = mediaTypeMovie
		details.FileID = fileID
		if movie, _ := s.queries.GetMovie(ctx, file.MovieID); movie != nil {
			details.MediaTitle = movie.Title
		}
		return &reviewFileInfo{path: file.Path, quality: file.Quality.String, size: file.Size, mediaID: file.MovieID, mediaType: mediaTypeMovie}, nil

	case mediaTypeEpisode:
		file, err := s.queries.GetEpisodeFile(ctx, fileID)
		if err != nil {
			return nil, err
		}
		details.MediaType = mediaTypeEpisode
		details.FileID = fileID
		if episode, _ := s.queries.GetEpisode(ctx, file.EpisodeID); episode != nil {
			if series, _ := s.queries.GetSeries(ctx, episode.SeriesID); series != nil {
				details.MediaTitle = fmt.Sprintf("%s S%02dE%02d", series.Title, episode.SeasonNumber, episode.EpisodeNumber)
			}
		}
		return &reviewFileInfo{path: file.Path, quality: file.Quality.String, size: file.Size, mediaID: file.EpisodeID, mediaType: mediaTypeEpisode}, nil
	}
	return nil, fmt.Errorf("unknown media type: %s", mediaType)
}

// populateDetectedAttributes fills in detected quality attributes from path parsing.
func populateDetectedAttributes(details *ReviewQueueItemDetails, path string) {
	parsed := scanner.ParsePath(path)
	if parsed == nil {
		return
	}
	details.DetectedQuality = parsed.Quality
	details.DetectedSource = parsed.Source
	if len(parsed.HDRFormats) > 0 {
		details.DetectedHDR = parsed.HDRFormats[0]
	}
	details.DetectedVideoCodec = parsed.Codec
	if len(parsed.AudioCodecs) > 0 {
		details.DetectedAudioCodec = parsed.AudioCodecs[0]
	}
}

// buildSlotOptions builds the available slot options for a review queue item.
func (s *Service) buildSlotOptions(ctx context.Context, path, qualityStr, mediaType string, mediaID int64) []SlotOption {
	slots, err := s.ListEnabled(ctx)
	if err != nil {
		return nil
	}

	parsed := scanner.ParsePath(path)
	if parsed == nil {
		parsed = &scanner.ParsedMedia{Quality: qualityStr}
	}

	options := make([]SlotOption, 0, len(slots))
	for _, slot := range slots {
		if slot.QualityProfileID == nil {
			continue
		}

		option := SlotOption{
			SlotID:     slot.ID,
			SlotNumber: slot.SlotNumber,
			SlotName:   slot.Name,
		}

		if profile, err := s.qualityService.Get(ctx, *slot.QualityProfileID); err == nil {
			releaseAttrs := parsed.ToReleaseAttributes()
			matchResult := quality.MatchProfileAttributes(&releaseAttrs, profile)
			qualityScore := s.calculateQualityScore(parsed)
			option.MatchScore = qualityScore + matchResult.TotalScore
			option.IsCompatible = matchResult.AllMatch
		}

		currentFile := s.getCurrentSlotFile(ctx, mediaType, mediaID, slot.ID)
		option.IsFilled = currentFile != nil
		options = append(options, option)
	}

	sort.Slice(options, func(i, j int) bool {
		return options[i].MatchScore > options[j].MatchScore
	})
	return options
}

// GetReviewQueueItemDetails returns detailed information about a review queue item.
// Req 14.3.2: Show file details, detected quality, and available slot options
func (s *Service) GetReviewQueueItemDetails(ctx context.Context, mediaType string, fileID int64) (*ReviewQueueItemDetails, error) {
	details := &ReviewQueueItemDetails{
		SlotOptions: make([]SlotOption, 0),
	}

	fi, err := s.loadReviewFileInfo(ctx, details, mediaType, fileID)
	if err != nil {
		return nil, err
	}

	details.FilePath = fi.path
	details.Quality = fi.quality
	details.Size = fi.size

	populateDetectedAttributes(details, fi.path)
	details.SlotOptions = s.buildSlotOptions(ctx, fi.path, fi.quality, fi.mediaType, fi.mediaID)

	return details, nil
}

// ReviewQueueItemDetails contains detailed information about a review queue item.
type ReviewQueueItemDetails struct {
	MediaType          string       `json:"mediaType"`
	FileID             int64        `json:"fileId"`
	FilePath           string       `json:"filePath"`
	MediaTitle         string       `json:"mediaTitle"`
	Quality            string       `json:"quality"`
	Size               int64        `json:"size"`
	DetectedQuality    string       `json:"detectedQuality,omitempty"`
	DetectedSource     string       `json:"detectedSource,omitempty"`
	DetectedHDR        string       `json:"detectedHdr,omitempty"`
	DetectedVideoCodec string       `json:"detectedVideoCodec,omitempty"`
	DetectedAudioCodec string       `json:"detectedAudioCodec,omitempty"`
	SlotOptions        []SlotOption `json:"slotOptions"`
}

// SlotOption represents a slot as an option for assignment.
type SlotOption struct {
	SlotID       int64   `json:"slotId"`
	SlotNumber   int     `json:"slotNumber"`
	SlotName     string  `json:"slotName"`
	MatchScore   float64 `json:"matchScore"`
	IsCompatible bool    `json:"isCompatible"`
	IsFilled     bool    `json:"isFilled"`
}
