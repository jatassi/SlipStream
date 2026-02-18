package arrimport

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type apiReader struct {
	client     *http.Client
	baseURL    string
	apiKey     string
	sourceType SourceType
}

func newAPIReader(cfg ConnectionConfig) *apiReader {
	baseURL := strings.TrimRight(cfg.URL, "/")
	return &apiReader{
		client:     &http.Client{Timeout: 60 * time.Second},
		baseURL:    baseURL,
		apiKey:     cfg.APIKey,
		sourceType: cfg.SourceType,
	}
}

func (r *apiReader) doRequest(ctx context.Context, path string) ([]byte, error) {
	url := r.baseURL + path
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("X-Api-Key", r.apiKey)

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	return io.ReadAll(resp.Body)
}

func (r *apiReader) Validate(ctx context.Context) error {
	data, err := r.doRequest(ctx, "/api/v3/system/status")
	if err != nil {
		return fmt.Errorf("failed to validate connection: %w", err)
	}

	var status struct {
		AppName string `json:"appName"`
	}
	if err := json.Unmarshal(data, &status); err != nil {
		return fmt.Errorf("failed to parse status response: %w", err)
	}

	expectedApp := "Radarr"
	if r.sourceType == SourceTypeSonarr {
		expectedApp = "Sonarr"
	}

	if !strings.EqualFold(status.AppName, expectedApp) {
		return fmt.Errorf("expected %s but connected to %s", expectedApp, status.AppName)
	}
	return nil
}

func (r *apiReader) Close() error {
	return nil
}

func (r *apiReader) ReadRootFolders(ctx context.Context) ([]SourceRootFolder, error) {
	data, err := r.doRequest(ctx, "/api/v3/rootfolder")
	if err != nil {
		return nil, err
	}

	var apiFolders []struct {
		ID   int64  `json:"id"`
		Path string `json:"path"`
	}
	if err := json.Unmarshal(data, &apiFolders); err != nil {
		return nil, fmt.Errorf("failed to parse root folders: %w", err)
	}

	folders := make([]SourceRootFolder, len(apiFolders))
	for i, f := range apiFolders {
		folders[i] = SourceRootFolder{ID: f.ID, Path: f.Path}
	}
	return folders, nil
}

func (r *apiReader) ReadQualityProfiles(ctx context.Context) ([]SourceQualityProfile, error) {
	data, err := r.doRequest(ctx, "/api/v3/qualityprofile")
	if err != nil {
		return nil, err
	}

	var apiProfiles []struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal(data, &apiProfiles); err != nil {
		return nil, fmt.Errorf("failed to parse quality profiles: %w", err)
	}

	inUse := r.profileIDsInUse(ctx)

	profiles := make([]SourceQualityProfile, len(apiProfiles))
	for i, p := range apiProfiles {
		profiles[i] = SourceQualityProfile{ID: p.ID, Name: p.Name, InUse: inUse[p.ID]}
	}
	return profiles, nil
}

// profileIDsInUse fetches media items and returns which quality profile IDs are referenced.
func (r *apiReader) profileIDsInUse(ctx context.Context) map[int64]bool {
	ids := make(map[int64]bool)

	endpoint := "/api/v3/movie"
	if r.sourceType == SourceTypeSonarr {
		endpoint = "/api/v3/series"
	}

	data, err := r.doRequest(ctx, endpoint)
	if err != nil {
		return ids
	}

	var items []struct {
		QualityProfileID int64 `json:"qualityProfileId"`
	}
	if err := json.Unmarshal(data, &items); err != nil {
		return ids
	}

	for _, item := range items {
		ids[item.QualityProfileID] = true
	}
	return ids
}

type apiMovie struct {
	ID               int64         `json:"id"`
	Title            string        `json:"title"`
	SortTitle        string        `json:"sortTitle"`
	Year             int           `json:"year"`
	TmdbID           int           `json:"tmdbId"`
	ImdbID           string        `json:"imdbId"`
	Overview         string        `json:"overview"`
	Runtime          int           `json:"runtime"`
	Path             string        `json:"path"`
	RootFolderPath   string        `json:"rootFolderPath"`
	QualityProfileID int64         `json:"qualityProfileId"`
	Monitored        bool          `json:"monitored"`
	Status           string        `json:"status"`
	Studio           string        `json:"studio"`
	Certification    string        `json:"certification"`
	InCinemas        string        `json:"inCinemas"`
	PhysicalRelease  string        `json:"physicalRelease"`
	DigitalRelease   string        `json:"digitalRelease"`
	Added            string        `json:"added"`
	HasFile          bool          `json:"hasFile"`
	Images           []sourceImage `json:"images"`
	MovieFile        *struct {
		ID               int64  `json:"id"`
		Path             string `json:"path"`
		Size             int64  `json:"size"`
		OriginalFilePath string `json:"originalFilePath"`
		DateAdded        string `json:"dateAdded"`
		Quality          struct {
			Quality struct {
				ID   int    `json:"id"`
				Name string `json:"name"`
			} `json:"quality"`
		} `json:"quality"`
		MediaInfo struct {
			VideoCodec        string  `json:"videoCodec"`
			AudioCodec        string  `json:"audioCodec"`
			Resolution        string  `json:"resolution"`
			AudioChannels     float64 `json:"audioChannels"`
			VideoDynamicRange string  `json:"videoDynamicRange"`
		} `json:"mediaInfo"`
	} `json:"movieFile"`
}

func (r *apiReader) ReadMovies(ctx context.Context) ([]SourceMovie, error) {
	if r.sourceType != SourceTypeRadarr {
		return []SourceMovie{}, nil
	}

	rootFolders, err := r.ReadRootFolders(ctx)
	if err != nil {
		return nil, err
	}

	data, err := r.doRequest(ctx, "/api/v3/movie")
	if err != nil {
		return nil, err
	}

	var apiMovies []apiMovie
	if err := json.Unmarshal(data, &apiMovies); err != nil {
		return nil, fmt.Errorf("failed to parse movies: %w", err)
	}

	var movies []SourceMovie
	for i := range apiMovies {
		if apiMovies[i].Status == "deleted" {
			continue
		}
		movies = append(movies, convertAPIMovie(&apiMovies[i], rootFolders))
	}
	return movies, nil
}

func convertAPIMovie(am *apiMovie, rootFolders []SourceRootFolder) SourceMovie {
	m := SourceMovie{
		ID:               am.ID,
		Title:            am.Title,
		SortTitle:        am.SortTitle,
		Year:             am.Year,
		TmdbID:           am.TmdbID,
		ImdbID:           am.ImdbID,
		Overview:         am.Overview,
		Runtime:          am.Runtime,
		Path:             am.Path,
		RootFolderPath:   am.RootFolderPath,
		QualityProfileID: am.QualityProfileID,
		Monitored:        am.Monitored,
		Status:           am.Status,
		Studio:           am.Studio,
		Certification:    am.Certification,
		InCinemas:        parseDateTime(am.InCinemas),
		PhysicalRelease:  parseDateTime(am.PhysicalRelease),
		DigitalRelease:   parseDateTime(am.DigitalRelease),
		Added:            parseDateTime(am.Added),
		HasFile:          am.HasFile,
		PosterURL:        apiPosterURL(am.Images),
	}

	if m.RootFolderPath == "" {
		m.RootFolderPath = deriveRootFolderPath(m.Path, rootFolders)
	}

	if am.MovieFile != nil {
		m.File = &SourceMovieFile{
			ID:               am.MovieFile.ID,
			Path:             am.MovieFile.Path,
			Size:             am.MovieFile.Size,
			QualityID:        am.MovieFile.Quality.Quality.ID,
			QualityName:      am.MovieFile.Quality.Quality.Name,
			VideoCodec:       am.MovieFile.MediaInfo.VideoCodec,
			AudioCodec:       am.MovieFile.MediaInfo.AudioCodec,
			Resolution:       am.MovieFile.MediaInfo.Resolution,
			AudioChannels:    formatAudioChannels(am.MovieFile.MediaInfo.AudioChannels),
			DynamicRange:     am.MovieFile.MediaInfo.VideoDynamicRange,
			OriginalFilePath: am.MovieFile.OriginalFilePath,
			DateAdded:        parseDateTime(am.MovieFile.DateAdded),
		}
	}

	return m
}

type apiSeriesItem struct {
	ID               int64         `json:"id"`
	Title            string        `json:"title"`
	SortTitle        string        `json:"sortTitle"`
	Year             int           `json:"year"`
	TvdbID           int           `json:"tvdbId"`
	TmdbID           int           `json:"tmdbId"`
	ImdbID           string        `json:"imdbId"`
	Overview         string        `json:"overview"`
	Runtime          int           `json:"runtime"`
	Path             string        `json:"path"`
	RootFolderPath   string        `json:"rootFolderPath"`
	QualityProfileID int64         `json:"qualityProfileId"`
	Monitored        bool          `json:"monitored"`
	SeasonFolder     bool          `json:"seasonFolder"`
	Status           string        `json:"status"`
	Network          string        `json:"network"`
	SeriesType       string        `json:"seriesType"`
	Certification    string        `json:"certification"`
	Added            string        `json:"added"`
	Images           []sourceImage `json:"images"`
	Seasons          []struct {
		SeasonNumber int  `json:"seasonNumber"`
		Monitored    bool `json:"monitored"`
	} `json:"seasons"`
}

func (r *apiReader) ReadSeries(ctx context.Context) ([]SourceSeries, error) {
	if r.sourceType != SourceTypeSonarr {
		return []SourceSeries{}, nil
	}

	rootFolders, err := r.ReadRootFolders(ctx)
	if err != nil {
		return nil, err
	}

	data, err := r.doRequest(ctx, "/api/v3/series")
	if err != nil {
		return nil, err
	}

	var apiSeries []apiSeriesItem
	if err := json.Unmarshal(data, &apiSeries); err != nil {
		return nil, fmt.Errorf("failed to parse series: %w", err)
	}

	var seriesList []SourceSeries
	for i := range apiSeries {
		if apiSeries[i].Status == "deleted" {
			continue
		}
		seriesList = append(seriesList, convertAPISeries(&apiSeries[i], rootFolders))
	}
	return seriesList, nil
}

func convertAPISeries(as *apiSeriesItem, rootFolders []SourceRootFolder) SourceSeries {
	s := SourceSeries{
		ID:               as.ID,
		Title:            as.Title,
		SortTitle:        as.SortTitle,
		Year:             as.Year,
		TvdbID:           as.TvdbID,
		TmdbID:           as.TmdbID,
		ImdbID:           as.ImdbID,
		Overview:         as.Overview,
		Runtime:          as.Runtime,
		Path:             as.Path,
		RootFolderPath:   as.RootFolderPath,
		QualityProfileID: as.QualityProfileID,
		Monitored:        as.Monitored,
		SeasonFolder:     as.SeasonFolder,
		Status:           as.Status,
		Network:          as.Network,
		SeriesType:       mapSonarrSeriesType(as.SeriesType),
		Certification:    as.Certification,
		Added:            parseDateTime(as.Added),
		PosterURL:        apiPosterURL(as.Images),
	}

	if s.RootFolderPath == "" {
		s.RootFolderPath = deriveRootFolderPath(s.Path, rootFolders)
	}

	for _, season := range as.Seasons {
		s.Seasons = append(s.Seasons, SourceSeason{
			SeasonNumber: season.SeasonNumber,
			Monitored:    season.Monitored,
		})
	}

	return s
}

func (r *apiReader) ReadEpisodes(ctx context.Context, seriesID int64) ([]SourceEpisode, error) {
	if r.sourceType != SourceTypeSonarr {
		return []SourceEpisode{}, nil
	}

	data, err := r.doRequest(ctx, "/api/v3/episode?seriesId="+strconv.FormatInt(seriesID, 10))
	if err != nil {
		return nil, err
	}

	var apiEpisodes []struct {
		ID            int64  `json:"id"`
		SeriesID      int64  `json:"seriesId"`
		SeasonNumber  int    `json:"seasonNumber"`
		EpisodeNumber int    `json:"episodeNumber"`
		Title         string `json:"title"`
		Overview      string `json:"overview"`
		AirDateUtc    string `json:"airDateUtc"`
		Monitored     bool   `json:"monitored"`
		EpisodeFileID int64  `json:"episodeFileId"`
		HasFile       bool   `json:"hasFile"`
	}
	if err := json.Unmarshal(data, &apiEpisodes); err != nil {
		return nil, fmt.Errorf("failed to parse episodes: %w", err)
	}

	episodes := make([]SourceEpisode, len(apiEpisodes))
	for i, ae := range apiEpisodes {
		episodes[i] = SourceEpisode{
			ID:            ae.ID,
			SeriesID:      ae.SeriesID,
			SeasonNumber:  ae.SeasonNumber,
			EpisodeNumber: ae.EpisodeNumber,
			Title:         ae.Title,
			Overview:      ae.Overview,
			AirDateUtc:    ae.AirDateUtc,
			Monitored:     ae.Monitored,
			EpisodeFileID: ae.EpisodeFileID,
			HasFile:       ae.HasFile,
		}
	}
	return episodes, nil
}

type apiEpisodeFile struct {
	ID               int64  `json:"id"`
	SeriesID         int64  `json:"seriesId"`
	SeasonNumber     int    `json:"seasonNumber"`
	Path             string `json:"path"`
	RelativePath     string `json:"relativePath"`
	Size             int64  `json:"size"`
	OriginalFilePath string `json:"originalFilePath"`
	DateAdded        string `json:"dateAdded"`
	Quality          struct {
		Quality struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		} `json:"quality"`
	} `json:"quality"`
	MediaInfo struct {
		VideoCodec            string  `json:"videoCodec"`
		AudioCodec            string  `json:"audioCodec"`
		Resolution            string  `json:"resolution"`
		AudioChannels         float64 `json:"audioChannels"`
		VideoDynamicRangeType string  `json:"videoDynamicRangeType"`
	} `json:"mediaInfo"`
}

func (r *apiReader) ReadEpisodeFiles(ctx context.Context, seriesID int64) ([]SourceEpisodeFile, error) {
	if r.sourceType != SourceTypeSonarr {
		return []SourceEpisodeFile{}, nil
	}

	data, err := r.doRequest(ctx, "/api/v3/episodefile?seriesId="+strconv.FormatInt(seriesID, 10))
	if err != nil {
		return nil, err
	}

	var apiFiles []apiEpisodeFile
	if err := json.Unmarshal(data, &apiFiles); err != nil {
		return nil, fmt.Errorf("failed to parse episode files: %w", err)
	}

	files := make([]SourceEpisodeFile, len(apiFiles))
	for i := range apiFiles {
		relPath := apiFiles[i].RelativePath
		if relPath == "" {
			relPath = apiFiles[i].Path
		}
		files[i] = SourceEpisodeFile{
			ID:               apiFiles[i].ID,
			SeriesID:         apiFiles[i].SeriesID,
			SeasonNumber:     apiFiles[i].SeasonNumber,
			RelativePath:     relPath,
			Size:             apiFiles[i].Size,
			QualityID:        apiFiles[i].Quality.Quality.ID,
			QualityName:      apiFiles[i].Quality.Quality.Name,
			VideoCodec:       apiFiles[i].MediaInfo.VideoCodec,
			AudioCodec:       apiFiles[i].MediaInfo.AudioCodec,
			Resolution:       apiFiles[i].MediaInfo.Resolution,
			AudioChannels:    formatAudioChannels(apiFiles[i].MediaInfo.AudioChannels),
			DynamicRange:     apiFiles[i].MediaInfo.VideoDynamicRangeType,
			OriginalFilePath: apiFiles[i].OriginalFilePath,
			DateAdded:        parseDateTime(apiFiles[i].DateAdded),
		}
	}
	return files, nil
}

func apiPosterURL(images []sourceImage) string {
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

func formatAudioChannels(channels float64) string {
	if channels == 0 {
		return ""
	}
	return strconv.FormatFloat(channels, 'f', -1, 64)
}

// apiField represents a single field in Sonarr/Radarr's provider settings array.
type apiField struct {
	Name  string `json:"name"`
	Value any    `json:"value"`
}

func fieldsToJSON(fields []apiField) json.RawMessage {
	m := make(map[string]any)
	for _, f := range fields {
		if f.Value != nil {
			m[f.Name] = f.Value
		}
	}
	b, _ := json.Marshal(m)
	return b
}

func (r *apiReader) ReadDownloadClients(ctx context.Context) ([]SourceDownloadClient, error) {
	body, err := r.doRequest(ctx, "/api/v3/downloadclient")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch download clients: %w", err)
	}

	var resp []struct {
		ID                       int64      `json:"id"`
		Name                     string     `json:"name"`
		Implementation           string     `json:"implementation"`
		Fields                   []apiField `json:"fields"`
		Enable                   bool       `json:"enable"` // G17: not "enabled"
		Priority                 int        `json:"priority"`
		RemoveCompletedDownloads bool       `json:"removeCompletedDownloads"`
		RemoveFailedDownloads    bool       `json:"removeFailedDownloads"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse download clients: %w", err)
	}

	clients := make([]SourceDownloadClient, 0, len(resp))
	for _, item := range resp {
		clients = append(clients, SourceDownloadClient{
			ID:                       item.ID,
			Name:                     item.Name,
			Implementation:           item.Implementation,
			Settings:                 fieldsToJSON(item.Fields),
			Enabled:                  item.Enable,
			Priority:                 item.Priority,
			RemoveCompletedDownloads: item.RemoveCompletedDownloads,
			RemoveFailedDownloads:    item.RemoveFailedDownloads,
		})
	}
	return clients, nil
}

func (r *apiReader) ReadIndexers(ctx context.Context) ([]SourceIndexer, error) {
	body, err := r.doRequest(ctx, "/api/v3/indexer")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch indexers: %w", err)
	}

	var resp []struct {
		ID                      int64      `json:"id"`
		Name                    string     `json:"name"`
		Implementation          string     `json:"implementation"`
		Fields                  []apiField `json:"fields"`
		EnableRss               bool       `json:"enableRss"`
		EnableAutomaticSearch   bool       `json:"enableAutomaticSearch"`
		EnableInteractiveSearch bool       `json:"enableInteractiveSearch"`
		Priority                int        `json:"priority"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse indexers: %w", err)
	}

	indexers := make([]SourceIndexer, 0, len(resp))
	for _, item := range resp {
		indexers = append(indexers, SourceIndexer{
			ID:                      item.ID,
			Name:                    item.Name,
			Implementation:          item.Implementation,
			Settings:                fieldsToJSON(item.Fields),
			EnableRss:               item.EnableRss,
			EnableAutomaticSearch:   item.EnableAutomaticSearch,
			EnableInteractiveSearch: item.EnableInteractiveSearch,
			Priority:                item.Priority,
		})
	}
	return indexers, nil
}

func (r *apiReader) ReadNotifications(ctx context.Context) ([]SourceNotification, error) {
	body, err := r.doRequest(ctx, "/api/v3/notification")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch notifications: %w", err)
	}

	// Use SourceNotification directly — json.Unmarshal fills matching fields, leaves others as zero.
	// The API returns event fields in camelCase matching the struct tags.
	// Sonarr: onSeriesAdd/onSeriesDelete will populate, Radarr: onMovieAdded/onMovieDelete will populate.
	var resp []struct {
		SourceNotification
		Fields []apiField `json:"fields"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse notifications: %w", err)
	}

	notifications := make([]SourceNotification, 0, len(resp))
	for _, item := range resp {
		n := item.SourceNotification
		n.Settings = fieldsToJSON(item.Fields)
		notifications = append(notifications, n)
	}
	return notifications, nil
}

func (r *apiReader) ReadQualityProfilesFull(ctx context.Context) ([]SourceQualityProfileFull, error) {
	body, err := r.doRequest(ctx, "/api/v3/qualityprofile")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch quality profiles: %w", err)
	}

	var profiles []SourceQualityProfileFull
	if err := json.Unmarshal(body, &profiles); err != nil {
		return nil, fmt.Errorf("failed to parse quality profiles: %w", err)
	}
	return profiles, nil
}

func (r *apiReader) ReadNamingConfig(ctx context.Context) (*SourceNamingConfig, error) {
	body, err := r.doRequest(ctx, "/api/v3/config/naming")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch naming config: %w", err)
	}

	var nc SourceNamingConfig
	if err := json.Unmarshal(body, &nc); err != nil {
		return nil, fmt.Errorf("failed to parse naming config: %w", err)
	}
	return &nc, nil
}
