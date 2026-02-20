package arrimport

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/slipstream/slipstream/internal/pathutil"
	_ "modernc.org/sqlite" // SQLite driver
)

type sqliteReader struct {
	db           *sql.DB
	sourceType   SourceType
	qualityNames map[int]string
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

	qualityNames := r.getQualityNames(ctx)

	rows, err := r.db.QueryContext(ctx, `
		SELECT m.Id, m.Path, m.QualityProfileId, m.Monitored, m.Added, m.MovieFileId,
		       mm.Title, mm.SortTitle, mm.Year, mm.TmdbId, mm.ImdbId, mm.Overview,
		       mm.Runtime, mm.Status, mm.Studio, mm.Certification,
		       mm.InCinemas, mm.PhysicalRelease, mm.DigitalRelease, mm.Images
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
		var imagesJSON sql.NullString

		if err := rows.Scan(
			&m.ID, &m.Path, &m.QualityProfileID, &m.Monitored, &addedStr, &movieFileID,
			&m.Title, &m.SortTitle, &m.Year, &m.TmdbID, &imdbID, &overview,
			&m.Runtime, &status, &studio, &certification,
			&inCinemasStr, &physicalReleaseStr, &digitalReleaseStr, &imagesJSON,
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
		m.PosterURL = extractPosterURL(imagesJSON.String)
		m.HasFile = movieFileID > 0

		if m.HasFile {
			file, err := r.readMovieFile(ctx, movieFileID, m.Path, qualityNames)
			if err == nil {
				m.File = file
			}
		}

		movies = append(movies, m)
	}
	return movies, rows.Err()
}

func (r *sqliteReader) readMovieFile(ctx context.Context, fileID int64, moviePath string, qualityNames map[int]string) (*SourceMovieFile, error) {
	var f SourceMovieFile
	var qualityJSON, mediaInfoJSON sql.NullString
	var originalFilePath sql.NullString
	var dateAddedStr string
	var relativePath string

	err := r.db.QueryRowContext(ctx, `
		SELECT Id, RelativePath, Size, Quality, MediaInfo, OriginalFilePath, DateAdded
		FROM MovieFiles WHERE Id = ?
	`, fileID).Scan(&f.ID, &relativePath, &f.Size, &qualityJSON, &mediaInfoJSON, &originalFilePath, &dateAddedStr)
	if err != nil {
		return nil, err
	}

	f.Path = resolveFilePath(moviePath, relativePath)
	f.OriginalFilePath = originalFilePath.String
	f.DateAdded = parseDateTime(dateAddedStr)
	parseQualityJSON(qualityJSON.String, &f.QualityID, &f.QualityName, qualityNames)
	parseMediaInfoJSON(mediaInfoJSON.String, &f.VideoCodec, &f.AudioCodec, &f.Resolution, &f.AudioChannels, &f.DynamicRange)

	return &f, nil
}

// sqliteSeriesRow holds raw row data scanned from the Series table.
type sqliteSeriesRow struct {
	s             SourceSeries
	status        int
	addedStr      string
	seasonsJSON   sql.NullString
	imagesJSON    sql.NullString
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

const seriesTypeStandard = "standard"

func mapSonarrSeriesType(raw string) string {
	switch raw {
	case "0", seriesTypeStandard:
		return seriesTypeStandard
	case "1", "daily":
		return "daily"
	case "2", "anime":
		return "anime"
	default:
		return seriesTypeStandard
	}
}

func (row *sqliteSeriesRow) toSourceSeries(rootFolders []SourceRootFolder) SourceSeries {
	row.s.Status = mapSeriesStatus(row.status)
	row.s.TmdbID = int(row.tmdbID.Int64)
	row.s.ImdbID = row.imdbID.String
	row.s.Overview = row.overview.String
	row.s.Network = row.network.String
	row.s.SeriesType = mapSonarrSeriesType(row.seriesType.String)
	row.s.Certification = row.certification.String
	row.s.Added = parseDateTime(row.addedStr)
	row.s.RootFolderPath = deriveRootFolderPath(row.s.Path, rootFolders)
	row.s.PosterURL = extractPosterURL(row.imagesJSON.String)

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
		       Status, Network, SeriesType, Certification, Added, Seasons, Images
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
			&row.imagesJSON,
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

	qualityNames := r.getQualityNames(ctx)

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
		parseQualityJSON(qualityJSON.String, &f.QualityID, &f.QualityName, qualityNames)
		parseMediaInfoJSON(mediaInfoJSON.String, &f.VideoCodec, &f.AudioCodec, &f.Resolution, &f.AudioChannels, &f.DynamicRange)

		files = append(files, f)
	}
	return files, rows.Err()
}

// sourceImage represents an image entry from Radarr/Sonarr.
type sourceImage struct {
	CoverType string `json:"coverType"`
	URL       string `json:"url"`
	RemoteURL string `json:"remoteUrl"`
}

// extractPosterURL finds the poster image URL from a Radarr/Sonarr images array.
func extractPosterURL(imagesJSON string) string {
	if imagesJSON == "" {
		return ""
	}
	var images []sourceImage
	if err := json.Unmarshal([]byte(imagesJSON), &images); err != nil {
		return ""
	}
	for _, img := range images {
		if img.CoverType == "poster" {
			if img.RemoteURL != "" {
				return img.RemoteURL
			}
			return img.URL
		}
	}
	return ""
}

// getQualityNames returns a cached quality definitions lookup map.
func (r *sqliteReader) getQualityNames(ctx context.Context) map[int]string {
	if r.qualityNames != nil {
		return r.qualityNames
	}
	m := make(map[int]string)
	rows, err := r.db.QueryContext(ctx, "SELECT Quality, Title FROM QualityDefinitions")
	if err != nil {
		return m
	}
	defer rows.Close()
	for rows.Next() {
		var id int
		var title string
		if err := rows.Scan(&id, &title); err == nil {
			m[id] = title
		}
	}
	r.qualityNames = m
	return m
}

// parseQualityJSON extracts quality ID and name from the JSON quality column.
// Handles three formats:
//   - Nested object: {"quality":{"id":7,"name":"Bluray-1080p",...},...}
//   - Integer ID:    {"quality":31,...} (Radarr v5+, name resolved via qualityNames map)
//   - Flat object:   {"id":7,"name":"Bluray-1080p"}
func parseQualityJSON(jsonStr string, qualityID *int, qualityName *string, qualityNames map[int]string) {
	if jsonStr == "" {
		return
	}

	var wrapper struct {
		Quality json.RawMessage `json:"quality"`
	}
	if err := json.Unmarshal([]byte(jsonStr), &wrapper); err == nil && len(wrapper.Quality) > 0 {
		id, name := parseQualityField(wrapper.Quality, qualityNames)
		*qualityID = id
		*qualityName = name
		return
	}

	var flat struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal([]byte(jsonStr), &flat); err == nil && flat.Name != "" {
		*qualityID = flat.ID
		*qualityName = flat.Name
	}
}

// parseQualityField parses the "quality" field which can be an object or an integer.
func parseQualityField(raw json.RawMessage, qualityNames map[int]string) (qualityID int, qualityName string) {
	var obj struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal(raw, &obj); err == nil && obj.Name != "" {
		return obj.ID, obj.Name
	}

	var intID int
	if err := json.Unmarshal(raw, &intID); err == nil && intID > 0 && qualityNames != nil {
		return intID, qualityNames[intID]
	}

	return 0, ""
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

// resolveFilePath constructs a full file path from a parent directory and a relative path.
// Handles Windows-origin paths where the relative path may use backslash separators.
// Always returns forward-slash normalized paths.
func resolveFilePath(parentPath, relativePath string) string {
	if strings.HasPrefix(relativePath, "/") || (len(relativePath) >= 2 && relativePath[1] == ':') {
		return pathutil.NormalizePath(relativePath)
	}
	sep := "/"
	if strings.Contains(parentPath, "\\") {
		sep = "\\"
	}
	return pathutil.NormalizePath(strings.TrimRight(parentPath, "/\\") + sep + relativePath)
}

// deriveRootFolderPath finds the root folder path that is a prefix of the given path.
// Handles both Unix (/) and Windows (\) path separators since source databases
// may originate from either OS.
func deriveRootFolderPath(mediaPath string, rootFolders []SourceRootFolder) string {
	for _, rf := range rootFolders {
		rfPath := rf.Path
		if !strings.HasSuffix(rfPath, "/") && !strings.HasSuffix(rfPath, "\\") {
			rfPath += "/"
		}
		if strings.HasPrefix(mediaPath, rfPath) {
			return rf.Path
		}
	}
	return ""
}

func (r *sqliteReader) ReadDownloadClients(ctx context.Context) ([]SourceDownloadClient, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT Id, Name, Implementation, Settings, Enable, Priority, RemoveCompletedDownloads, RemoveFailedDownloads FROM DownloadClients`)
	if err != nil {
		return nil, fmt.Errorf("failed to query download clients: %w", err)
	}
	defer rows.Close()

	var clients []SourceDownloadClient
	for rows.Next() {
		var c SourceDownloadClient
		var enabled, removeCompleted, removeFailed int
		if err := rows.Scan(&c.ID, &c.Name, &c.Implementation, &c.Settings, &enabled, &c.Priority, &removeCompleted, &removeFailed); err != nil {
			return nil, fmt.Errorf("failed to scan download client: %w", err)
		}
		c.Enabled = enabled != 0
		c.RemoveCompletedDownloads = removeCompleted != 0
		c.RemoveFailedDownloads = removeFailed != 0
		clients = append(clients, c)
	}
	return clients, rows.Err()
}

func (r *sqliteReader) ReadIndexers(ctx context.Context) ([]SourceIndexer, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT Id, Name, Implementation, Settings, EnableRss, EnableAutomaticSearch, EnableInteractiveSearch, Priority FROM Indexers`)
	if err != nil {
		return nil, fmt.Errorf("failed to query indexers: %w", err)
	}
	defer rows.Close()

	var indexers []SourceIndexer
	for rows.Next() {
		var idx SourceIndexer
		var rss, autoSearch, interactive int
		if err := rows.Scan(&idx.ID, &idx.Name, &idx.Implementation, &idx.Settings, &rss, &autoSearch, &interactive, &idx.Priority); err != nil {
			return nil, fmt.Errorf("failed to scan indexer: %w", err)
		}
		idx.EnableRss = rss != 0
		idx.EnableAutomaticSearch = autoSearch != 0
		idx.EnableInteractiveSearch = interactive != 0
		indexers = append(indexers, idx)
	}
	return indexers, rows.Err()
}

func (r *sqliteReader) ReadNotifications(ctx context.Context) ([]SourceNotification, error) {
	// G7: notification columns differ between Sonarr and Radarr
	var query string
	switch r.sourceType {
	case SourceTypeSonarr:
		query = `SELECT Id, Name, Implementation, Settings, OnGrab, OnDownload, OnUpgrade,
			OnHealthIssue, IncludeHealthWarnings, OnHealthRestored, OnApplicationUpdate,
			OnSeriesAdd, OnSeriesDelete
			FROM Notifications`
	case SourceTypeRadarr:
		query = `SELECT Id, Name, Implementation, Settings, OnGrab, OnDownload, OnUpgrade,
			OnHealthIssue, IncludeHealthWarnings, OnHealthRestored, OnApplicationUpdate,
			OnMovieAdded, OnMovieDelete
			FROM Notifications`
	default:
		return nil, fmt.Errorf("unsupported source type: %s", r.sourceType)
	}

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query notifications: %w", err)
	}
	defer rows.Close()

	var notifications []SourceNotification
	for rows.Next() {
		var n SourceNotification
		var onGrab, onDownload, onUpgrade, onHealthIssue, includeHealthWarnings, onHealthRestored, onAppUpdate int
		var extra1, extra2 int

		if err := rows.Scan(&n.ID, &n.Name, &n.Implementation, &n.Settings,
			&onGrab, &onDownload, &onUpgrade,
			&onHealthIssue, &includeHealthWarnings, &onHealthRestored, &onAppUpdate,
			&extra1, &extra2,
		); err != nil {
			return nil, fmt.Errorf("failed to scan notification: %w", err)
		}

		n.OnGrab = onGrab != 0
		n.OnDownload = onDownload != 0
		n.OnUpgrade = onUpgrade != 0
		n.OnHealthIssue = onHealthIssue != 0
		n.IncludeHealthWarnings = includeHealthWarnings != 0
		n.OnHealthRestored = onHealthRestored != 0
		n.OnApplicationUpdate = onAppUpdate != 0

		switch r.sourceType {
		case SourceTypeSonarr:
			n.OnSeriesAdd = extra1 != 0
			n.OnSeriesDelete = extra2 != 0
		case SourceTypeRadarr:
			n.OnMovieAdded = extra1 != 0
			n.OnMovieDelete = extra2 != 0
		}

		notifications = append(notifications, n)
	}
	return notifications, rows.Err()
}

func (r *sqliteReader) ReadQualityProfilesFull(ctx context.Context) ([]SourceQualityProfileFull, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT Id, Name, Cutoff, UpgradeAllowed, Items FROM QualityProfiles`)
	if err != nil {
		return nil, fmt.Errorf("failed to query quality profiles: %w", err)
	}
	defer rows.Close()

	var profiles []SourceQualityProfileFull
	for rows.Next() {
		var p SourceQualityProfileFull
		var upgradeAllowed int
		if err := rows.Scan(&p.ID, &p.Name, &p.Cutoff, &upgradeAllowed, &p.Items); err != nil {
			return nil, fmt.Errorf("failed to scan quality profile: %w", err)
		}
		p.UpgradeAllowed = upgradeAllowed != 0
		profiles = append(profiles, p)
	}
	return profiles, rows.Err()
}

func (r *sqliteReader) ReadNamingConfig(ctx context.Context) (*SourceNamingConfig, error) {
	// G8: naming config columns differ between Sonarr and Radarr
	var nc SourceNamingConfig
	switch r.sourceType {
	case SourceTypeSonarr:
		var renameEpisodes, replaceIllegal int
		err := r.db.QueryRowContext(ctx, `SELECT RenameEpisodes, ReplaceIllegalCharacters, ColonReplacementFormat,
			MultiEpisodeStyle, StandardEpisodeFormat, DailyEpisodeFormat,
			AnimeEpisodeFormat, SeriesFolderFormat, SeasonFolderFormat, SpecialsFolderFormat
			FROM NamingConfig WHERE Id = 1`).Scan(
			&renameEpisodes, &replaceIllegal, &nc.ColonReplacementFormat,
			&nc.MultiEpisodeStyle, &nc.StandardEpisodeFormat, &nc.DailyEpisodeFormat,
			&nc.AnimeEpisodeFormat, &nc.SeriesFolderFormat, &nc.SeasonFolderFormat, &nc.SpecialsFolderFormat,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to query naming config: %w", err)
		}
		nc.RenameEpisodes = renameEpisodes != 0
		nc.ReplaceIllegalCharacters = replaceIllegal != 0
	case SourceTypeRadarr:
		var renameMovies, replaceIllegal int
		err := r.db.QueryRowContext(ctx, `SELECT RenameMovies, ReplaceIllegalCharacters, ColonReplacementFormat,
			StandardMovieFormat, MovieFolderFormat
			FROM NamingConfig WHERE Id = 1`).Scan(
			&renameMovies, &replaceIllegal, &nc.ColonReplacementFormat,
			&nc.StandardMovieFormat, &nc.MovieFolderFormat,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to query naming config: %w", err)
		}
		nc.RenameMovies = renameMovies != 0
		nc.ReplaceIllegalCharacters = replaceIllegal != 0
	default:
		return nil, fmt.Errorf("unsupported source type: %s", r.sourceType)
	}
	return &nc, nil
}
