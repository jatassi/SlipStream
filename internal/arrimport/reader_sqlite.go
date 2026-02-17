package arrimport

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	_ "modernc.org/sqlite" // SQLite driver
)

type sqliteReader struct {
	db         *sql.DB
	sourceType SourceType
}

func newSQLiteReader(cfg ConnectionConfig) (*sqliteReader, error) {
	db, err := sql.Open("sqlite", cfg.DBPath+"?mode=ro&_journal_mode=wal")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	return &sqliteReader{db: db, sourceType: cfg.SourceType}, nil
}

func (r *sqliteReader) Validate(ctx context.Context) error {
	switch r.sourceType {
	case SourceTypeRadarr:
		return r.validateTable(ctx, "Movies")
	case SourceTypeSonarr:
		return r.validateTable(ctx, "Series")
	default:
		return fmt.Errorf("unknown source type: %s", r.sourceType)
	}
}

func (r *sqliteReader) validateTable(ctx context.Context, tableName string) error {
	var name string
	err := r.db.QueryRowContext(ctx,
		"SELECT name FROM sqlite_master WHERE type='table' AND name=?", tableName).Scan(&name)
	if err != nil {
		return fmt.Errorf("table %q not found in database: %w", tableName, err)
	}
	return nil
}

func (r *sqliteReader) Close() error {
	return r.db.Close()
}

// ReadRootFolders reads root folders from the source database.
func (r *sqliteReader) ReadRootFolders(ctx context.Context) ([]SourceRootFolder, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT Id, Path FROM RootFolders")
	if err != nil {
		return nil, fmt.Errorf("failed to query root folders: %w", err)
	}
	defer rows.Close()

	var folders []SourceRootFolder
	for rows.Next() {
		var f SourceRootFolder
		if err := rows.Scan(&f.ID, &f.Path); err != nil {
			return nil, fmt.Errorf("failed to scan root folder: %w", err)
		}
		folders = append(folders, f)
	}
	return folders, rows.Err()
}

// ReadQualityProfiles reads quality profiles from the source database.
func (r *sqliteReader) ReadQualityProfiles(ctx context.Context) ([]SourceQualityProfile, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT Id, Name FROM QualityProfiles")
	if err != nil {
		return nil, fmt.Errorf("failed to query quality profiles: %w", err)
	}
	defer rows.Close()

	var profiles []SourceQualityProfile
	for rows.Next() {
		var p SourceQualityProfile
		if err := rows.Scan(&p.ID, &p.Name); err != nil {
			return nil, fmt.Errorf("failed to scan quality profile: %w", err)
		}
		profiles = append(profiles, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	inUse, err := r.profileIDsInUse(ctx)
	if err != nil {
		return profiles, nil
	}
	for i := range profiles {
		if inUse[profiles[i].ID] {
			profiles[i].InUse = true
		}
	}
	return profiles, nil
}

// profileIDsInUse returns the set of quality profile IDs referenced by media items.
func (r *sqliteReader) profileIDsInUse(ctx context.Context) (map[int64]bool, error) {
	table := "Movies"
	if r.sourceType == SourceTypeSonarr {
		table = "Series"
	}

	rows, err := r.db.QueryContext(ctx,
		"SELECT DISTINCT QualityProfileId FROM "+table) //nolint:gosec // table name is from a trusted constant
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := make(map[int64]bool)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids[id] = true
	}
	return ids, rows.Err()
}

// ReadMovies reads movies from a Radarr database.
func (r *sqliteReader) ReadMovies(ctx context.Context) ([]SourceMovie, error) {
	if r.sourceType != SourceTypeRadarr {
		return []SourceMovie{}, nil
	}

	rootFolders, err := r.ReadRootFolders(ctx)
	if err != nil {
		return nil, err
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT m.Id, m.Path, m.QualityProfileId, m.Monitored, m.Added, m.MovieFileId,
		       mm.Title, mm.SortTitle, mm.Year, mm.TmdbId, mm.ImdbId, mm.Overview,
		       mm.Runtime, mm.Status, mm.Studio, mm.Certification,
		       mm.InCinemas, mm.PhysicalRelease, mm.DigitalRelease
		FROM Movies m
		JOIN MovieMetadata mm ON m.MovieMetadataId = mm.Id
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query movies: %w", err)
	}
	defer rows.Close()

	var movies []SourceMovie
	for rows.Next() {
		var m SourceMovie
		var movieFileID int64
		var status int
		var addedStr string
		var inCinemasStr, physicalReleaseStr, digitalReleaseStr sql.NullString
		var imdbID sql.NullString
		var overview, studio, certification sql.NullString

		if err := rows.Scan(
			&m.ID, &m.Path, &m.QualityProfileID, &m.Monitored, &addedStr, &movieFileID,
			&m.Title, &m.SortTitle, &m.Year, &m.TmdbID, &imdbID, &overview,
			&m.Runtime, &status, &studio, &certification,
			&inCinemasStr, &physicalReleaseStr, &digitalReleaseStr,
		); err != nil {
			return nil, fmt.Errorf("failed to scan movie: %w", err)
		}

		// Skip deleted movies
		if status == -1 {
			continue
		}

		m.Status = fmt.Sprintf("%d", status)
		m.ImdbID = imdbID.String
		m.Overview = overview.String
		m.Studio = studio.String
		m.Certification = certification.String
		m.Added = parseDateTime(addedStr)
		m.InCinemas = parseDateTime(inCinemasStr.String)
		m.PhysicalRelease = parseDateTime(physicalReleaseStr.String)
		m.DigitalRelease = parseDateTime(digitalReleaseStr.String)
		m.RootFolderPath = deriveRootFolderPath(m.Path, rootFolders)
		m.HasFile = movieFileID > 0

		if m.HasFile {
			file, err := r.readMovieFile(ctx, movieFileID)
			if err == nil {
				m.File = file
			}
		}

		movies = append(movies, m)
	}
	return movies, rows.Err()
}

func (r *sqliteReader) readMovieFile(ctx context.Context, fileID int64) (*SourceMovieFile, error) {
	var f SourceMovieFile
	var qualityJSON, mediaInfoJSON sql.NullString
	var originalFilePath sql.NullString
	var dateAddedStr string

	err := r.db.QueryRowContext(ctx, `
		SELECT Id, Path, Size, Quality, MediaInfo, OriginalFilePath, DateAdded
		FROM MovieFiles WHERE Id = ?
	`, fileID).Scan(&f.ID, &f.Path, &f.Size, &qualityJSON, &mediaInfoJSON, &originalFilePath, &dateAddedStr)
	if err != nil {
		return nil, err
	}

	f.OriginalFilePath = originalFilePath.String
	f.DateAdded = parseDateTime(dateAddedStr)
	parseQualityJSON(qualityJSON.String, &f.QualityID, &f.QualityName)
	parseMediaInfoJSON(mediaInfoJSON.String, &f.VideoCodec, &f.AudioCodec, &f.Resolution, &f.AudioChannels, &f.DynamicRange)

	return &f, nil
}

// sqliteSeriesRow holds raw row data scanned from the Series table.
type sqliteSeriesRow struct {
	s             SourceSeries
	status        int
	addedStr      string
	seasonsJSON   sql.NullString
	imdbID        sql.NullString
	overview      sql.NullString
	network       sql.NullString
	seriesType    sql.NullString
	certification sql.NullString
	tmdbID        sql.NullInt64
}

func mapSeriesStatus(status int) string {
	switch status {
	case 0:
		return "continuing"
	case 1:
		return "ended"
	case 2:
		return "upcoming"
	default:
		return "continuing"
	}
}

func (row *sqliteSeriesRow) toSourceSeries(rootFolders []SourceRootFolder) SourceSeries {
	row.s.Status = mapSeriesStatus(row.status)
	row.s.TmdbID = int(row.tmdbID.Int64)
	row.s.ImdbID = row.imdbID.String
	row.s.Overview = row.overview.String
	row.s.Network = row.network.String
	row.s.SeriesType = row.seriesType.String
	row.s.Certification = row.certification.String
	row.s.Added = parseDateTime(row.addedStr)
	row.s.RootFolderPath = deriveRootFolderPath(row.s.Path, rootFolders)

	if row.seasonsJSON.Valid {
		var seasons []SourceSeason
		if err := json.Unmarshal([]byte(row.seasonsJSON.String), &seasons); err == nil {
			row.s.Seasons = seasons
		}
	}

	return row.s
}

// ReadSeries reads series from a Sonarr database.
func (r *sqliteReader) ReadSeries(ctx context.Context) ([]SourceSeries, error) {
	if r.sourceType != SourceTypeSonarr {
		return []SourceSeries{}, nil
	}

	rootFolders, err := r.ReadRootFolders(ctx)
	if err != nil {
		return nil, err
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT Id, Title, SortTitle, Year, TvdbId, TmdbId, ImdbId, Overview,
		       Runtime, Path, QualityProfileId, Monitored, SeasonFolder,
		       Status, Network, SeriesType, Certification, Added, Seasons
		FROM Series
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query series: %w", err)
	}
	defer rows.Close()

	var seriesList []SourceSeries
	for rows.Next() {
		var row sqliteSeriesRow
		if err := rows.Scan(
			&row.s.ID, &row.s.Title, &row.s.SortTitle, &row.s.Year, &row.s.TvdbID,
			&row.tmdbID, &row.imdbID, &row.overview,
			&row.s.Runtime, &row.s.Path, &row.s.QualityProfileID, &row.s.Monitored, &row.s.SeasonFolder,
			&row.status, &row.network, &row.seriesType, &row.certification, &row.addedStr, &row.seasonsJSON,
		); err != nil {
			return nil, fmt.Errorf("failed to scan series: %w", err)
		}

		if row.status == -1 {
			continue
		}

		seriesList = append(seriesList, row.toSourceSeries(rootFolders))
	}
	return seriesList, rows.Err()
}

// ReadEpisodes reads episodes for a series from a Sonarr database.
func (r *sqliteReader) ReadEpisodes(ctx context.Context, seriesID int64) ([]SourceEpisode, error) {
	if r.sourceType != SourceTypeSonarr {
		return []SourceEpisode{}, nil
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT Id, SeriesId, SeasonNumber, EpisodeNumber, Title, Overview,
		       AirDateUtc, Monitored, EpisodeFileId
		FROM Episodes WHERE SeriesId = ?
	`, seriesID)
	if err != nil {
		return nil, fmt.Errorf("failed to query episodes: %w", err)
	}
	defer rows.Close()

	var episodes []SourceEpisode
	for rows.Next() {
		var ep SourceEpisode
		var title, overview, airDateUtc sql.NullString
		var episodeFileID int64

		if err := rows.Scan(
			&ep.ID, &ep.SeriesID, &ep.SeasonNumber, &ep.EpisodeNumber, &title, &overview,
			&airDateUtc, &ep.Monitored, &episodeFileID,
		); err != nil {
			return nil, fmt.Errorf("failed to scan episode: %w", err)
		}

		ep.Title = title.String
		ep.Overview = overview.String
		ep.AirDateUtc = airDateUtc.String
		ep.EpisodeFileID = episodeFileID
		ep.HasFile = episodeFileID > 0

		episodes = append(episodes, ep)
	}
	return episodes, rows.Err()
}

// ReadEpisodeFiles reads episode files for a series from a Sonarr database.
func (r *sqliteReader) ReadEpisodeFiles(ctx context.Context, seriesID int64) ([]SourceEpisodeFile, error) {
	if r.sourceType != SourceTypeSonarr {
		return []SourceEpisodeFile{}, nil
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT Id, SeriesId, SeasonNumber, RelativePath, Size, Quality, MediaInfo,
		       OriginalFilePath, DateAdded
		FROM EpisodeFiles WHERE SeriesId = ?
	`, seriesID)
	if err != nil {
		return nil, fmt.Errorf("failed to query episode files: %w", err)
	}
	defer rows.Close()

	var files []SourceEpisodeFile
	for rows.Next() {
		var f SourceEpisodeFile
		var qualityJSON, mediaInfoJSON sql.NullString
		var originalFilePath sql.NullString
		var dateAddedStr string

		if err := rows.Scan(
			&f.ID, &f.SeriesID, &f.SeasonNumber, &f.RelativePath, &f.Size,
			&qualityJSON, &mediaInfoJSON, &originalFilePath, &dateAddedStr,
		); err != nil {
			return nil, fmt.Errorf("failed to scan episode file: %w", err)
		}

		f.OriginalFilePath = originalFilePath.String
		f.DateAdded = parseDateTime(dateAddedStr)
		parseQualityJSON(qualityJSON.String, &f.QualityID, &f.QualityName)
		parseMediaInfoJSON(mediaInfoJSON.String, &f.VideoCodec, &f.AudioCodec, &f.Resolution, &f.AudioChannels, &f.DynamicRange)

		files = append(files, f)
	}
	return files, rows.Err()
}

// parseQualityJSON extracts quality ID and name from the JSON quality column.
// Structure: {"quality":{"id":7,"name":"Bluray-1080p","source":"bluray","resolution":1080},...}
func parseQualityJSON(jsonStr string, qualityID *int, qualityName *string) {
	if jsonStr == "" {
		return
	}
	var wrapper struct {
		Quality struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		} `json:"quality"`
	}
	if err := json.Unmarshal([]byte(jsonStr), &wrapper); err == nil {
		*qualityID = wrapper.Quality.ID
		*qualityName = wrapper.Quality.Name
	}
}

// parseMediaInfoJSON extracts media info fields from the JSON column.
func parseMediaInfoJSON(jsonStr string, videoCodec, audioCodec, resolution, audioChannels, dynamicRange *string) {
	if jsonStr == "" {
		return
	}
	var info map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &info); err != nil {
		return
	}

	if v, ok := info["videoCodec"].(string); ok {
		*videoCodec = v
	}
	if v, ok := info["audioCodec"].(string); ok {
		*audioCodec = v
	}
	if v, ok := info["resolution"].(string); ok {
		*resolution = v
	}
	// audioChannels can be a float (5.1)
	if v, ok := info["audioChannels"].(float64); ok {
		*audioChannels = strconv.FormatFloat(v, 'f', -1, 64)
	}
	// Radarr uses videoDynamicRange, Sonarr uses videoDynamicRangeType
	if v, ok := info["videoDynamicRange"].(string); ok {
		*dynamicRange = v
	} else if v, ok := info["videoDynamicRangeType"].(string); ok {
		*dynamicRange = v
	}
}

// parseDateTime parses a date/time string in various formats.
func parseDateTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	s = strings.TrimSpace(s)

	formats := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

// deriveRootFolderPath finds the root folder path that is a prefix of the given path.
func deriveRootFolderPath(mediaPath string, rootFolders []SourceRootFolder) string {
	for _, rf := range rootFolders {
		rfPath := rf.Path
		if !strings.HasSuffix(rfPath, "/") {
			rfPath += "/"
		}
		if strings.HasPrefix(mediaPath, rfPath) {
			return rf.Path
		}
	}
	return ""
}
