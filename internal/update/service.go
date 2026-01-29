package update

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/config"
	"github.com/slipstream/slipstream/internal/database/sqlc"
	"github.com/slipstream/slipstream/internal/websocket"
)

const (
	githubAPIURL     = "https://api.github.com/repos/jatassi/SlipStream/releases/latest"
	settingAutoInstall = "update_auto_install"
)

type State string

const (
	StateIdle            State = "idle"
	StateChecking        State = "checking"
	StateUpToDate        State = "up-to-date"
	StateUpdateAvailable State = "update-available"
	StateError           State = "error"
	StateDownloading     State = "downloading"
	StateInstalling      State = "installing"
	StateRestarting      State = "restarting"
	StateComplete        State = "complete"
	StateFailed          State = "failed"
)

type ReleaseInfo struct {
	Version      string    `json:"version"`
	TagName      string    `json:"tagName"`
	ReleaseDate  string    `json:"releaseDate"`
	ReleaseNotes string    `json:"releaseNotes"`
	DownloadURL  string    `json:"downloadUrl"`
	AssetName    string    `json:"assetName"`
	AssetSize    int64     `json:"assetSize"`
	PublishedAt  time.Time `json:"publishedAt"`
}

type Status struct {
	State          State        `json:"state"`
	CurrentVersion string       `json:"currentVersion"`
	LatestRelease  *ReleaseInfo `json:"latestRelease,omitempty"`
	Progress       float64      `json:"progress"`
	DownloadedMB   float64      `json:"downloadedMB,omitempty"`
	TotalMB        float64      `json:"totalMB,omitempty"`
	Error          string       `json:"error,omitempty"`
	LastChecked    *time.Time   `json:"lastChecked,omitempty"`
}

type Settings struct {
	AutoInstall bool `json:"autoInstall"`
}

type githubRelease struct {
	TagName     string        `json:"tag_name"`
	Name        string        `json:"name"`
	Body        string        `json:"body"`
	Draft       bool          `json:"draft"`
	Prerelease  bool          `json:"prerelease"`
	PublishedAt time.Time     `json:"published_at"`
	Assets      []githubAsset `json:"assets"`
}

type githubAsset struct {
	Name               string `json:"name"`
	Size               int64  `json:"size"`
	BrowserDownloadURL string `json:"browser_download_url"`
	ContentType        string `json:"content_type"`
}

type Broadcaster interface {
	BroadcastUpdateStatus(status interface{})
}

type Service struct {
	db             *sql.DB
	logger         zerolog.Logger
	hub            Broadcaster
	httpClient     *http.Client
	downloadClient *http.Client
	restartChan    chan<- struct{}

	mu            sync.RWMutex
	status        Status
	cancelFunc    context.CancelFunc
	downloadPath  string
}

func NewService(db *sql.DB, logger zerolog.Logger, restartChan chan<- struct{}) *Service {
	return &Service{
		db:     db,
		logger: logger.With().Str("service", "update").Logger(),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		downloadClient: &http.Client{
			Timeout: 0, // No timeout for large file downloads; cancellation is handled via context
		},
		restartChan: restartChan,
		status: Status{
			State:          StateIdle,
			CurrentVersion: config.Version,
		},
	}
}

func (s *Service) SetBroadcaster(hub Broadcaster) {
	s.hub = hub
}

func (s *Service) GetStatus() Status {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.status
}

func (s *Service) GetSettings(ctx context.Context) (*Settings, error) {
	queries := sqlc.New(s.db)
	setting, err := queries.GetSetting(ctx, settingAutoInstall)
	if err != nil {
		if err == sql.ErrNoRows {
			return &Settings{AutoInstall: true}, nil
		}
		return nil, err
	}
	return &Settings{
		AutoInstall: setting.Value == "true" || setting.Value == "1",
	}, nil
}

func (s *Service) UpdateSettings(ctx context.Context, settings *Settings) error {
	queries := sqlc.New(s.db)
	value := "false"
	if settings.AutoInstall {
		value = "true"
	}
	_, err := queries.SetSetting(ctx, sqlc.SetSettingParams{
		Key:   settingAutoInstall,
		Value: value,
	})
	return err
}

func (s *Service) setState(state State, err error) {
	s.mu.Lock()
	s.status.State = state
	if err != nil {
		s.status.Error = err.Error()
	} else {
		s.status.Error = ""
	}
	status := s.status
	s.mu.Unlock()

	s.broadcast(&status)
}

func (s *Service) setProgress(progress float64, downloadedMB, totalMB float64) {
	s.mu.Lock()
	s.status.Progress = progress
	s.status.DownloadedMB = downloadedMB
	s.status.TotalMB = totalMB
	status := s.status
	s.mu.Unlock()

	s.broadcast(&status)
}

func (s *Service) broadcast(status *Status) {
	if s.hub != nil {
		s.hub.BroadcastUpdateStatus(status)
	}
}

func (s *Service) CheckForUpdate(ctx context.Context) (*ReleaseInfo, error) {
	s.setState(StateChecking, nil)

	release, err := s.fetchLatestRelease(ctx)
	if err != nil {
		s.setState(StateError, err)
		return nil, err
	}

	currentVersion := config.Version
	if currentVersion == "" || currentVersion == "dev" {
		s.logger.Debug().Msg("Running development build, skipping version comparison")
		s.mu.Lock()
		s.status.State = StateUpToDate
		s.status.LastChecked = ptr(time.Now())
		status := s.status
		s.mu.Unlock()
		s.broadcast(&status)
		return nil, nil
	}

	isNewer, err := IsNewerThan(release.TagName, currentVersion)
	if err != nil {
		s.logger.Warn().Err(err).Str("tagName", release.TagName).Str("currentVersion", currentVersion).Msg("Failed to compare versions")
		isNewer = release.TagName != currentVersion && release.TagName != "v"+currentVersion
	}

	now := time.Now()
	s.mu.Lock()
	s.status.LastChecked = &now
	if isNewer {
		releaseInfo := s.buildReleaseInfo(release)
		s.status.State = StateUpdateAvailable
		s.status.LatestRelease = releaseInfo
		status := s.status
		s.mu.Unlock()
		s.broadcast(&status)
		return releaseInfo, nil
	}

	s.status.State = StateUpToDate
	s.status.LatestRelease = nil
	status := s.status
	s.mu.Unlock()
	s.broadcast(&status)
	return nil, nil
}

func (s *Service) fetchLatestRelease(ctx context.Context) (*githubRelease, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, githubAPIURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "SlipStream/"+config.Version)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to decode release: %w", err)
	}

	if release.Draft || release.Prerelease {
		return nil, fmt.Errorf("latest release is draft or prerelease")
	}

	return &release, nil
}

func (s *Service) buildReleaseInfo(release *githubRelease) *ReleaseInfo {
	info := &ReleaseInfo{
		Version:      strings.TrimPrefix(release.TagName, "v"),
		TagName:      release.TagName,
		ReleaseDate:  release.PublishedAt.Format("2006-01-02"),
		ReleaseNotes: release.Body,
		PublishedAt:  release.PublishedAt,
	}

	asset := s.findPlatformAsset(release.Assets)
	if asset != nil {
		info.DownloadURL = asset.BrowserDownloadURL
		info.AssetName = asset.Name
		info.AssetSize = asset.Size
	}

	return info
}

func (s *Service) findPlatformAsset(assets []githubAsset) *githubAsset {
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	var patterns []string
	switch goos {
	case "windows":
		patterns = []string{
			fmt.Sprintf("_windows_%s_setup.exe", goarch),
			fmt.Sprintf("_windows_%s.zip", goarch),
		}
	case "darwin":
		patterns = []string{
			fmt.Sprintf("_darwin_%s.dmg", goarch),
		}
	case "linux":
		patterns = []string{
			fmt.Sprintf("_linux_%s.AppImage", goarch),
			fmt.Sprintf("_%s.deb", goarch),
			fmt.Sprintf("_%s.rpm", goarch),
		}
	}

	for _, pattern := range patterns {
		for i := range assets {
			if strings.Contains(assets[i].Name, pattern) {
				return &assets[i]
			}
		}
	}

	return nil
}

func (s *Service) DownloadAndInstall(ctx context.Context) error {
	s.logger.Info().Msg("DownloadAndInstall called")

	s.mu.RLock()
	release := s.status.LatestRelease
	s.mu.RUnlock()

	if release == nil || release.DownloadURL == "" {
		s.logger.Error().Msg("No update available to download")
		return fmt.Errorf("no update available to download")
	}

	s.logger.Info().
		Str("version", release.Version).
		Str("url", release.DownloadURL).
		Int64("sizeBytes", release.AssetSize).
		Msg("Release info validated")

	ctx, cancel := context.WithCancel(ctx)
	s.mu.Lock()
	s.cancelFunc = cancel
	s.mu.Unlock()

	s.logger.Info().Msg("Created cancellable context for download")

	defer func() {
		s.logger.Info().Msg("Cleaning up download context")
		s.mu.Lock()
		s.cancelFunc = nil
		s.mu.Unlock()
	}()

	s.logger.Info().Str("version", release.Version).Str("url", release.DownloadURL).Msg("Starting update download")

	downloadPath, err := s.downloadUpdate(ctx, release)
	if err != nil {
		s.logger.Error().Err(err).Bool("contextCanceled", ctx.Err() != nil).Msg("Download failed")
		s.setState(StateFailed, err)
		return err
	}

	s.logger.Info().Str("downloadPath", downloadPath).Msg("Download completed, proceeding to install")

	s.mu.Lock()
	s.downloadPath = downloadPath
	s.mu.Unlock()

	if err := s.backupDatabase(ctx); err != nil {
		s.logger.Warn().Err(err).Msg("Database backup failed, continuing with update")
	}

	if err := s.installUpdate(ctx, downloadPath); err != nil {
		s.setState(StateFailed, err)
		return err
	}

	return nil
}

func (s *Service) downloadUpdate(ctx context.Context, release *ReleaseInfo) (string, error) {
	s.setState(StateDownloading, nil)
	s.logger.Info().Str("url", release.DownloadURL).Msg("Creating HTTP request for download")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, release.DownloadURL, nil)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to create download request")
		return "", fmt.Errorf("failed to create download request: %w", err)
	}

	s.logger.Info().Msg("Executing HTTP request to download update")
	startTime := time.Now()

	resp, err := s.downloadClient.Do(req)
	if err != nil {
		s.logger.Error().
			Err(err).
			Dur("elapsed", time.Since(startTime)).
			Bool("contextCanceled", ctx.Err() != nil).
			Msg("HTTP request failed")
		if ctx.Err() != nil {
			s.logger.Error().Err(ctx.Err()).Msg("Context error details")
		}
		return "", fmt.Errorf("failed to download update: %w", err)
	}
	defer resp.Body.Close()

	s.logger.Info().
		Int("statusCode", resp.StatusCode).
		Int64("contentLength", resp.ContentLength).
		Dur("responseTime", time.Since(startTime)).
		Msg("Received HTTP response")

	if resp.StatusCode != http.StatusOK {
		s.logger.Error().Int("statusCode", resp.StatusCode).Msg("Download returned non-200 status")
		return "", fmt.Errorf("download returned status %d", resp.StatusCode)
	}

	tmpDir := os.TempDir()
	downloadPath := filepath.Join(tmpDir, "slipstream-update", release.AssetName)
	s.logger.Info().Str("path", downloadPath).Msg("Creating download file")

	if err := os.MkdirAll(filepath.Dir(downloadPath), 0755); err != nil {
		s.logger.Error().Err(err).Str("dir", filepath.Dir(downloadPath)).Msg("Failed to create download directory")
		return "", fmt.Errorf("failed to create download directory: %w", err)
	}

	file, err := os.Create(downloadPath)
	if err != nil {
		s.logger.Error().Err(err).Str("path", downloadPath).Msg("Failed to create download file")
		return "", fmt.Errorf("failed to create download file: %w", err)
	}
	defer file.Close()

	totalSize := float64(release.AssetSize)
	totalMB := totalSize / (1024 * 1024)
	var downloaded int64

	buf := make([]byte, 32*1024)
	lastUpdate := time.Now()
	lastLogTime := time.Now()

	s.logger.Info().Float64("totalMB", totalMB).Msg("Starting download loop")

	for {
		select {
		case <-ctx.Done():
			s.logger.Warn().
				Err(ctx.Err()).
				Int64("downloadedBytes", downloaded).
				Float64("percentComplete", float64(downloaded)/totalSize*100).
				Msg("Download cancelled via context")
			os.Remove(downloadPath)
			return "", ctx.Err()
		default:
		}

		n, err := resp.Body.Read(buf)
		if n > 0 {
			if _, writeErr := file.Write(buf[:n]); writeErr != nil {
				s.logger.Error().Err(writeErr).Int64("downloadedBytes", downloaded).Msg("Failed to write to download file")
				os.Remove(downloadPath)
				return "", fmt.Errorf("failed to write download: %w", writeErr)
			}
			downloaded += int64(n)

			if time.Since(lastUpdate) > 100*time.Millisecond {
				progress := float64(downloaded) / totalSize * 100
				downloadedMB := float64(downloaded) / (1024 * 1024)
				s.setProgress(progress, downloadedMB, totalMB)
				lastUpdate = time.Now()
			}

			if time.Since(lastLogTime) > 5*time.Second {
				s.logger.Info().
					Int64("downloadedBytes", downloaded).
					Float64("percentComplete", float64(downloaded)/totalSize*100).
					Msg("Download progress")
				lastLogTime = time.Now()
			}
		}

		if err == io.EOF {
			s.logger.Info().Int64("totalBytes", downloaded).Msg("Download complete (EOF reached)")
			break
		}
		if err != nil {
			s.logger.Error().
				Err(err).
				Int64("downloadedBytes", downloaded).
				Float64("percentComplete", float64(downloaded)/totalSize*100).
				Bool("contextCanceled", ctx.Err() != nil).
				Msg("Error reading from response body")
			os.Remove(downloadPath)
			return "", fmt.Errorf("download read error: %w", err)
		}
	}

	s.setProgress(100, totalMB, totalMB)
	s.logger.Info().Str("path", downloadPath).Int64("size", downloaded).Dur("totalTime", time.Since(startTime)).Msg("Update downloaded successfully")

	return downloadPath, nil
}

func (s *Service) backupDatabase(ctx context.Context) error {
	s.logger.Info().Msg("Creating database backup before update")
	// TODO: Implement actual database backup when backup system is built
	// For now, just log a placeholder message
	return nil
}

func (s *Service) installUpdate(ctx context.Context, downloadPath string) error {
	s.setState(StateInstalling, nil)
	s.logger.Info().Str("path", downloadPath).Str("platform", runtime.GOOS).Msg("Starting installation")

	goos := runtime.GOOS
	var err error

	switch goos {
	case "windows":
		s.logger.Info().Msg("Installing Windows update")
		err = s.installWindows(ctx, downloadPath)
	case "darwin":
		s.logger.Info().Msg("Installing macOS update")
		err = s.installMacOS(ctx, downloadPath)
	case "linux":
		s.logger.Info().Msg("Installing Linux update")
		err = s.installLinux(ctx, downloadPath)
	default:
		err = fmt.Errorf("unsupported platform: %s", goos)
	}

	if err != nil {
		s.logger.Error().Err(err).Str("platform", goos).Msg("Installation failed")
		return err
	}

	s.setState(StateRestarting, nil)
	s.logger.Info().Msg("Update installed, triggering restart")

	go func() {
		time.Sleep(2 * time.Second)
		if s.restartChan != nil {
			s.restartChan <- struct{}{}
		}
	}()

	return nil
}

func (s *Service) installWindows(ctx context.Context, downloadPath string) error {
	if strings.HasSuffix(downloadPath, ".exe") {
		s.logger.Info().Str("installer", downloadPath).Msg("Launching Windows installer with /S /CLOSEAPPLICATIONS flags")
		cmd := exec.CommandContext(ctx, downloadPath, "/S", "/CLOSEAPPLICATIONS")
		if err := cmd.Start(); err != nil {
			s.logger.Error().Err(err).Str("installer", downloadPath).Msg("Failed to start Windows installer")
			return err
		}
		s.logger.Info().Int("pid", cmd.Process.Pid).Msg("Windows installer process started")
		return nil
	}
	s.logger.Error().Str("ext", filepath.Ext(downloadPath)).Msg("Unsupported Windows update format")
	return fmt.Errorf("unsupported Windows update format: %s", filepath.Ext(downloadPath))
}

func (s *Service) installMacOS(ctx context.Context, downloadPath string) error {
	if !strings.HasSuffix(downloadPath, ".dmg") {
		return fmt.Errorf("unsupported macOS update format: %s", filepath.Ext(downloadPath))
	}

	mountPoint := "/Volumes/SlipStream-Update"

	cmd := exec.CommandContext(ctx, "hdiutil", "attach", downloadPath, "-mountpoint", mountPoint, "-nobrowse", "-quiet")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to mount DMG: %w", err)
	}

	defer func() {
		exec.Command("hdiutil", "detach", mountPoint, "-quiet").Run()
	}()

	appPath := filepath.Join(mountPoint, "SlipStream.app")
	destPath := "/Applications/SlipStream.app"

	os.RemoveAll(destPath)
	cmd = exec.CommandContext(ctx, "cp", "-R", appPath, destPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to copy application: %w", err)
	}

	return nil
}

func (s *Service) installLinux(ctx context.Context, downloadPath string) error {
	ext := filepath.Ext(downloadPath)

	switch ext {
	case ".AppImage":
		execPath, err := os.Executable()
		if err != nil {
			return fmt.Errorf("failed to get executable path: %w", err)
		}
		if err := os.Rename(downloadPath, execPath); err != nil {
			cmd := exec.CommandContext(ctx, "cp", downloadPath, execPath)
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("failed to replace AppImage: %w", err)
			}
		}
		os.Chmod(execPath, 0755)
		return nil

	case ".deb":
		cmd := exec.CommandContext(ctx, "sudo", "dpkg", "-i", downloadPath)
		return cmd.Run()

	case ".rpm":
		cmd := exec.CommandContext(ctx, "sudo", "rpm", "-U", downloadPath)
		return cmd.Run()

	default:
		return fmt.Errorf("unsupported Linux update format: %s", ext)
	}
}

func (s *Service) Cancel() error {
	s.logger.Info().Msg("Cancel called - attempting to cancel update")

	s.mu.Lock()
	cancel := s.cancelFunc
	downloadPath := s.downloadPath
	s.mu.Unlock()

	s.logger.Info().
		Bool("hasCancelFunc", cancel != nil).
		Str("downloadPath", downloadPath).
		Msg("Cancel state")

	if cancel != nil {
		s.logger.Info().Msg("Invoking cancel function")
		cancel()
	}

	s.mu.Lock()
	if s.downloadPath != "" {
		s.logger.Info().Str("path", s.downloadPath).Msg("Removing partial download file")
		os.Remove(s.downloadPath)
		s.downloadPath = ""
	}
	s.mu.Unlock()

	s.setState(StateIdle, nil)
	s.logger.Info().Msg("Update cancelled successfully")
	return nil
}

func (s *Service) CheckAndAutoInstall(ctx context.Context) error {
	settings, err := s.GetSettings(ctx)
	if err != nil {
		s.logger.Warn().Err(err).Msg("Failed to get update settings, using defaults")
		settings = &Settings{AutoInstall: true}
	}

	release, err := s.CheckForUpdate(ctx)
	if err != nil {
		return err
	}

	if release == nil {
		return nil
	}

	s.logger.Info().Str("version", release.Version).Bool("autoInstall", settings.AutoInstall).Msg("Update available")

	if settings.AutoInstall {
		return s.DownloadAndInstall(ctx)
	}

	return nil
}

func ptr[T any](v T) *T {
	return &v
}

var _ websocket.UpdateStatusProvider = (*Service)(nil)

func (s *Service) GetUpdateStatus() *websocket.UpdateStatus {
	status := s.GetStatus()
	return &websocket.UpdateStatus{
		State:          string(status.State),
		CurrentVersion: status.CurrentVersion,
		LatestVersion:  getLatestVersion(status.LatestRelease),
		Progress:       status.Progress,
		DownloadedMB:   status.DownloadedMB,
		TotalMB:        status.TotalMB,
		Error:          status.Error,
	}
}

func getLatestVersion(release *ReleaseInfo) string {
	if release == nil {
		return ""
	}
	return release.Version
}
