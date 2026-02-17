package arrimport

import (
	"os"
	"path/filepath"
	"runtime"
)

type DBCandidate struct {
	Path   string `json:"path"`
	Exists bool   `json:"exists"`
}

type DetectDBResponse struct {
	Candidates []DBCandidate `json:"candidates"`
	Found      string        `json:"found"`
}

func detectDBPaths(sourceType SourceType) DetectDBResponse {
	home, _ := os.UserHomeDir()

	var paths []string
	switch sourceType {
	case SourceTypeRadarr:
		paths = radarrPaths(home)
	case SourceTypeSonarr:
		paths = sonarrPaths(home)
	}

	var candidates []DBCandidate
	var found string
	for _, p := range paths {
		exists := fileExists(p)
		candidates = append(candidates, DBCandidate{Path: p, Exists: exists})
		if exists && found == "" {
			found = p
		}
	}

	return DetectDBResponse{
		Candidates: candidates,
		Found:      found,
	}
}

func radarrPaths(home string) []string {
	switch runtime.GOOS {
	case "darwin":
		return []string{
			filepath.Join(home, "Library", "Application Support", "Radarr", "radarr.db"),
		}
	case "linux":
		return []string{
			filepath.Join(home, ".config", "Radarr", "radarr.db"),
			"/var/lib/radarr/radarr.db",
			"/config/radarr.db",
		}
	case "windows":
		appdata := os.Getenv("APPDATA")
		if appdata == "" {
			return nil
		}
		return []string{
			filepath.Join(appdata, "Radarr", "radarr.db"),
		}
	default:
		return nil
	}
}

func sonarrPaths(home string) []string {
	switch runtime.GOOS {
	case "darwin":
		return []string{
			filepath.Join(home, ".config", "Sonarr", "sonarr.db"),
		}
	case "linux":
		return []string{
			filepath.Join(home, ".config", "Sonarr", "sonarr.db"),
			"/var/lib/sonarr/sonarr.db",
			filepath.Join(home, ".config", "NzbDrone", "nzbdrone.db"),
			"/config/sonarr.db",
		}
	case "windows":
		appdata := os.Getenv("APPDATA")
		if appdata == "" {
			return nil
		}
		return []string{
			filepath.Join(appdata, "Sonarr", "sonarr.db"),
		}
	default:
		return nil
	}
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
