package world

import (
	"errors"
	"fmt"
	"io/fs"
	"slices"

	"github.com/jatassi/slipstream/fakes/internal/fsinfo"
)

// FileInfo is a file as the side that received it sees it: its size, link
// count and BLAKE3.
type FileInfo = fsinfo.Info

func StatFile(path string) (FileInfo, error) { return fsinfo.Stat(path) }

// MatchRelease reads the file at path and checks that it holds the
// payload of the manifest's release name: its size and its BLAKE3.
func MatchRelease(m Manifest, path, name string) (FileInfo, error) {
	i := slices.IndexFunc(m.Releases, func(r Release) bool { return r.Name == name })
	if i < 0 {
		return FileInfo{}, fmt.Errorf("the manifest lists no release %s", name)
	}
	rel := m.Releases[i]
	info, err := fsinfo.Stat(path)
	if err != nil {
		return FileInfo{}, err
	}
	if info.Size != rel.Size || info.BLAKE3 != rel.BLAKE3 {
		return info, fmt.Errorf("%s holds %d bytes with BLAKE3 %s, not release %s's %d bytes with BLAKE3 %s", path, info.Size, info.BLAKE3, name, rel.Size, rel.BLAKE3)
	}
	return info, nil
}

// ReadLastManifest reads the manifest of the run in runDir: manifest.json
// while it is live, else the manifest.stopped.json its stop left.
func ReadLastManifest(runDir string) (Manifest, error) {
	m, err := ReadManifest(runDir)
	if errors.Is(err, fs.ErrNotExist) {
		return readManifestFile(runDir, stoppedManifestFile)
	}
	return m, err
}
