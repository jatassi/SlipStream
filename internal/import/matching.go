package importer

import (
	"context"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/slipstream/slipstream/internal/library/movies"
	"github.com/slipstream/slipstream/internal/library/tv"
)

const (
	mediaTypeMovie   = "movie"
	mediaTypeEpisode = "episode"
	mediaSeason      = "season"
	mediaSeries      = "series"
	fileStatusFailed = "failed"
)

// matchToLibrary attempts to match a file to a library item.
func (s *Service) matchToLibrary(ctx context.Context, path string, mapping *DownloadMapping) (*LibraryMatch, error) {
	return s.matchToLibraryWithSettings(ctx, path, mapping, nil)
}

// matchToLibraryWithSettings attempts to match a file to a library item using provided settings.
func (s *Service) matchToLibraryWithSettings(ctx context.Context, path string, mapping *DownloadMapping, settings *ImportSettings) (*LibraryMatch, error) {
	if settings == nil {
		settings = s.loadDefaultSettings(ctx)
	}

	queueMatch := s.getQueueMatch(ctx, mapping)
	parsedMatch := s.matchFromParse(ctx, path)

	finalMatch, err := s.resolveMatchConflict(ctx, path, queueMatch, parsedMatch, settings)
	if err != nil {
		return nil, err
	}
	if finalMatch == nil {
		return nil, ErrNoMatch
	}

	return finalMatch, nil
}

func (s *Service) loadDefaultSettings(ctx context.Context) *ImportSettings {
	loaded, err := s.GetSettings(ctx)
	if err != nil {
		s.logger.Warn().Err(err).Msg("Failed to load settings for matching, using defaults")
		defaults := DefaultImportSettings()
		return &defaults
	}
	return loaded
}

func (s *Service) getQueueMatch(ctx context.Context, mapping *DownloadMapping) *LibraryMatch {
	if mapping == nil {
		return nil
	}
	return s.matchFromMapping(ctx, mapping)
}

func (s *Service) resolveMatchConflict(ctx context.Context, path string, queueMatch, parsedMatch *LibraryMatch, settings *ImportSettings) (*LibraryMatch, error) {
	if queueMatch == nil {
		return parsedMatch, nil
	}
	if parsedMatch == nil {
		return queueMatch, nil
	}

	if !matchesAreCompatible(queueMatch, parsedMatch) {
		return s.handleIncompatibleMatches(path, queueMatch, parsedMatch, settings)
	}

	s.enrichQueueMatch(queueMatch, parsedMatch)

	if queueMatch.EpisodeID == nil && queueMatch.SeriesID != nil {
		if epMatch := s.enrichSeasonPackMatch(ctx, path, queueMatch); epMatch != nil {
			return epMatch, nil
		}
	}

	return queueMatch, nil
}

func (s *Service) handleIncompatibleMatches(path string, queueMatch, parsedMatch *LibraryMatch, settings *ImportSettings) (*LibraryMatch, error) {
	switch settings.MatchConflictBehavior {
	case MatchTrustQueue:
		s.logger.Warn().
			Str("path", path).
			Str("behavior", "trust_queue").
			Msg("Queue mapping doesn't match parsed info, using queue")
		return queueMatch, nil

	case MatchTrustParse:
		s.logger.Warn().
			Str("path", path).
			Str("behavior", "trust_parse").
			Msg("Queue mapping doesn't match parsed info, using parsed")
		return parsedMatch, nil

	case MatchFail:
		s.logger.Warn().
			Str("path", path).
			Str("behavior", "fail").
			Msg("Queue mapping doesn't match parsed info, failing import")
		return nil, ErrMatchConflict
	}
	return queueMatch, nil
}

func (s *Service) enrichQueueMatch(queueMatch, parsedMatch *LibraryMatch) {
	if queueMatch.MediaType != mediaTypeEpisode || queueMatch.EpisodeID != nil || parsedMatch.EpisodeID == nil {
		return
	}

	queueMatch.EpisodeID = parsedMatch.EpisodeID
	queueMatch.EpisodeIDs = parsedMatch.EpisodeIDs
	if parsedMatch.SeasonNum != nil {
		queueMatch.SeasonNum = parsedMatch.SeasonNum
	}

	if parsedMatch.IsUpgrade {
		queueMatch.IsUpgrade = parsedMatch.IsUpgrade
		queueMatch.ExistingFile = parsedMatch.ExistingFile
		queueMatch.ExistingFileID = parsedMatch.ExistingFileID
	}
}

// matchFromMapping creates a LibraryMatch from a queue mapping.
func (s *Service) matchFromMapping(ctx context.Context, mapping *DownloadMapping) *LibraryMatch {
	match := &LibraryMatch{
		Source:     "queue",
		Confidence: 1.0,
	}

	switch {
	case mapping.MediaType == mediaTypeMovie && mapping.MovieID != nil:
		s.populateMovieMatch(ctx, match, mapping)
	case mapping.MediaType == mediaTypeEpisode && mapping.SeriesID != nil:
		s.populateEpisodeMatch(ctx, match, mapping)
	case (mapping.MediaType == mediaSeason || mapping.MediaType == mediaSeries) && mapping.SeriesID != nil:
		s.populateSeasonMatch(ctx, match, mapping)
	}

	return match
}

func (s *Service) populateMovieMatch(ctx context.Context, match *LibraryMatch, mapping *DownloadMapping) {
	match.MediaType = mediaTypeMovie
	match.MovieID = mapping.MovieID

	movie, err := s.movies.Get(ctx, *mapping.MovieID)
	if err != nil || movie.Path == "" {
		return
	}

	match.RootFolder = filepath.Dir(movie.Path)
	if len(movie.MovieFiles) > 0 {
		match.IsUpgrade = true
		match.ExistingFile = movie.MovieFiles[0].Path
		match.ExistingFileID = &movie.MovieFiles[0].ID
	}
}

func (s *Service) populateEpisodeMatch(ctx context.Context, match *LibraryMatch, mapping *DownloadMapping) {
	match.MediaType = mediaTypeEpisode
	match.SeriesID = mapping.SeriesID
	match.SeasonNum = mapping.SeasonNumber
	if mapping.EpisodeID != nil {
		match.EpisodeID = mapping.EpisodeID
	}

	series, err := s.tv.GetSeries(ctx, *mapping.SeriesID)
	if err == nil && series.Path != "" {
		match.RootFolder = series.Path
	}
}

func (s *Service) populateSeasonMatch(ctx context.Context, match *LibraryMatch, mapping *DownloadMapping) {
	match.MediaType = mediaTypeEpisode
	match.SeriesID = mapping.SeriesID
	match.SeasonNum = mapping.SeasonNumber

	series, err := s.tv.GetSeries(ctx, *mapping.SeriesID)
	if err == nil && series.Path != "" {
		match.RootFolder = series.Path
	}
}

// matchFromParse attempts to match a file by parsing its filename.
func (s *Service) matchFromParse(ctx context.Context, path string) *LibraryMatch {
	filename := filepath.Base(path)
	filename = strings.TrimSuffix(filename, filepath.Ext(filename))

	// Try TV first
	if match := s.matchTVFromParse(ctx, filename); match != nil {
		return match
	}

	// Try movie
	if match := s.matchMovieFromParse(ctx, filename); match != nil {
		return match
	}

	return nil
}

// TV show patterns
var (
	// Standard: Show.Name.S01E02, Show.Name.1x02, Show Name - S01E02
	tvPattern1 = regexp.MustCompile(`(?i)^(.+?)[.\s_-]+[Ss](\d{1,2})[Ee](\d{1,2})`)
	tvPattern2 = regexp.MustCompile(`(?i)^(.+?)[.\s_-]+(\d{1,2})x(\d{1,2})`)

	// Spelled out: Show.Season.1.Episode.01
	tvPatternSpelled = regexp.MustCompile(`(?i)^(.+?)[.\s_-]+[Ss]eason[.\s_-]+(\d{1,2})[.\s_-]+[Ee]pisode[.\s_-]+(\d{1,2})`)

	// Multi-episode: S01E01-E03, S01E01E02E03
	multiEpPattern = regexp.MustCompile(`(?i)[Ss](\d{1,2})[Ee](\d{1,2})(?:[Ee-](\d{1,2}))+`)
)

func (s *Service) matchTVFromParse(ctx context.Context, filename string) *LibraryMatch {
	seriesTitle, season, episode := s.extractTVInfo(filename)
	if seriesTitle == "" || (season == 0 && episode == 0) {
		return nil
	}

	series, err := s.searchSeries(ctx, TVSearchOptions{Title: seriesTitle})
	if err != nil || series == nil {
		return nil
	}

	match := &LibraryMatch{
		MediaType:  "episode",
		SeriesID:   &series.ID,
		SeasonNum:  &season,
		Source:     "parse",
		Confidence: 0.8,
		RootFolder: series.Path,
	}

	s.populateEpisodeInfo(ctx, match, series.ID, season, episode)

	if multiMatches := multiEpPattern.FindAllStringSubmatch(filename, -1); len(multiMatches) > 0 {
		match.EpisodeIDs = s.extractMultiEpisodeIDs(ctx, series.ID, season, filename)
	}

	return match
}

func (s *Service) extractTVInfo(filename string) (title string, season, episode int) {
	if matches := tvPattern1.FindStringSubmatch(filename); len(matches) >= 4 {
		season, _ := strconv.Atoi(matches[2])
		episode, _ := strconv.Atoi(matches[3])
		return cleanTitle(matches[1]), season, episode
	}
	if matches := tvPattern2.FindStringSubmatch(filename); len(matches) >= 4 {
		season, _ := strconv.Atoi(matches[2])
		episode, _ := strconv.Atoi(matches[3])
		return cleanTitle(matches[1]), season, episode
	}
	if matches := tvPatternSpelled.FindStringSubmatch(filename); len(matches) >= 4 {
		season, _ := strconv.Atoi(matches[2])
		episode, _ := strconv.Atoi(matches[3])
		return cleanTitle(matches[1]), season, episode
	}
	return "", 0, 0
}

func (s *Service) populateEpisodeInfo(ctx context.Context, match *LibraryMatch, seriesID int64, season, episode int) {
	episodes, err := s.tv.ListEpisodes(ctx, seriesID, &season)
	if err != nil {
		return
	}

	for _, ep := range episodes {
		if ep.EpisodeNumber == episode {
			match.EpisodeID = &ep.ID
			if ep.EpisodeFile != nil {
				match.IsUpgrade = true
				match.ExistingFile = ep.EpisodeFile.Path
				match.ExistingFileID = &ep.EpisodeFile.ID
			}
			break
		}
	}
}

func (s *Service) matchMovieFromParse(ctx context.Context, filename string) *LibraryMatch {
	// Movie pattern: Title (Year) or Title.Year
	moviePattern := regexp.MustCompile(`(?i)^(.+?)[.\s_-]*[(\[]?(\d{4})[)\]]?`)

	matches := moviePattern.FindStringSubmatch(filename)
	if len(matches) < 3 {
		return nil
	}

	title := cleanTitle(matches[1])
	year, _ := strconv.Atoi(matches[2])

	// Search for movie in library
	searchOpts := MovieSearchOptions{
		Title: title,
		Year:  year,
	}
	movie, err := s.searchMovie(ctx, searchOpts)
	if err != nil || movie == nil {
		return nil
	}

	match := &LibraryMatch{
		MediaType:  "movie",
		MovieID:    &movie.ID,
		Source:     "parse",
		Confidence: 0.8,
		RootFolder: filepath.Dir(movie.Path),
	}

	if len(movie.MovieFiles) > 0 {
		match.IsUpgrade = true
		match.ExistingFile = movie.MovieFiles[0].Path
		match.ExistingFileID = &movie.MovieFiles[0].ID
	}

	return match
}

// TVSearchOptions for searching series.
type TVSearchOptions struct {
	Title string
	Year  int
}

// MovieSearchOptions for searching movies.
type MovieSearchOptions struct {
	Title string
	Year  int
}

// searchSeries searches for a series by title.
func (s *Service) searchSeries(ctx context.Context, opts TVSearchOptions) (*SeriesInfo, error) {
	normalizedTitle := normalizeTitle(opts.Title)

	// Use the first word of the cleaned title for broader SQL matching
	// This handles cases where DB has "Show: Subtitle" but we're searching for "show subtitle"
	searchTerm := cleanTitle(opts.Title)
	words := strings.Fields(searchTerm)
	if len(words) > 0 {
		searchTerm = words[0]
	}

	// Search in library using first word for broader matching
	series, err := s.tv.ListSeries(ctx, tv.ListSeriesOptions{
		Search: searchTerm,
	})
	if err != nil {
		return nil, err
	}

	// Find best match using normalized comparison
	var bestMatch *SeriesInfo
	var bestScore float64

	for _, ser := range series {
		score := calculateTitleSimilarity(normalizedTitle, normalizeTitle(ser.Title))
		if score > bestScore && score > 0.7 {
			bestScore = score
			bestMatch = &SeriesInfo{
				ID:    ser.ID,
				Title: ser.Title,
				Path:  ser.Path,
			}
		}
	}

	return bestMatch, nil
}

// SeriesInfo is a simplified series representation for matching.
type SeriesInfo struct {
	ID    int64
	Title string
	Path  string
}

// MovieInfo is a simplified movie representation for matching.
type MovieInfo struct {
	ID    int64
	Title string
	Year  int
	Path  string
}

// searchMovie searches for a movie by title and year.
func (s *Service) searchMovie(ctx context.Context, opts MovieSearchOptions) (*MovieWithFiles, error) {
	normalizedTitle := normalizeTitle(opts.Title)

	// Use the first word of the cleaned title for broader SQL matching
	// This handles cases where DB has "Tron: Ares" but we're searching for "tron ares"
	searchTerm := cleanTitle(opts.Title)
	words := strings.Fields(searchTerm)
	if len(words) > 0 {
		searchTerm = words[0]
	}

	// Search in library using first word for broader matching
	moviesList, err := s.movies.List(ctx, movies.ListMoviesOptions{
		Search: searchTerm,
	})
	if err != nil {
		return nil, err
	}

	// Find best match using normalized comparison
	var bestMatch *MovieWithFiles
	var bestScore float64

	for _, m := range moviesList {
		score := calculateTitleSimilarity(normalizedTitle, normalizeTitle(m.Title))

		// Boost score if year matches
		if opts.Year > 0 && m.Year == opts.Year {
			score += 0.2
		}

		if score > bestScore && score > 0.7 {
			bestScore = score
			// Get full movie with files
			fullMovie, err := s.movies.Get(ctx, m.ID)
			if err == nil {
				bestMatch = &MovieWithFiles{
					ID:         fullMovie.ID,
					Title:      fullMovie.Title,
					Year:       fullMovie.Year,
					Path:       fullMovie.Path,
					MovieFiles: fullMovie.MovieFiles,
				}
			}
		}
	}

	return bestMatch, nil
}

// MovieWithFiles represents a movie with its files.
type MovieWithFiles struct {
	ID         int64
	Title      string
	Year       int
	Path       string
	MovieFiles []movies.MovieFile
}

// extractMultiEpisodeIDs extracts episode IDs for multi-episode files.
func (s *Service) extractMultiEpisodeIDs(ctx context.Context, seriesID int64, season int, filename string) []int64 {
	var ids []int64

	// Find all episode numbers in the filename
	epPattern := regexp.MustCompile(`[Ee](\d{1,2})`)
	matches := epPattern.FindAllStringSubmatch(filename, -1)

	episodes, err := s.tv.ListEpisodes(ctx, seriesID, &season)
	if err != nil {
		return nil
	}

	episodeMap := make(map[int]int64)
	for _, ep := range episodes {
		episodeMap[ep.EpisodeNumber] = ep.ID
	}

	for _, match := range matches {
		if len(match) >= 2 {
			epNum, err := strconv.Atoi(match[1])
			if err == nil {
				if id, ok := episodeMap[epNum]; ok {
					ids = append(ids, id)
				}
			}
		}
	}

	return ids
}

// cleanTitle cleans a title from a filename.
func cleanTitle(title string) string {
	// Replace separators with spaces
	title = strings.ReplaceAll(title, ".", " ")
	title = strings.ReplaceAll(title, "_", " ")
	title = strings.ReplaceAll(title, "-", " ")

	// Replace common punctuation with spaces (colons, ampersands, etc.)
	title = strings.ReplaceAll(title, ":", " ")
	title = strings.ReplaceAll(title, "&", " ")
	title = strings.ReplaceAll(title, "/", " ")

	// Remove apostrophes entirely (don't replace with space)
	title = strings.ReplaceAll(title, "'", "")
	title = strings.ReplaceAll(title, "'", "")

	// Remove year patterns in parentheses/brackets like (2017) or [2017]
	yearPattern := regexp.MustCompile(`\s*[\(\[]\d{4}[\)\]]`)
	title = yearPattern.ReplaceAllString(title, "")

	// Collapse multiple spaces
	title = regexp.MustCompile(`\s+`).ReplaceAllString(title, " ")

	return strings.TrimSpace(title)
}

// normalizeTitle normalizes a title for comparison.
func normalizeTitle(title string) string {
	title = strings.ToLower(title)
	title = cleanTitle(title)

	// Remove common prefixes
	prefixes := []string{"the ", "a ", "an "}
	for _, prefix := range prefixes {
		if strings.HasPrefix(title, prefix) {
			title = strings.TrimPrefix(title, prefix)
			break
		}
	}

	return title
}

// calculateTitleSimilarity calculates similarity between two titles.
func calculateTitleSimilarity(a, b string) float64 {
	a = normalizeTitle(a)
	b = normalizeTitle(b)

	if a == b {
		return 1.0
	}

	// Simple word overlap similarity
	wordsA := strings.Fields(a)
	wordsB := strings.Fields(b)

	if len(wordsA) == 0 || len(wordsB) == 0 {
		return 0.0
	}

	matches := 0
	for _, wa := range wordsA {
		for _, wb := range wordsB {
			if wa == wb {
				matches++
				break
			}
		}
	}

	// Use Jaccard-like similarity
	union := len(wordsA) + len(wordsB) - matches
	if union == 0 {
		return 0.0
	}

	return float64(matches) / float64(union)
}

// matchesAreCompatible checks if two matches refer to the same content.
func matchesAreCompatible(a, b *LibraryMatch) bool {
	if a.MediaType != b.MediaType {
		return false
	}

	if a.MediaType == mediaTypeMovie {
		return matchesAreCompatibleMovies(a, b)
	}

	return matchesAreCompatibleEpisodes(a, b)
}

func matchesAreCompatibleMovies(a, b *LibraryMatch) bool {
	return a.MovieID != nil && b.MovieID != nil && *a.MovieID == *b.MovieID
}

func matchesAreCompatibleEpisodes(a, b *LibraryMatch) bool {
	if a.SeriesID == nil || b.SeriesID == nil || *a.SeriesID != *b.SeriesID {
		return false
	}

	if a.EpisodeID != nil && a.SeasonNum != nil && b.SeasonNum != nil && *a.SeasonNum != *b.SeasonNum {
		return false
	}

	return true
}

// enrichSeasonPackMatch extracts season/episode numbers from the filename and looks up
// the episode directly using the known SeriesID from the queue mapping. This is used
// when matchFromParse fails (e.g., due to apostrophes in titles preventing SQL LIKE matches)
// but we already know the series from the download mapping.
func (s *Service) enrichSeasonPackMatch(ctx context.Context, path string, queueMatch *LibraryMatch) *LibraryMatch {
	filename := filepath.Base(path)
	name := strings.TrimSuffix(filename, filepath.Ext(filename))

	var season, episode int

	if matches := tvPattern1.FindStringSubmatch(name); len(matches) >= 4 {
		season, _ = strconv.Atoi(matches[2])
		episode, _ = strconv.Atoi(matches[3])
	} else if matches := tvPattern2.FindStringSubmatch(name); len(matches) >= 4 {
		season, _ = strconv.Atoi(matches[2])
		episode, _ = strconv.Atoi(matches[3])
	}

	if season == 0 || episode == 0 {
		return nil
	}

	episodes, err := s.tv.ListEpisodes(ctx, *queueMatch.SeriesID, &season)
	if err != nil {
		return nil
	}

	for _, ep := range episodes {
		if ep.EpisodeNumber == episode {
			queueMatch.EpisodeID = &ep.ID
			queueMatch.SeasonNum = &season
			if ep.EpisodeFile != nil {
				queueMatch.IsUpgrade = true
				queueMatch.ExistingFile = ep.EpisodeFile.Path
				queueMatch.ExistingFileID = &ep.EpisodeFile.ID
			}
			return queueMatch
		}
	}

	return nil
}

// MatchPreview provides information about a potential match for manual import.
type MatchPreview struct {
	Matches    []*LibraryMatch `json:"matches"`
	ParsedInfo *ParsedInfo     `json:"parsedInfo"`
	Warnings   []string        `json:"warnings,omitempty"`
}

// ParsedInfo contains parsed information from a filename.
type ParsedInfo struct {
	Title       string `json:"title,omitempty"`
	Year        int    `json:"year,omitempty"`
	Season      int    `json:"season,omitempty"`
	Episodes    []int  `json:"episodes,omitempty"`
	Quality     string `json:"quality,omitempty"`
	Source      string `json:"source,omitempty"`
	Codec       string `json:"codec,omitempty"`
	Group       string `json:"group,omitempty"`
	IsTV        bool   `json:"isTV"`
	IsMovie     bool   `json:"isMovie"`
	RawFilename string `json:"rawFilename"`
}

// GetMatchPreview returns potential matches for a file without importing.
func (s *Service) GetMatchPreview(ctx context.Context, path string) (*MatchPreview, error) {
	preview := &MatchPreview{
		Matches: make([]*LibraryMatch, 0),
	}

	filename := filepath.Base(path)
	parsed := s.parseFilename(filename)
	preview.ParsedInfo = parsed

	// Find potential matches
	if parsed.IsTV {
		if match := s.matchTVFromParse(ctx, strings.TrimSuffix(filename, filepath.Ext(filename))); match != nil {
			preview.Matches = append(preview.Matches, match)
		}
	}

	if parsed.IsMovie {
		if match := s.matchMovieFromParse(ctx, strings.TrimSuffix(filename, filepath.Ext(filename))); match != nil {
			preview.Matches = append(preview.Matches, match)
		}
	}

	// Add warnings
	if len(preview.Matches) == 0 {
		preview.Warnings = append(preview.Warnings, "No matches found in library")
	}

	return preview, nil
}

// parseFilename extracts information from a filename.
func (s *Service) parseFilename(filename string) *ParsedInfo {
	info := &ParsedInfo{
		RawFilename: filename,
	}

	name := strings.TrimSuffix(filename, filepath.Ext(filename))

	s.parseTVOrMovie(name, info)
	s.parseQuality(name, info)
	s.parseSource(name, info)
	s.parseCodec(name, info)
	s.parseReleaseGroup(name, info)

	return info
}

func (s *Service) parseTVOrMovie(name string, info *ParsedInfo) {
	if s.parseTVPattern(name, info) {
		return
	}

	moviePattern := regexp.MustCompile(`(?i)^(.+?)[.\s_-]*[(\[]?(\d{4})[)\]]?`)
	if matches := moviePattern.FindStringSubmatch(name); len(matches) >= 3 {
		info.IsMovie = true
		info.Title = cleanTitle(matches[1])
		info.Year, _ = strconv.Atoi(matches[2])
	}
}

func (s *Service) parseTVPattern(name string, info *ParsedInfo) bool {
	if matches := tvPattern1.FindStringSubmatch(name); len(matches) >= 4 {
		info.IsTV = true
		info.Title = cleanTitle(matches[1])
		info.Season, _ = strconv.Atoi(matches[2])
		info.Episodes = append(info.Episodes, mustAtoi(matches[3]))
		return true
	}
	if matches := tvPattern2.FindStringSubmatch(name); len(matches) >= 4 {
		info.IsTV = true
		info.Title = cleanTitle(matches[1])
		info.Season, _ = strconv.Atoi(matches[2])
		info.Episodes = append(info.Episodes, mustAtoi(matches[3]))
		return true
	}
	return false
}

func (s *Service) parseQuality(name string, info *ParsedInfo) {
	qualityPatterns := []struct {
		pattern *regexp.Regexp
		quality string
	}{
		{regexp.MustCompile(`(?i)2160p`), "2160p"},
		{regexp.MustCompile(`(?i)1080p`), "1080p"},
		{regexp.MustCompile(`(?i)720p`), "720p"},
		{regexp.MustCompile(`(?i)480p`), "480p"},
		{regexp.MustCompile(`(?i)4K|UHD`), "2160p"},
	}

	for _, qp := range qualityPatterns {
		if qp.pattern.MatchString(name) {
			info.Quality = qp.quality
			break
		}
	}
}

func (s *Service) parseSource(name string, info *ParsedInfo) {
	sourcePatterns := []struct {
		pattern *regexp.Regexp
		source  string
	}{
		{regexp.MustCompile(`(?i)BluRay|BDRip|BRRip`), "BluRay"},
		{regexp.MustCompile(`(?i)WEB-?Rip`), "WEBRip"},
		{regexp.MustCompile(`(?i)(WEB-?DL|\bWEB\b)`), "WEBDL"},
		{regexp.MustCompile(`(?i)HDTV`), "HDTV"},
		{regexp.MustCompile(`(?i)DVDRip`), "DVDRip"},
	}

	for _, sp := range sourcePatterns {
		if sp.pattern.MatchString(name) {
			info.Source = sp.source
			break
		}
	}
}

func (s *Service) parseCodec(name string, info *ParsedInfo) {
	codecPatterns := []struct {
		pattern *regexp.Regexp
		codec   string
	}{
		{regexp.MustCompile(`(?i)x265|HEVC|h\.?265`), "x265"},
		{regexp.MustCompile(`(?i)x264|h\.?264|AVC`), "x264"},
		{regexp.MustCompile(`(?i)XviD`), "XviD"},
		{regexp.MustCompile(`(?i)AV1`), "AV1"},
	}

	for _, cp := range codecPatterns {
		if cp.pattern.MatchString(name) {
			info.Codec = cp.codec
			break
		}
	}
}

func (s *Service) parseReleaseGroup(name string, info *ParsedInfo) {
	groupPattern := regexp.MustCompile(`-([A-Za-z0-9]+)$`)
	if matches := groupPattern.FindStringSubmatch(name); len(matches) >= 2 {
		info.Group = matches[1]
	}
}

func mustAtoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}
