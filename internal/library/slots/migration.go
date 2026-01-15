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
	SeasonNumber int                      `json:"seasonNumber"`
	Episodes     []EpisodeMigrationPreview `json:"episodes"`
	TotalFiles   int                      `json:"totalFiles"`
	HasConflict  bool                     `json:"hasConflict"`
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
	Type   string `json:"type"` // "ignore", "assign", or "unassign"
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
func (s *Service) evaluateFileAgainstSlots(ctx context.Context, path, qualityStr string, slots []*Slot) FileEvaluation {
	eval := FileEvaluation{
		Path:       path,
		Quality:    qualityStr,
		Rejections: make([]SlotRejection, 0),
	}

	// Parse the file quality from the path
	parsed := scanner.ParsePath(path)
	if parsed == nil {
		parsed = &scanner.ParsedMedia{Quality: qualityStr}
	}

	var bestSlot *Slot
	var bestScore float64 = -1

	for _, slot := range slots {
		if slot.QualityProfileID == nil {
			eval.Rejections = append(eval.Rejections, SlotRejection{
				SlotID:   slot.ID,
				SlotName: slot.Name,
				Reasons:  []string{"No quality profile assigned"},
			})
			continue
		}

		profile, err := s.qualityService.Get(ctx, *slot.QualityProfileID)
		if err != nil {
			eval.Rejections = append(eval.Rejections, SlotRejection{
				SlotID:   slot.ID,
				SlotName: slot.Name,
				Reasons:  []string{"Failed to load quality profile"},
			})
			continue
		}

		// Calculate match - both attributes AND quality must pass
		releaseAttrs := parsed.ToReleaseAttributes()
		matchResult := quality.MatchProfileAttributes(releaseAttrs, profile)
		qualityMatchResult := quality.MatchQuality(parsed.Quality, parsed.Source, profile)
		qualityScore := s.calculateQualityScore(parsed, profile)
		totalScore := qualityScore + matchResult.TotalScore

		// Collect rejection reasons if not matching
		if !matchResult.AllMatch || !qualityMatchResult.Matches {
			var reasons []string
			if !qualityMatchResult.Matches && qualityMatchResult.Reason != "" {
				reasons = append(reasons, "Quality: "+qualityMatchResult.Reason)
			}
			reasons = append(reasons, matchResult.RejectionReasons()...)
			eval.Rejections = append(eval.Rejections, SlotRejection{
				SlotID:   slot.ID,
				SlotName: slot.Name,
				Reasons:  reasons,
			})
			continue
		}

		// This slot matches - check if it's the best
		if totalScore > bestScore {
			bestScore = totalScore
			bestSlot = slot
		}
	}

	if bestSlot != nil {
		eval.BestSlotID = &bestSlot.ID
		eval.BestSlotName = bestSlot.Name
		eval.MatchScore = bestScore
		eval.CanMatch = true
	} else {
		eval.CanMatch = false
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

// resolveSlotAssignments determines optimal slot assignments for files of a single media item.
// Sorts by score, resolves conflicts, and returns assignment decisions.
// This is the ONLY place where assignment logic lives - used by both preview and execution.
func (s *Service) resolveSlotAssignments(ctx context.Context, evals []FileEvaluation, slots []*Slot) []ResolvedAssignment {
	// Sort by score descending (best matches first get priority)
	sort.Slice(evals, func(i, j int) bool {
		return evals[i].MatchScore > evals[j].MatchScore
	})

	assignments := make([]ResolvedAssignment, 0, len(evals))
	filledSlots := make(map[int64]int64) // slotID -> fileID

	for _, eval := range evals {
		assignment := ResolvedAssignment{FileEvaluation: eval}

		if !eval.CanMatch || eval.BestSlotID == nil {
			assignment.Conflict = eval.Reason
		} else if _, taken := filledSlots[*eval.BestSlotID]; taken {
			// Best slot taken by higher-scored file, try alternatives
			foundAlt := false
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
					foundAlt = true
					break
				}
			}
			if !foundAlt {
				assignment.Conflict = fmt.Sprintf("%s slot taken by higher-scored file", eval.BestSlotName)
			}
		} else {
			assignment.AssignedSlotID = eval.BestSlotID
			assignment.AssignedSlotName = eval.BestSlotName
			filledSlots[*eval.BestSlotID] = eval.FileID
		}

		assignments = append(assignments, assignment)
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
	for _, a := range assignments {
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
	RequiresPrompt      bool           `json:"requiresPrompt"`
	AffectedFilesCount  int            `json:"affectedFilesCount"`
	AffectedFiles       []SlotFileInfo `json:"affectedFiles,omitempty"`
	IncompatibleCount   int            `json:"incompatibleCount"`
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

	// Build slot lookup
	slotMap := make(map[int64]*Slot)
	for _, slot := range slots {
		slotMap[slot.ID] = slot
	}

	// Process movies
	if err := s.generateMoviePreview(ctx, preview, slots); err != nil {
		s.logger.Warn().Err(err).Msg("Error generating movie preview")
	}

	// Process TV shows
	if err := s.generateTVShowPreview(ctx, preview, slots); err != nil {
		s.logger.Warn().Err(err).Msg("Error generating TV show preview")
	}

	// Calculate summary
	s.calculateMigrationSummary(preview)

	return preview, nil
}

// generateMoviePreview generates the movie portion of the migration preview.
func (s *Service) generateMoviePreview(ctx context.Context, preview *MigrationPreview, slots []*Slot) error {
	movies, err := s.queries.ListMovies(ctx)
	if err != nil {
		return err
	}

	for _, movie := range movies {
		files, err := s.queries.ListMovieFiles(ctx, movie.ID)
		if err != nil {
			continue
		}

		if len(files) == 0 {
			continue
		}

		moviePreview := MovieMigrationPreview{
			MovieID:   movie.ID,
			Title:     movie.Title,
			Year:      int(movie.Year.Int64),
			Files:     make([]FileMigrationPreview, 0, len(files)),
			Conflicts: make([]string, 0),
		}

		// Evaluate all files using shared logic
		var evals []FileEvaluation
		for _, file := range files {
			eval := s.evaluateFileAgainstSlots(ctx, file.Path, file.Quality.String, slots)
			eval.FileID = file.ID
			eval.MediaID = movie.ID
			eval.MediaType = "movie"
			eval.Size = file.Size
			evals = append(evals, eval)
			preview.Summary.TotalFiles++
		}

		// Resolve assignments using shared logic, then convert to preview format
		assignments := s.resolveSlotAssignments(ctx, evals, slots)
		for i := range assignments {
			moviePreview.Files = append(moviePreview.Files, assignments[i].toFileMigrationPreview())
			if assignments[i].Conflict != "" {
				moviePreview.HasConflict = true
			}
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

// generateTVShowPreview generates the TV show portion of the migration preview.
func (s *Service) generateTVShowPreview(ctx context.Context, preview *MigrationPreview, slots []*Slot) error {
	// Get all series with files
	series, err := s.queries.ListSeries(ctx)
	if err != nil {
		return err
	}

	for _, show := range series {
		files, err := s.queries.ListEpisodeFilesBySeries(ctx, show.ID)
		if err != nil {
			continue
		}

		if len(files) == 0 {
			continue
		}

		tvPreview := TVShowMigrationPreview{
			SeriesID:   show.ID,
			Title:      show.Title,
			Seasons:    make([]SeasonMigrationPreview, 0),
			TotalFiles: len(files),
		}

		// Group files by season/episode
		episodeFiles := make(map[int64][]*sqlc.EpisodeFile)
		episodeInfo := make(map[int64]struct {
			seasonNumber  int
			episodeNumber int
			title         string
		})

		for _, file := range files {
			episode, err := s.queries.GetEpisode(ctx, file.EpisodeID)
			if err != nil {
				continue
			}
			episodeFiles[file.EpisodeID] = append(episodeFiles[file.EpisodeID], file)
			episodeInfo[file.EpisodeID] = struct {
				seasonNumber  int
				episodeNumber int
				title         string
			}{
				seasonNumber:  int(episode.SeasonNumber),
				episodeNumber: int(episode.EpisodeNumber),
				title:         episode.Title.String,
			}
		}

		// Group by season
		seasonEpisodes := make(map[int][]int64) // seasonNumber -> episodeIDs
		for episodeID, info := range episodeInfo {
			seasonEpisodes[info.seasonNumber] = append(seasonEpisodes[info.seasonNumber], episodeID)
		}

		// Build season previews
		for seasonNum, episodeIDs := range seasonEpisodes {
			seasonPreview := SeasonMigrationPreview{
				SeasonNumber: seasonNum,
				Episodes:     make([]EpisodeMigrationPreview, 0),
			}

			for _, episodeID := range episodeIDs {
				info := episodeInfo[episodeID]
				epPreview := EpisodeMigrationPreview{
					EpisodeID:     episodeID,
					EpisodeNumber: info.episodeNumber,
					Title:         info.title,
					Files:         make([]FileMigrationPreview, 0),
				}

				// Evaluate all files using shared logic
				var evals []FileEvaluation
				for _, file := range episodeFiles[episodeID] {
					eval := s.evaluateFileAgainstSlots(ctx, file.Path, file.Quality.String, slots)
					eval.FileID = file.ID
					eval.MediaID = episodeID
					eval.MediaType = "episode"
					eval.Size = file.Size
					evals = append(evals, eval)
					seasonPreview.TotalFiles++
					preview.Summary.TotalFiles++
				}

				// Resolve assignments using shared logic, then convert to preview format
				assignments := s.resolveSlotAssignments(ctx, evals, slots)
				for i := range assignments {
					epPreview.Files = append(epPreview.Files, assignments[i].toFileMigrationPreview())
					if assignments[i].Conflict != "" {
						epPreview.HasConflict = true
					}
				}

				if len(episodeFiles[episodeID]) > len(slots) {
					epPreview.HasConflict = true
				}

				if epPreview.HasConflict {
					seasonPreview.HasConflict = true
					tvPreview.HasConflict = true
				}

				seasonPreview.Episodes = append(seasonPreview.Episodes, epPreview)
			}

			// Sort episodes by number
			sort.Slice(seasonPreview.Episodes, func(i, j int) bool {
				return seasonPreview.Episodes[i].EpisodeNumber < seasonPreview.Episodes[j].EpisodeNumber
			})

			tvPreview.Seasons = append(tvPreview.Seasons, seasonPreview)
		}

		// Sort seasons by number
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

// calculateMigrationSummary calculates the summary statistics for a migration preview.
func (s *Service) calculateMigrationSummary(preview *MigrationPreview) {
	for _, movie := range preview.Movies {
		for _, file := range movie.Files {
			if file.ProposedSlotID != nil && !file.NeedsReview {
				preview.Summary.FilesWithSlots++
			} else {
				preview.Summary.FilesNeedingReview++
			}
			if file.Conflict != "" {
				preview.Summary.Conflicts++
			}
		}
	}

	for _, show := range preview.TVShows {
		for _, season := range show.Seasons {
			for _, episode := range season.Episodes {
				for _, file := range episode.Files {
					if file.ProposedSlotID != nil && !file.NeedsReview {
						preview.Summary.FilesWithSlots++
					} else {
						preview.Summary.FilesNeedingReview++
					}
					if file.Conflict != "" {
						preview.Summary.Conflicts++
					}
				}
			}
		}
	}
}

// ExecuteMigration executes the migration, assigning files to slots.
// Req 14.2.1: Intelligently assign existing files to slots based on quality profile matching
// Req 14.2.2: Files that can't be matched to any slot go to review queue
// Req 14.2.3: Quality profile must be assigned to slot before saving configuration
func (s *Service) ExecuteMigration(ctx context.Context, overrides []FileOverride) (*MigrationResult, error) {
	result := &MigrationResult{
		Errors: make([]string, 0),
	}

	// Build override lookup map
	overrideMap := make(map[int64]FileOverride)
	for _, o := range overrides {
		overrideMap[o.FileID] = o
	}

	// Req 14.2.3: Validate all enabled slots have profiles
	if err := s.ValidateSlotConfiguration(ctx); err != nil {
		return nil, err
	}

	slots, err := s.ListEnabled(ctx)
	if err != nil {
		return nil, err
	}

	// Build slot lookup map for manual assignments
	slotMap := make(map[int64]*Slot)
	for _, slot := range slots {
		slotMap[slot.ID] = slot
	}

	// Process movie files - group by movie, resolve assignments, execute
	movieFiles, err := s.queries.ListMovieFilesWithoutSlot(ctx)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to list movie files: %v", err))
	} else {
		// Group files by movie
		movieGroups := make(map[int64][]FileEvaluation)
		for _, file := range movieFiles {
			// Check for override
			if override, exists := overrideMap[file.ID]; exists {
				switch override.Type {
				case "ignore":
					// Skip this file entirely
					continue
				case "assign":
					// Manual assignment to specific slot
					if override.SlotID != nil {
						if slot, ok := slotMap[*override.SlotID]; ok {
							eval := FileEvaluation{
								FileID:       file.ID,
								MediaID:      file.MovieID,
								MediaType:    "movie",
								Path:         file.Path,
								Quality:      file.Quality.String,
								Size:         file.Size,
								BestSlotID:   override.SlotID,
								BestSlotName: slot.Name,
								MatchScore:   100, // Manual assignment gets perfect score
								CanMatch:     true,
							}
							movieGroups[file.MovieID] = append(movieGroups[file.MovieID], eval)
							continue
						}
					}
				case "unassign":
					// Send to review queue
					eval := FileEvaluation{
						FileID:    file.ID,
						MediaID:   file.MovieID,
						MediaType: "movie",
						Path:      file.Path,
						Quality:   file.Quality.String,
						Size:      file.Size,
						CanMatch:  false,
						Reason:    "Manually marked for review",
					}
					movieGroups[file.MovieID] = append(movieGroups[file.MovieID], eval)
					continue
				}
			}

			// Normal automatic evaluation
			eval := s.evaluateFileAgainstSlots(ctx, file.Path, file.Quality.String, slots)
			eval.FileID = file.ID
			eval.MediaID = file.MovieID
			eval.MediaType = "movie"
			eval.Size = file.Size
			movieGroups[file.MovieID] = append(movieGroups[file.MovieID], eval)
		}
		// Resolve and execute for each movie
		for _, evals := range movieGroups {
			assignments := s.resolveSlotAssignments(ctx, evals, slots)
			assigned, queued := s.executeAssignments(ctx, assignments)
			result.FilesAssigned += assigned
			result.FilesQueued += queued
		}
	}

	// Process episode files - group by episode, resolve assignments, execute
	episodeFiles, err := s.queries.ListEpisodeFilesWithoutSlot(ctx)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to list episode files: %v", err))
	} else {
		// Group files by episode
		episodeGroups := make(map[int64][]FileEvaluation)
		for _, file := range episodeFiles {
			// Check for override
			if override, exists := overrideMap[file.ID]; exists {
				switch override.Type {
				case "ignore":
					// Skip this file entirely
					continue
				case "assign":
					// Manual assignment to specific slot
					if override.SlotID != nil {
						if slot, ok := slotMap[*override.SlotID]; ok {
							eval := FileEvaluation{
								FileID:       file.ID,
								MediaID:      file.EpisodeID,
								MediaType:    "episode",
								Path:         file.Path,
								Quality:      file.Quality.String,
								Size:         file.Size,
								BestSlotID:   override.SlotID,
								BestSlotName: slot.Name,
								MatchScore:   100, // Manual assignment gets perfect score
								CanMatch:     true,
							}
							episodeGroups[file.EpisodeID] = append(episodeGroups[file.EpisodeID], eval)
							continue
						}
					}
				case "unassign":
					// Send to review queue
					eval := FileEvaluation{
						FileID:    file.ID,
						MediaID:   file.EpisodeID,
						MediaType: "episode",
						Path:      file.Path,
						Quality:   file.Quality.String,
						Size:      file.Size,
						CanMatch:  false,
						Reason:    "Manually marked for review",
					}
					episodeGroups[file.EpisodeID] = append(episodeGroups[file.EpisodeID], eval)
					continue
				}
			}

			// Normal automatic evaluation
			eval := s.evaluateFileAgainstSlots(ctx, file.Path, file.Quality.String, slots)
			eval.FileID = file.ID
			eval.MediaID = file.EpisodeID
			eval.MediaType = "episode"
			eval.Size = file.Size
			episodeGroups[file.EpisodeID] = append(episodeGroups[file.EpisodeID], eval)
		}
		// Resolve and execute for each episode
		for _, evals := range episodeGroups {
			assignments := s.resolveSlotAssignments(ctx, evals, slots)
			assigned, queued := s.executeAssignments(ctx, assignments)
			result.FilesAssigned += assigned
			result.FilesQueued += queued
		}
	}

	// Mark dry-run as completed
	if err := s.SetDryRunCompleted(ctx, true); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to mark dry-run completed: %v", err))
	}

	// Update last migration timestamp
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
func (s *Service) CheckProfileChange(ctx context.Context, slotID int64, newProfileID int64) (*ProfileChangeResult, error) {
	result := &ProfileChangeResult{
		AffectedFiles: make([]SlotFileInfo, 0),
	}

	// Get files currently assigned to this slot
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

	// Check how many files would be incompatible with the new profile
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
		matchResult := quality.MatchProfileAttributes(releaseAttrs, newProfile)
		if !matchResult.AllMatch {
			result.IncompatibleCount++
		}
	}

	return result, nil
}

// ChangeSlotProfile changes a slot's profile with the specified action.
// Req 15.1.2: Options: keep current assignments, re-evaluate (may queue non-matches), or cancel
func (s *Service) ChangeSlotProfile(ctx context.Context, req ProfileChangeRequest) (*Slot, error) {
	if req.Action == ProfileChangeCancel {
		slot, _ := s.Get(ctx, req.SlotID)
		return slot, nil
	}

	// Get current files in slot
	files, err := s.ListFilesInSlot(ctx, req.SlotID)
	if err != nil {
		return nil, err
	}

	if len(files) > 0 && req.Action == "" {
		return nil, fmt.Errorf("slot has files assigned; action required")
	}

	// Get the new profile
	newProfile, err := s.qualityService.Get(ctx, req.NewProfileID)
	if err != nil {
		return nil, fmt.Errorf("failed to get new profile: %w", err)
	}

	// Handle based on action
	switch req.Action {
	case ProfileChangeKeep:
		// Just update the profile, keep all assignments
		s.logger.Info().
			Int64("slotId", req.SlotID).
			Int64("newProfileId", req.NewProfileID).
			Int("filesKept", len(files)).
			Msg("Keeping file assignments despite profile change")

	case ProfileChangeReevaluate:
		// Re-evaluate each file and unassign those that don't match
		for _, file := range files {
			parsed := scanner.ParsePath(file.FilePath)
			if parsed == nil {
				// Can't parse - move to review queue
				if err := s.unassignFileByID(ctx, file.MediaType, file.FileID); err != nil {
					s.logger.Warn().Err(err).Int64("fileId", file.FileID).Msg("Failed to unassign file")
				}
				continue
			}

			releaseAttrs := parsed.ToReleaseAttributes()
			matchResult := quality.MatchProfileAttributes(releaseAttrs, newProfile)
			if !matchResult.AllMatch {
				// File doesn't match new profile - unassign
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

	// Update the slot's profile
	return s.SetProfile(ctx, req.SlotID, &req.NewProfileID)
}

// unassignFileByID unassigns a file from its slot by file ID.
func (s *Service) unassignFileByID(ctx context.Context, mediaType string, fileID int64) error {
	switch mediaType {
	case "movie":
		// Clear from slot assignments
		if err := s.queries.ClearMovieSlotFileByFileID(ctx, sql.NullInt64{Int64: fileID, Valid: true}); err != nil {
			return err
		}
		// Clear slot_id from file table
		return s.queries.ClearMovieFileSlot(ctx, fileID)
	case "episode":
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
		// Get the media ID for this file
		switch req.MediaType {
		case "movie":
			file, err := s.queries.GetMovieFile(ctx, req.FileID)
			if err != nil {
				return fmt.Errorf("failed to get movie file: %w", err)
			}
			return s.AssignFileToSlot(ctx, "movie", file.MovieID, *req.TargetSlotID, req.FileID)
		case "episode":
			file, err := s.queries.GetEpisodeFile(ctx, req.FileID)
			if err != nil {
				return fmt.Errorf("failed to get episode file: %w", err)
			}
			return s.AssignFileToSlot(ctx, "episode", file.EpisodeID, *req.TargetSlotID, req.FileID)
		}

	case ReviewActionDelete:
		if s.fileDeleter == nil {
			return fmt.Errorf("file deleter not configured")
		}
		return s.fileDeleter.DeleteFile(ctx, req.MediaType, req.FileID)

	case ReviewActionSkip:
		// Do nothing - file stays in review queue
		return nil
	}

	return fmt.Errorf("invalid action: %s", req.Action)
}

// GetReviewQueueItemDetails returns detailed information about a review queue item.
// Req 14.3.2: Show file details, detected quality, and available slot options
func (s *Service) GetReviewQueueItemDetails(ctx context.Context, mediaType string, fileID int64) (*ReviewQueueItemDetails, error) {
	details := &ReviewQueueItemDetails{
		SlotOptions: make([]SlotOption, 0),
	}

	var path, qualityStr string
	var size int64
	var mediaID int64

	switch mediaType {
	case "movie":
		file, err := s.queries.GetMovieFile(ctx, fileID)
		if err != nil {
			return nil, err
		}
		path = file.Path
		qualityStr = file.Quality.String
		size = file.Size
		mediaID = file.MovieID
		details.MediaType = "movie"
		details.FileID = fileID

		movie, _ := s.queries.GetMovie(ctx, file.MovieID)
		if movie != nil {
			details.MediaTitle = movie.Title
		}

	case "episode":
		file, err := s.queries.GetEpisodeFile(ctx, fileID)
		if err != nil {
			return nil, err
		}
		path = file.Path
		qualityStr = file.Quality.String
		size = file.Size
		mediaID = file.EpisodeID
		details.MediaType = "episode"
		details.FileID = fileID

		episode, _ := s.queries.GetEpisode(ctx, file.EpisodeID)
		if episode != nil {
			series, _ := s.queries.GetSeries(ctx, episode.SeriesID)
			if series != nil {
				details.MediaTitle = fmt.Sprintf("%s S%02dE%02d", series.Title, episode.SeasonNumber, episode.EpisodeNumber)
			}
		}
	}

	details.FilePath = path
	details.Quality = qualityStr
	details.Size = size

	// Parse file to get detected attributes
	if parsed := scanner.ParsePath(path); parsed != nil {
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

	// Get slot options with match scores
	slots, err := s.ListEnabled(ctx)
	if err != nil {
		return nil, err
	}

	parsed := scanner.ParsePath(path)
	if parsed == nil {
		parsed = &scanner.ParsedMedia{Quality: qualityStr}
	}

	for _, slot := range slots {
		if slot.QualityProfileID == nil {
			continue
		}

		option := SlotOption{
			SlotID:     slot.ID,
			SlotNumber: slot.SlotNumber,
			SlotName:   slot.Name,
		}

		profile, err := s.qualityService.Get(ctx, *slot.QualityProfileID)
		if err == nil {
			releaseAttrs := parsed.ToReleaseAttributes()
			matchResult := quality.MatchProfileAttributes(releaseAttrs, profile)
			qualityScore := s.calculateQualityScore(parsed, profile)
			option.MatchScore = qualityScore + matchResult.TotalScore
			option.IsCompatible = matchResult.AllMatch
		}

		// Check if slot already has a file for this media item
		currentFile := s.getCurrentSlotFile(ctx, mediaType, mediaID, slot.ID)
		option.IsFilled = currentFile != nil

		details.SlotOptions = append(details.SlotOptions, option)
	}

	// Sort by match score descending
	sort.Slice(details.SlotOptions, func(i, j int) bool {
		return details.SlotOptions[i].MatchScore > details.SlotOptions[j].MatchScore
	})

	return details, nil
}

// ReviewQueueItemDetails contains detailed information about a review queue item.
type ReviewQueueItemDetails struct {
	MediaType         string       `json:"mediaType"`
	FileID            int64        `json:"fileId"`
	FilePath          string       `json:"filePath"`
	MediaTitle        string       `json:"mediaTitle"`
	Quality           string       `json:"quality"`
	Size              int64        `json:"size"`
	DetectedQuality   string       `json:"detectedQuality,omitempty"`
	DetectedSource    string       `json:"detectedSource,omitempty"`
	DetectedHDR       string       `json:"detectedHdr,omitempty"`
	DetectedVideoCodec string      `json:"detectedVideoCodec,omitempty"`
	DetectedAudioCodec string      `json:"detectedAudioCodec,omitempty"`
	SlotOptions       []SlotOption `json:"slotOptions"`
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
