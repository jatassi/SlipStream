package slots

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/slipstream/slipstream/internal/library/quality"
	"github.com/slipstream/slipstream/internal/library/scanner"
)

// DevModeChecker is a function that returns whether developer mode is enabled.
type DevModeChecker func() bool

// DebugHandlers provides HTTP handlers for debug/testing operations.
// Req 20.2.5: All debug features gated behind developerMode
type DebugHandlers struct {
	service        *Service
	isDevModeFunc  DevModeChecker
}

// NewDebugHandlers creates new debug handlers.
func NewDebugHandlers(service *Service, isDevModeFunc DevModeChecker) *DebugHandlers {
	return &DebugHandlers{
		service:       service,
		isDevModeFunc: isDevModeFunc,
	}
}

// RegisterDebugRoutes registers debug routes.
// These routes are only functional when developerMode is enabled.
func (h *DebugHandlers) RegisterDebugRoutes(g *echo.Group) {
	g.POST("/parse-release", h.ParseRelease)
	g.POST("/profile-match", h.ProfileMatch)
	g.POST("/simulate-import", h.SimulateImport)
	g.POST("/generate-preview", h.GeneratePreview)
}

// requireDeveloperMode checks if developer mode is enabled.
func (h *DebugHandlers) requireDeveloperMode(c echo.Context) error {
	if h.isDevModeFunc == nil || !h.isDevModeFunc() {
		return echo.NewHTTPError(http.StatusForbidden, "debug features require developer mode")
	}
	return nil
}

// ParseReleaseInput is the request body for parsing a release title.
type ParseReleaseInput struct {
	ReleaseTitle string `json:"releaseTitle"`
}

// ParseReleaseOutput is the detailed output from parsing a release title.
type ParseReleaseOutput struct {
	Title           string   `json:"title"`
	Year            int      `json:"year,omitempty"`
	Season          int      `json:"season,omitempty"`
	Episode         int      `json:"episode,omitempty"`
	Quality         string   `json:"quality,omitempty"`
	Source          string   `json:"source,omitempty"`
	VideoCodec      string   `json:"videoCodec,omitempty"`
	AudioCodecs     []string `json:"audioCodecs,omitempty"`
	AudioChannels   []string `json:"audioChannels,omitempty"`
	HDRFormats      []string `json:"hdrFormats,omitempty"`
	ReleaseGroup    string   `json:"releaseGroup,omitempty"`
	IsSeasonPack    bool     `json:"isSeasonPack"`
	IsCompleteSeries bool    `json:"isCompleteSeries"`
	IsTV            bool     `json:"isTv"`

	// Computed quality score
	QualityScore float64 `json:"qualityScore"`
}

// ParseRelease parses a release title and returns detailed attribute information.
// POST /api/v1/slots/debug/parse-release
// Req 20.2.2: Mock File Import - parse release with custom quality attributes
func (h *DebugHandlers) ParseRelease(c echo.Context) error {
	if err := h.requireDeveloperMode(c); err != nil {
		return err
	}

	var input ParseReleaseInput
	if err := c.Bind(&input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if input.ReleaseTitle == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "releaseTitle is required")
	}

	parsed := scanner.ParseFilename(input.ReleaseTitle)
	if parsed == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "failed to parse release title")
	}

	output := ParseReleaseOutput{
		Title:            parsed.Title,
		Year:             parsed.Year,
		Season:           parsed.Season,
		Episode:          parsed.Episode,
		Quality:          parsed.Quality,
		Source:           parsed.Source,
		VideoCodec:       parsed.Codec,
		AudioCodecs:      parsed.AudioCodecs,
		AudioChannels:    parsed.AudioChannels,
		HDRFormats:       parsed.HDRFormats,
		ReleaseGroup:     parsed.ReleaseGroup,
		IsSeasonPack:     parsed.IsSeasonPack,
		IsCompleteSeries: parsed.IsCompleteSeries,
		IsTV:             parsed.IsTV,
		QualityScore:     calculateQualityScoreForDebug(parsed),
	}

	return c.JSON(http.StatusOK, output)
}

// ProfileMatchInput is the request body for detailed profile matching.
type ProfileMatchInput struct {
	ReleaseTitle     string `json:"releaseTitle"`
	QualityProfileID int64  `json:"qualityProfileId"`
}

// ProfileMatchOutput is the detailed output from profile matching.
type ProfileMatchOutput struct {
	// Parsed release info
	Release ParseReleaseOutput `json:"release"`

	// Profile info
	ProfileID   int64  `json:"profileId"`
	ProfileName string `json:"profileName"`

	// Overall match result
	AllAttributesMatch bool    `json:"allAttributesMatch"`
	QualityMatch       bool    `json:"qualityMatch"` // Whether quality/resolution is allowed
	TotalScore         float64 `json:"totalScore"`
	QualityScore       float64 `json:"qualityScore"`
	CombinedScore      float64 `json:"combinedScore"`

	// Individual attribute results
	QualityMatchResult QualityMatchDetail   `json:"qualityMatchResult"`
	HDRMatch           AttributeMatchResult `json:"hdrMatch"`
	VideoCodecMatch    AttributeMatchResult `json:"videoCodecMatch"`
	AudioCodecMatch    AttributeMatchResult `json:"audioCodecMatch"`
	AudioChannelMatch  AttributeMatchResult `json:"audioChannelMatch"`
}

// QualityMatchDetail shows the result of matching quality/resolution.
type QualityMatchDetail struct {
	Matches          bool    `json:"matches"`
	MatchedQuality   string  `json:"matchedQuality,omitempty"`
	MatchedQualityID int     `json:"matchedQualityId,omitempty"`
	ReleaseQuality   string  `json:"releaseQuality"`
	ReleaseSource    string  `json:"releaseSource"`
	Score            float64 `json:"score"`
	Reason           string  `json:"reason,omitempty"`
}

// AttributeMatchResult shows the result of matching a single attribute.
type AttributeMatchResult struct {
	Mode          string   `json:"mode"`            // "any", "required", "preferred"
	ProfileValues []string `json:"profileValues"`   // Values from profile
	ReleaseValue  string   `json:"releaseValue"`    // Value from release (or comma-separated for multi)
	Matches       bool     `json:"matches"`
	Score         float64  `json:"score"`
	Reason        string   `json:"reason,omitempty"` // Explanation if not matching
}

// ProfileMatch performs detailed profile matching for debugging.
// POST /api/v1/slots/debug/profile-match
// Req 20.2.3: Profile Matching Tester - input release attributes, see which slots match
func (h *DebugHandlers) ProfileMatch(c echo.Context) error {
	if err := h.requireDeveloperMode(c); err != nil {
		return err
	}

	var input ProfileMatchInput
	if err := c.Bind(&input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if input.ReleaseTitle == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "releaseTitle is required")
	}
	if input.QualityProfileID == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "qualityProfileId is required")
	}

	// Parse the release
	parsed := scanner.ParseFilename(input.ReleaseTitle)
	if parsed == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "failed to parse release title")
	}

	// Get the profile
	profile, err := h.service.qualityService.Get(c.Request().Context(), input.QualityProfileID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "quality profile not found")
	}

	releaseAttrs := parsed.ToReleaseAttributes()
	matchResult := quality.MatchProfileAttributes(releaseAttrs, profile)
	qualityMatchResult := quality.MatchQuality(parsed.Quality, parsed.Source, profile)
	qualityScore := calculateQualityScoreForDebug(parsed)

	// Overall match requires BOTH attribute match AND quality match
	allMatch := matchResult.AllMatch && qualityMatchResult.Matches

	output := ProfileMatchOutput{
		Release: ParseReleaseOutput{
			Title:         parsed.Title,
			Year:          parsed.Year,
			Quality:       parsed.Quality,
			Source:        parsed.Source,
			VideoCodec:    parsed.Codec,
			AudioCodecs:   parsed.AudioCodecs,
			AudioChannels: parsed.AudioChannels,
			HDRFormats:    parsed.HDRFormats,
			QualityScore:  qualityScore,
		},
		ProfileID:          profile.ID,
		ProfileName:        profile.Name,
		AllAttributesMatch: allMatch, // Now includes quality check
		QualityMatch:       qualityMatchResult.Matches,
		TotalScore:         matchResult.TotalScore,
		QualityScore:       qualityScore,
		CombinedScore:      qualityScore + matchResult.TotalScore,
		QualityMatchResult: QualityMatchDetail{
			Matches:          qualityMatchResult.Matches,
			MatchedQuality:   qualityMatchResult.MatchedQuality,
			MatchedQualityID: qualityMatchResult.MatchedQualityID,
			ReleaseQuality:   parsed.Quality,
			ReleaseSource:    parsed.Source,
			Score:            qualityMatchResult.Score,
			Reason:           qualityMatchResult.Reason,
		},
	}

	// Build detailed attribute results
	output.HDRMatch = buildAttributeMatchResult(
		profile.HDRSettings,
		releaseAttrs.HDRFormats,
		quality.MatchHDRAttribute(releaseAttrs.HDRFormats, profile.HDRSettings),
	)

	output.VideoCodecMatch = buildSingleAttributeMatchResult(
		profile.VideoCodecSettings,
		releaseAttrs.VideoCodec,
		quality.MatchAttribute(releaseAttrs.VideoCodec, profile.VideoCodecSettings),
	)

	output.AudioCodecMatch = buildAttributeMatchResult(
		profile.AudioCodecSettings,
		releaseAttrs.AudioCodecs,
		quality.MatchAudioAttribute(releaseAttrs.AudioCodecs, profile.AudioCodecSettings),
	)

	output.AudioChannelMatch = buildAttributeMatchResult(
		profile.AudioChannelSettings,
		releaseAttrs.AudioChannels,
		quality.MatchAudioAttribute(releaseAttrs.AudioChannels, profile.AudioChannelSettings),
	)

	return c.JSON(http.StatusOK, output)
}

// SimulateImportInput is the request body for simulating a file import.
type SimulateImportInput struct {
	ReleaseTitle string `json:"releaseTitle"`
	MediaType    string `json:"mediaType"` // "movie" or "episode"
	MediaID      int64  `json:"mediaId"`
}

// SimulateImportOutput is the detailed output from simulating an import.
type SimulateImportOutput struct {
	Release          ParseReleaseOutput        `json:"release"`
	SlotEvaluations  []SlotEvaluationDetail    `json:"slotEvaluations"`
	RecommendedSlot  *SlotEvaluationDetail     `json:"recommendedSlot,omitempty"`
	RequiresSelection bool                     `json:"requiresSelection"`
	MatchingCount    int                       `json:"matchingCount"`
	ImportAction     string                    `json:"importAction"` // "accept", "reject", "user_choice"
	ImportReason     string                    `json:"importReason"` // Explanation
}

// SlotEvaluationDetail provides detailed evaluation for a single slot.
type SlotEvaluationDetail struct {
	SlotID          int64   `json:"slotId"`
	SlotNumber      int     `json:"slotNumber"`
	SlotName        string  `json:"slotName"`
	ProfileID       *int64  `json:"profileId,omitempty"`
	ProfileName     string  `json:"profileName,omitempty"`
	MatchScore      float64 `json:"matchScore"`
	AttributeScore  float64 `json:"attributeScore"`
	QualityScore    float64 `json:"qualityScore"`
	IsEmpty         bool    `json:"isEmpty"`
	IsUpgrade       bool    `json:"isUpgrade"`
	CurrentQuality  string  `json:"currentQuality,omitempty"`
	Confidence      float64 `json:"confidence"`
	AttributesPassed bool   `json:"attributesPassed"`
}

// SimulateImport simulates importing a file and shows slot assignment logic.
// POST /api/v1/slots/debug/simulate-import
// Req 20.2.2: Mock File Import - simulate file imports with custom quality attributes
func (h *DebugHandlers) SimulateImport(c echo.Context) error {
	if err := h.requireDeveloperMode(c); err != nil {
		return err
	}

	var input SimulateImportInput
	if err := c.Bind(&input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if input.ReleaseTitle == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "releaseTitle is required")
	}
	if input.MediaType == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "mediaType is required")
	}
	if input.MediaID == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "mediaId is required")
	}

	// Parse the release
	parsed := scanner.ParseFilename(input.ReleaseTitle)
	if parsed == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "failed to parse release title")
	}

	releaseAttrs := parsed.ToReleaseAttributes()
	qualityScore := calculateQualityScoreForDebug(parsed)

	output := SimulateImportOutput{
		Release: ParseReleaseOutput{
			Title:         parsed.Title,
			Year:          parsed.Year,
			Quality:       parsed.Quality,
			Source:        parsed.Source,
			VideoCodec:    parsed.Codec,
			AudioCodecs:   parsed.AudioCodecs,
			AudioChannels: parsed.AudioChannels,
			HDRFormats:    parsed.HDRFormats,
			QualityScore:  qualityScore,
		},
		SlotEvaluations: make([]SlotEvaluationDetail, 0),
	}

	// Get enabled slots with profiles
	slots, err := h.service.ListEnabledWithProfiles(c.Request().Context())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	if len(slots) == 0 {
		output.ImportAction = "reject"
		output.ImportReason = "No enabled slots configured"
		return c.JSON(http.StatusOK, output)
	}

	// Evaluate against each slot
	for _, slot := range slots {
		detail := SlotEvaluationDetail{
			SlotID:       slot.ID,
			SlotNumber:   slot.SlotNumber,
			SlotName:     slot.Name,
			QualityScore: qualityScore,
		}

		if slot.QualityProfileID != nil {
			detail.ProfileID = slot.QualityProfileID
			profile, err := h.service.qualityService.Get(c.Request().Context(), *slot.QualityProfileID)
			if err == nil {
				detail.ProfileName = profile.Name

				// Calculate attribute match (HDR, codecs, etc.)
				matchResult := quality.MatchProfileAttributes(releaseAttrs, profile)
				// Check if quality/resolution is allowed in profile
				qualityMatchResult := quality.MatchQuality(parsed.Quality, parsed.Source, profile)

				// Both must pass for a full match
				detail.AttributesPassed = matchResult.AllMatch && qualityMatchResult.Matches
				detail.AttributeScore = matchResult.TotalScore
				detail.MatchScore = qualityScore + matchResult.TotalScore
				detail.Confidence = 1.0
				if !detail.AttributesPassed {
					detail.Confidence = 0.5
				}
			}
		}

		// Check current slot status
		currentFile := h.service.getCurrentSlotFile(c.Request().Context(), input.MediaType, input.MediaID, slot.ID)
		detail.IsEmpty = currentFile == nil
		if currentFile != nil {
			detail.CurrentQuality = currentFile.Quality
			detail.IsUpgrade = detail.MatchScore > currentFile.QualityScore
		}

		output.SlotEvaluations = append(output.SlotEvaluations, detail)
	}

	// Determine recommended slot and action
	var bestMatch *SlotEvaluationDetail
	var matchingCount int
	for i := range output.SlotEvaluations {
		if output.SlotEvaluations[i].AttributesPassed {
			matchingCount++
			if bestMatch == nil || output.SlotEvaluations[i].MatchScore > bestMatch.MatchScore {
				bestMatch = &output.SlotEvaluations[i]
			}
		}
	}

	output.MatchingCount = matchingCount

	if matchingCount == 0 {
		output.ImportAction = "reject"
		output.ImportReason = "Release does not match any slot profile requirements"
	} else if matchingCount > 1 {
		// Check if scores are equal
		var equalScores bool
		for _, eval := range output.SlotEvaluations {
			if eval.AttributesPassed && &eval != bestMatch && eval.MatchScore == bestMatch.MatchScore {
				equalScores = true
				break
			}
		}
		if equalScores {
			output.RequiresSelection = true
			output.ImportAction = "user_choice"
			output.ImportReason = "Multiple slots match equally - user selection required"
		} else {
			output.ImportAction = "accept"
			output.ImportReason = "Best matching slot determined"
			output.RecommendedSlot = bestMatch
		}
	} else {
		output.ImportAction = "accept"
		output.ImportReason = "Single matching slot found"
		output.RecommendedSlot = bestMatch
	}

	return c.JSON(http.StatusOK, output)
}

// Helper functions

func calculateQualityScoreForDebug(parsed *scanner.ParsedMedia) float64 {
	var score float64

	switch parsed.Quality {
	case "2160p":
		score += 40
	case "1080p":
		score += 30
	case "720p":
		score += 20
	case "480p":
		score += 10
	}

	switch parsed.Source {
	case "Remux":
		score += 10
	case "BluRay":
		score += 8
	case "WEB-DL":
		score += 6
	case "WEBRip":
		score += 5
	case "HDTV":
		score += 4
	case "DVDRip":
		score += 2
	case "SDTV":
		score += 1
	}

	return score
}

func buildAttributeMatchResult(settings quality.AttributeSettings, releaseValues []string, result quality.AttributeMatchResult) AttributeMatchResult {
	// Determine effective mode based on per-item settings
	mode := "any"
	requiredValues := settings.GetRequired()
	preferredValues := settings.GetPreferred()
	notAllowedValues := settings.GetNotAllowed()

	if len(requiredValues) > 0 {
		mode = "required"
	} else if len(preferredValues) > 0 {
		mode = "preferred"
	} else if len(notAllowedValues) > 0 {
		mode = "notAllowed"
	}

	releaseVal := ""
	if len(releaseValues) > 0 {
		for i, v := range releaseValues {
			if i > 0 {
				releaseVal += ", "
			}
			releaseVal += v
		}
	}

	// Build profile values from all non-any items
	var profileValues []string
	profileValues = append(profileValues, requiredValues...)
	profileValues = append(profileValues, preferredValues...)
	profileValues = append(profileValues, notAllowedValues...)

	matchResult := AttributeMatchResult{
		Mode:          mode,
		ProfileValues: profileValues,
		ReleaseValue:  releaseVal,
		Matches:       result.Matches,
		Score:         result.Score,
	}

	if !result.Matches {
		if len(notAllowedValues) > 0 {
			matchResult.Reason = "Release contains a blocked (not allowed) value"
		} else if len(requiredValues) > 0 {
			if releaseVal == "" {
				matchResult.Reason = "Required attribute missing from release"
			} else {
				matchResult.Reason = "Release value does not match required values"
			}
		}
	}

	return matchResult
}

func buildSingleAttributeMatchResult(settings quality.AttributeSettings, releaseValue string, result quality.AttributeMatchResult) AttributeMatchResult {
	// Determine effective mode based on per-item settings
	mode := "any"
	requiredValues := settings.GetRequired()
	preferredValues := settings.GetPreferred()
	notAllowedValues := settings.GetNotAllowed()

	if len(requiredValues) > 0 {
		mode = "required"
	} else if len(preferredValues) > 0 {
		mode = "preferred"
	} else if len(notAllowedValues) > 0 {
		mode = "notAllowed"
	}

	// Build profile values from all non-any items
	var profileValues []string
	profileValues = append(profileValues, requiredValues...)
	profileValues = append(profileValues, preferredValues...)
	profileValues = append(profileValues, notAllowedValues...)

	matchResult := AttributeMatchResult{
		Mode:          mode,
		ProfileValues: profileValues,
		ReleaseValue:  releaseValue,
		Matches:       result.Matches,
		Score:         result.Score,
	}

	if !result.Matches {
		if len(notAllowedValues) > 0 {
			matchResult.Reason = "Release value is blocked (not allowed)"
		} else if len(requiredValues) > 0 {
			if releaseValue == "" {
				matchResult.Reason = "Required attribute missing from release"
			} else {
				matchResult.Reason = "Release value does not match required values"
			}
		}
	}

	return matchResult
}

// GeneratePreviewInput is the request body for generating a mock migration preview.
type GeneratePreviewInput struct {
	Movies  []MockMovie  `json:"movies"`
	TVShows []MockTVShow `json:"tvShows"`
}

// MockMovie represents a mock movie with files for preview generation.
type MockMovie struct {
	MovieID int64      `json:"movieId"`
	Title   string     `json:"title"`
	Year    int        `json:"year,omitempty"`
	Files   []MockFile `json:"files"`
}

// MockTVShow represents a mock TV show with seasons for preview generation.
type MockTVShow struct {
	SeriesID int64        `json:"seriesId"`
	Title    string       `json:"title"`
	Seasons  []MockSeason `json:"seasons"`
}

// MockSeason represents a mock season with episodes.
type MockSeason struct {
	SeasonNumber int           `json:"seasonNumber"`
	Episodes     []MockEpisode `json:"episodes"`
}

// MockEpisode represents a mock episode with files.
type MockEpisode struct {
	EpisodeID     int64      `json:"episodeId"`
	EpisodeNumber int        `json:"episodeNumber"`
	Title         string     `json:"title,omitempty"`
	Files         []MockFile `json:"files"`
}

// MockFile represents a mock file for evaluation.
type MockFile struct {
	FileID  int64  `json:"fileId"`
	Path    string `json:"path"`    // Release title is parsed from this
	Quality string `json:"quality"` // Display quality (e.g., "2160p WEB-DL")
	Size    int64  `json:"size"`
}

// GeneratePreview generates a migration preview from mock data using the exact same
// evaluation logic as real files. This allows testing slot configuration without real library data.
// POST /api/v1/slots/debug/generate-preview
func (h *DebugHandlers) GeneratePreview(c echo.Context) error {
	if err := h.requireDeveloperMode(c); err != nil {
		return err
	}

	var input GeneratePreviewInput
	if err := c.Bind(&input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	ctx := c.Request().Context()

	// Get enabled slots
	slots, err := h.service.ListEnabled(ctx)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	preview := MigrationPreview{
		Movies:  make([]MovieMigrationPreview, 0, len(input.Movies)),
		TVShows: make([]TVShowMigrationPreview, 0, len(input.TVShows)),
	}

	// Process movies - uses the same logic as real migration preview
	for _, movie := range input.Movies {
		var evals []FileEvaluation
		for _, file := range movie.Files {
			eval := h.service.evaluateFileAgainstSlots(ctx, file.Path, file.Quality, slots)
			eval.FileID = file.FileID
			eval.MediaID = movie.MovieID
			eval.MediaType = "movie"
			eval.Size = file.Size
			evals = append(evals, eval)
		}

		assignments := h.service.resolveSlotAssignments(ctx, evals, slots)

		moviePreview := MovieMigrationPreview{
			MovieID: movie.MovieID,
			Title:   movie.Title,
			Year:    movie.Year,
			Files:   make([]FileMigrationPreview, 0, len(assignments)),
		}

		for i := range assignments {
			filePreview := assignments[i].toFileMigrationPreview()
			if filePreview.NeedsReview || filePreview.Conflict != "" {
				moviePreview.HasConflict = true
				if filePreview.Conflict != "" {
					moviePreview.Conflicts = append(moviePreview.Conflicts, filePreview.Conflict)
				}
			}
			moviePreview.Files = append(moviePreview.Files, filePreview)
		}

		preview.Movies = append(preview.Movies, moviePreview)
	}

	// Process TV shows
	for _, show := range input.TVShows {
		showPreview := TVShowMigrationPreview{
			SeriesID: show.SeriesID,
			Title:    show.Title,
			Seasons:  make([]SeasonMigrationPreview, 0, len(show.Seasons)),
		}

		for _, season := range show.Seasons {
			seasonPreview := SeasonMigrationPreview{
				SeasonNumber: season.SeasonNumber,
				Episodes:     make([]EpisodeMigrationPreview, 0, len(season.Episodes)),
			}

			for _, episode := range season.Episodes {
				var evals []FileEvaluation
				for _, file := range episode.Files {
					eval := h.service.evaluateFileAgainstSlots(ctx, file.Path, file.Quality, slots)
					eval.FileID = file.FileID
					eval.MediaID = episode.EpisodeID
					eval.MediaType = "episode"
					eval.Size = file.Size
					evals = append(evals, eval)
				}

				assignments := h.service.resolveSlotAssignments(ctx, evals, slots)

				episodePreview := EpisodeMigrationPreview{
					EpisodeID:     episode.EpisodeID,
					EpisodeNumber: episode.EpisodeNumber,
					Title:         episode.Title,
					Files:         make([]FileMigrationPreview, 0, len(assignments)),
				}

				for i := range assignments {
					filePreview := assignments[i].toFileMigrationPreview()
					if filePreview.NeedsReview || filePreview.Conflict != "" {
						episodePreview.HasConflict = true
					}
					episodePreview.Files = append(episodePreview.Files, filePreview)
					seasonPreview.TotalFiles++
					showPreview.TotalFiles++
				}

				if episodePreview.HasConflict {
					seasonPreview.HasConflict = true
				}
				seasonPreview.Episodes = append(seasonPreview.Episodes, episodePreview)
			}

			if seasonPreview.HasConflict {
				showPreview.HasConflict = true
			}
			showPreview.Seasons = append(showPreview.Seasons, seasonPreview)
		}

		preview.TVShows = append(preview.TVShows, showPreview)
	}

	// Calculate summary
	preview.Summary = calculateMigrationSummary(preview)

	return c.JSON(http.StatusOK, preview)
}

// calculateMigrationSummary calculates summary statistics for a migration preview.
func calculateMigrationSummary(preview MigrationPreview) MigrationSummary {
	summary := MigrationSummary{
		TotalMovies:  len(preview.Movies),
		TotalTVShows: len(preview.TVShows),
	}

	for _, movie := range preview.Movies {
		for _, file := range movie.Files {
			summary.TotalFiles++
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
	}

	for _, show := range preview.TVShows {
		for _, season := range show.Seasons {
			for _, episode := range season.Episodes {
				for _, file := range episode.Files {
					summary.TotalFiles++
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
			}
		}
	}

	return summary
}
