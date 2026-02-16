package slots

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/slipstream/slipstream/internal/library/quality"
	"github.com/slipstream/slipstream/internal/library/scanner"
)

// DevModeChecker is a function that returns whether developer mode is enabled.
type DevModeChecker func() bool

// DebugHandlers provides HTTP handlers for debug/testing operations.
// Req 20.2.5: All debug features gated behind developerMode
type DebugHandlers struct {
	service       *Service
	isDevModeFunc DevModeChecker
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
func (h *DebugHandlers) requireDeveloperMode(_ echo.Context) error {
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
	Title            string   `json:"title"`
	Year             int      `json:"year,omitempty"`
	Season           int      `json:"season,omitempty"`
	Episode          int      `json:"episode,omitempty"`
	Quality          string   `json:"quality,omitempty"`
	Source           string   `json:"source,omitempty"`
	VideoCodec       string   `json:"videoCodec,omitempty"`
	AudioCodecs      []string `json:"audioCodecs,omitempty"`
	AudioChannels    []string `json:"audioChannels,omitempty"`
	HDRFormats       []string `json:"hdrFormats,omitempty"`
	ReleaseGroup     string   `json:"releaseGroup,omitempty"`
	IsSeasonPack     bool     `json:"isSeasonPack"`
	IsCompleteSeries bool     `json:"isCompleteSeries"`
	IsTV             bool     `json:"isTv"`

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
	Mode          string   `json:"mode"`          // "any", "required", "preferred"
	ProfileValues []string `json:"profileValues"` // Values from profile
	ReleaseValue  string   `json:"releaseValue"`  // Value from release (or comma-separated for multi)
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

	parsed := scanner.ParseFilename(input.ReleaseTitle)
	if parsed == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "failed to parse release title")
	}

	profile, err := h.service.qualityService.Get(c.Request().Context(), input.QualityProfileID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "quality profile not found")
	}

	releaseAttrs := parsed.ToReleaseAttributes()
	matchResult := quality.MatchProfileAttributes(&releaseAttrs, profile)
	qualityMatchResult := quality.MatchQuality(parsed.Quality, parsed.Source, profile)
	qualityScore := calculateQualityScoreForDebug(parsed)

	output := buildProfileMatchOutput(parsed, profile, &matchResult, qualityMatchResult, qualityScore, &releaseAttrs)
	return c.JSON(http.StatusOK, output)
}

func buildProfileMatchOutput(
	parsed *scanner.ParsedMedia,
	profile *quality.Profile,
	matchResult *quality.ProfileAttributeMatchResult,
	qualityMatchResult quality.QualityMatchResult,
	qualityScore float64,
	releaseAttrs *quality.ReleaseAttributes,
) ProfileMatchOutput {
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
		AllAttributesMatch: allMatch,
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
		HDRMatch: buildAttributeMatchResult(
			profile.HDRSettings,
			releaseAttrs.HDRFormats,
			quality.MatchHDRAttribute(releaseAttrs.HDRFormats, profile.HDRSettings),
		),
		VideoCodecMatch: buildSingleAttributeMatchResult(
			profile.VideoCodecSettings,
			releaseAttrs.VideoCodec,
			quality.MatchAttribute(releaseAttrs.VideoCodec, profile.VideoCodecSettings),
		),
		AudioCodecMatch: buildAttributeMatchResult(
			profile.AudioCodecSettings,
			releaseAttrs.AudioCodecs,
			quality.MatchAudioAttribute(releaseAttrs.AudioCodecs, profile.AudioCodecSettings),
		),
		AudioChannelMatch: buildAttributeMatchResult(
			profile.AudioChannelSettings,
			releaseAttrs.AudioChannels,
			quality.MatchAudioAttribute(releaseAttrs.AudioChannels, profile.AudioChannelSettings),
		),
	}

	return output
}

// SimulateImportInput is the request body for simulating a file import.
type SimulateImportInput struct {
	ReleaseTitle string `json:"releaseTitle"`
	MediaType    string `json:"mediaType"` // "movie" or "episode"
	MediaID      int64  `json:"mediaId"`
}

// SimulateImportOutput is the detailed output from simulating an import.
type SimulateImportOutput struct {
	Release           ParseReleaseOutput     `json:"release"`
	SlotEvaluations   []SlotEvaluationDetail `json:"slotEvaluations"`
	RecommendedSlot   *SlotEvaluationDetail  `json:"recommendedSlot,omitempty"`
	RequiresSelection bool                   `json:"requiresSelection"`
	MatchingCount     int                    `json:"matchingCount"`
	ImportAction      string                 `json:"importAction"` // "accept", "reject", "user_choice"
	ImportReason      string                 `json:"importReason"` // Explanation
}

// SlotEvaluationDetail provides detailed evaluation for a single slot.
type SlotEvaluationDetail struct {
	SlotID           int64   `json:"slotId"`
	SlotNumber       int     `json:"slotNumber"`
	SlotName         string  `json:"slotName"`
	ProfileID        *int64  `json:"profileId,omitempty"`
	ProfileName      string  `json:"profileName,omitempty"`
	MatchScore       float64 `json:"matchScore"`
	AttributeScore   float64 `json:"attributeScore"`
	QualityScore     float64 `json:"qualityScore"`
	IsEmpty          bool    `json:"isEmpty"`
	IsUpgrade        bool    `json:"isUpgrade"`
	CurrentQuality   string  `json:"currentQuality,omitempty"`
	Confidence       float64 `json:"confidence"`
	AttributesPassed bool    `json:"attributesPassed"`
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

	parsed := scanner.ParseFilename(input.ReleaseTitle)
	if parsed == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "failed to parse release title")
	}

	releaseAttrs := parsed.ToReleaseAttributes()
	qualityScore := calculateQualityScoreForDebug(parsed)

	output := SimulateImportOutput{
		Release:         buildParsedReleaseOutput(parsed, qualityScore),
		SlotEvaluations: make([]SlotEvaluationDetail, 0),
	}

	slots, err := h.service.ListEnabledWithProfiles(c.Request().Context())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	if len(slots) == 0 {
		output.ImportAction = "reject"
		output.ImportReason = "No enabled slots configured"
		return c.JSON(http.StatusOK, output)
	}

	for _, slot := range slots {
		detail := h.evaluateSlotForImport(c, slot, &releaseAttrs, parsed, qualityScore, input.MediaType, input.MediaID)
		output.SlotEvaluations = append(output.SlotEvaluations, detail)
	}

	determineImportAction(&output)
	return c.JSON(http.StatusOK, output)
}

func buildParsedReleaseOutput(parsed *scanner.ParsedMedia, qualityScore float64) ParseReleaseOutput {
	return ParseReleaseOutput{
		Title:         parsed.Title,
		Year:          parsed.Year,
		Quality:       parsed.Quality,
		Source:        parsed.Source,
		VideoCodec:    parsed.Codec,
		AudioCodecs:   parsed.AudioCodecs,
		AudioChannels: parsed.AudioChannels,
		HDRFormats:    parsed.HDRFormats,
		QualityScore:  qualityScore,
	}
}

func (h *DebugHandlers) evaluateSlotForImport(
	c echo.Context,
	slot *SlotWithProfile,
	releaseAttrs *quality.ReleaseAttributes,
	parsed *scanner.ParsedMedia,
	qualityScore float64,
	mediaType string,
	mediaID int64,
) SlotEvaluationDetail {
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
			matchResult := quality.MatchProfileAttributes(releaseAttrs, profile)
			qualityMatchResult := quality.MatchQuality(parsed.Quality, parsed.Source, profile)
			detail.AttributesPassed = matchResult.AllMatch && qualityMatchResult.Matches
			detail.AttributeScore = matchResult.TotalScore
			detail.MatchScore = qualityScore + matchResult.TotalScore
			detail.Confidence = 1.0
			if !detail.AttributesPassed {
				detail.Confidence = 0.5
			}
		}
	}

	currentFile := h.service.getCurrentSlotFile(c.Request().Context(), mediaType, mediaID, slot.ID)
	detail.IsEmpty = currentFile == nil
	if currentFile != nil {
		detail.CurrentQuality = currentFile.Quality
		detail.IsUpgrade = detail.MatchScore > currentFile.QualityScore
	}

	return detail
}

func determineImportAction(output *SimulateImportOutput) {
	var bestMatch *SlotEvaluationDetail
	var matchingCount int
	for i := range output.SlotEvaluations {
		if !output.SlotEvaluations[i].AttributesPassed {
			continue
		}
		matchingCount++
		if bestMatch == nil || output.SlotEvaluations[i].MatchScore > bestMatch.MatchScore {
			bestMatch = &output.SlotEvaluations[i]
		}
	}

	output.MatchingCount = matchingCount

	switch {
	case matchingCount == 0:
		output.ImportAction = "reject"
		output.ImportReason = "Release does not match any slot profile requirements"
	case matchingCount > 1 && hasEqualTopScores(output.SlotEvaluations, bestMatch):
		output.RequiresSelection = true
		output.ImportAction = "user_choice"
		output.ImportReason = "Multiple slots match equally - user selection required"
	default:
		output.ImportAction = "accept"
		if matchingCount == 1 {
			output.ImportReason = "Single matching slot found"
		} else {
			output.ImportReason = "Best matching slot determined"
		}
		output.RecommendedSlot = bestMatch
	}
}

func hasEqualTopScores(evals []SlotEvaluationDetail, best *SlotEvaluationDetail) bool {
	for i := range evals {
		if evals[i].AttributesPassed && &evals[i] != best && evals[i].MatchScore == best.MatchScore {
			return true
		}
	}
	return false
}

// Helper functions

func resolutionScore(q string) float64 {
	switch q {
	case "2160p":
		return 40
	case "1080p":
		return 30
	case "720p":
		return 20
	case "480p":
		return 10
	default:
		return 0
	}
}

func sourceScore(s string) float64 {
	switch s {
	case "Remux":
		return 10
	case "BluRay":
		return 8
	case "WEB-DL":
		return 6
	case "WEBRip":
		return 5
	case "HDTV":
		return 4
	case "DVDRip":
		return 2
	case "SDTV":
		return 1
	default:
		return 0
	}
}

func calculateQualityScoreForDebug(parsed *scanner.ParsedMedia) float64 {
	return resolutionScore(parsed.Quality) + sourceScore(parsed.Source)
}

func attributeMode(settings quality.AttributeSettings) string {
	switch {
	case len(settings.GetRequired()) > 0:
		return "required"
	case len(settings.GetPreferred()) > 0:
		return "preferred"
	case len(settings.GetNotAllowed()) > 0:
		return "notAllowed"
	default:
		return "any"
	}
}

func collectProfileValues(settings quality.AttributeSettings) []string {
	var vals []string
	vals = append(vals, settings.GetRequired()...)
	vals = append(vals, settings.GetPreferred()...)
	vals = append(vals, settings.GetNotAllowed()...)
	return vals
}

func mismatchReason(settings quality.AttributeSettings, releaseVal string) string {
	if len(settings.GetNotAllowed()) > 0 {
		return "Release contains a blocked (not allowed) value"
	}
	if len(settings.GetRequired()) > 0 {
		if releaseVal == "" {
			return "Required attribute missing from release"
		}
		return "Release value does not match required values"
	}
	return ""
}

func buildAttributeMatchResult(settings quality.AttributeSettings, releaseValues []string, result quality.AttributeMatchResult) AttributeMatchResult {
	releaseVal := strings.Join(releaseValues, ", ")

	matchResult := AttributeMatchResult{
		Mode:          attributeMode(settings),
		ProfileValues: collectProfileValues(settings),
		ReleaseValue:  releaseVal,
		Matches:       result.Matches,
		Score:         result.Score,
	}

	if !result.Matches {
		matchResult.Reason = mismatchReason(settings, releaseVal)
	}

	return matchResult
}

func buildSingleAttributeMatchResult(settings quality.AttributeSettings, releaseValue string, result quality.AttributeMatchResult) AttributeMatchResult {
	matchResult := AttributeMatchResult{
		Mode:          attributeMode(settings),
		ProfileValues: collectProfileValues(settings),
		ReleaseValue:  releaseValue,
		Matches:       result.Matches,
		Score:         result.Score,
	}

	if !result.Matches {
		matchResult.Reason = singleMismatchReason(settings, releaseValue)
	}

	return matchResult
}

func singleMismatchReason(settings quality.AttributeSettings, releaseValue string) string {
	if len(settings.GetNotAllowed()) > 0 {
		return "Release value is blocked (not allowed)"
	}
	if len(settings.GetRequired()) > 0 {
		if releaseValue == "" {
			return "Required attribute missing from release"
		}
		return "Release value does not match required values"
	}
	return ""
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

	slots, err := h.service.ListEnabled(ctx)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	preview := MigrationPreview{
		Movies:  make([]MovieMigrationPreview, 0, len(input.Movies)),
		TVShows: make([]TVShowMigrationPreview, 0, len(input.TVShows)),
	}

	for _, movie := range input.Movies {
		preview.Movies = append(preview.Movies, h.previewMovie(c, movie, slots))
	}

	for _, show := range input.TVShows {
		preview.TVShows = append(preview.TVShows, h.previewTVShow(c, show, slots))
	}

	preview.Summary = calculateMigrationSummary(&preview)
	return c.JSON(http.StatusOK, preview)
}

func (h *DebugHandlers) previewMovie(ctx echo.Context, movie MockMovie, slots []*Slot) MovieMigrationPreview {
	var evals []FileEvaluation
	for _, file := range movie.Files {
		eval := h.service.evaluateFileAgainstSlots(ctx.Request().Context(), file.Path, file.Quality, slots)
		eval.FileID = file.FileID
		eval.MediaID = movie.MovieID
		eval.MediaType = mediaTypeMovie
		eval.Size = file.Size
		evals = append(evals, eval)
	}

	assignments := h.service.resolveSlotAssignments(ctx.Request().Context(), evals, slots)

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

	return moviePreview
}

func (h *DebugHandlers) previewTVShow(ctx echo.Context, show MockTVShow, slots []*Slot) TVShowMigrationPreview {
	showPreview := TVShowMigrationPreview{
		SeriesID: show.SeriesID,
		Title:    show.Title,
		Seasons:  make([]SeasonMigrationPreview, 0, len(show.Seasons)),
	}

	for _, season := range show.Seasons {
		seasonPreview := h.previewSeason(ctx, season, slots)
		showPreview.TotalFiles += seasonPreview.TotalFiles
		if seasonPreview.HasConflict {
			showPreview.HasConflict = true
		}
		showPreview.Seasons = append(showPreview.Seasons, seasonPreview)
	}

	return showPreview
}

func (h *DebugHandlers) previewSeason(ctx echo.Context, season MockSeason, slots []*Slot) SeasonMigrationPreview {
	seasonPreview := SeasonMigrationPreview{
		SeasonNumber: season.SeasonNumber,
		Episodes:     make([]EpisodeMigrationPreview, 0, len(season.Episodes)),
	}

	for _, episode := range season.Episodes {
		episodePreview := h.previewEpisode(ctx, episode, slots)
		seasonPreview.TotalFiles += len(episodePreview.Files)
		if episodePreview.HasConflict {
			seasonPreview.HasConflict = true
		}
		seasonPreview.Episodes = append(seasonPreview.Episodes, episodePreview)
	}

	return seasonPreview
}

func (h *DebugHandlers) previewEpisode(ctx echo.Context, episode MockEpisode, slots []*Slot) EpisodeMigrationPreview {
	var evals []FileEvaluation
	for _, file := range episode.Files {
		eval := h.service.evaluateFileAgainstSlots(ctx.Request().Context(), file.Path, file.Quality, slots)
		eval.FileID = file.FileID
		eval.MediaID = episode.EpisodeID
		eval.MediaType = mediaTypeEpisode
		eval.Size = file.Size
		evals = append(evals, eval)
	}

	assignments := h.service.resolveSlotAssignments(ctx.Request().Context(), evals, slots)

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
	}

	return episodePreview
}

// calculateMigrationSummary calculates summary statistics for a migration preview.
func calculateMigrationSummary(preview *MigrationPreview) MigrationSummary {
	summary := MigrationSummary{
		TotalMovies:  len(preview.Movies),
		TotalTVShows: len(preview.TVShows),
	}

	for i := range preview.Movies {
		accumulateFileSummary(&summary, preview.Movies[i].Files)
	}

	for i := range preview.TVShows {
		for j := range preview.TVShows[i].Seasons {
			for k := range preview.TVShows[i].Seasons[j].Episodes {
				accumulateFileSummary(&summary, preview.TVShows[i].Seasons[j].Episodes[k].Files)
			}
		}
	}

	return summary
}

func accumulateFileSummary(summary *MigrationSummary, files []FileMigrationPreview) {
	for i := range files {
		summary.TotalFiles++
		if files[i].ProposedSlotID != nil && !files[i].NeedsReview && files[i].Conflict == "" {
			summary.FilesWithSlots++
		}
		if files[i].NeedsReview {
			summary.FilesNeedingReview++
		}
		if files[i].Conflict != "" {
			summary.Conflicts++
		}
	}
}
