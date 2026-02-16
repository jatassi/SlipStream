package mock

import (
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	MockPathPrefix = "/mock/"
	MockMoviesPath = "/mock/movies"
	MockTVPath     = "/mock/tv"
)

type FileType int

const (
	FileTypeDirectory FileType = iota
	FileTypeVideo
)

type VirtualFile struct {
	Name     string
	Path     string
	Type     FileType
	Size     int64
	ModTime  time.Time
	Children map[string]*VirtualFile
}

type VirtualFS struct {
	mu   sync.RWMutex
	root *VirtualFile
}

var (
	instance *VirtualFS
	once     sync.Once
)

func GetInstance() *VirtualFS {
	once.Do(func() {
		instance = &VirtualFS{}
		instance.initialize()
	})
	return instance
}

func ResetInstance() {
	if instance != nil {
		instance.mu.Lock()
		instance.initialize()
		instance.mu.Unlock()
	}
}

func (vfs *VirtualFS) initialize() {
	vfs.root = &VirtualFile{
		Name:     "mock",
		Path:     "/mock",
		Type:     FileTypeDirectory,
		ModTime:  time.Now(),
		Children: make(map[string]*VirtualFile),
	}

	vfs.createDirectory("/mock/movies")
	vfs.createDirectory("/mock/tv")
	vfs.createDirectory("/mock/downloads")

	vfs.populateMovies()
	vfs.populateTV()
}

func (vfs *VirtualFS) populateMovies() {
	// Movies WITH files - multi-version testing scenarios
	// Each movie has multiple quality tiers for slot assignment testing
	availableMovies := []struct {
		title string
		year  int
		files []struct {
			name string
			size int64
		}
	}{
		// The Matrix - 3 versions for full multi-slot testing
		// Slot 1: 4K DV for local playback
		// Slot 2: 1080p for remote streaming
		// Slot 3: 720p for mobile (or review queue testing with 2 slots)
		{
			title: "The Matrix",
			year:  1999,
			files: []struct {
				name string
				size int64
			}{
				{"The.Matrix.1999.2160p.UHD.BluRay.Remux.HEVC.DV.TrueHD.7.1.Atmos-GROUP.mkv", 65 * 1024 * 1024 * 1024},
				{"The.Matrix.1999.1080p.BluRay.x264.DTS-HD.MA.5.1-GROUP.mkv", 12 * 1024 * 1024 * 1024},
				{"The.Matrix.1999.720p.WEB-DL.x264.AAC.2.0-GROUP.mkv", 4 * 1024 * 1024 * 1024},
			},
		},
		// Inception - 2 versions: 4K HDR10 and 1080p SDR
		{
			title: "Inception",
			year:  2010,
			files: []struct {
				name string
				size int64
			}{
				{"Inception.2010.2160p.UHD.BluRay.HDR10.HEVC.DTS-HD.MA.5.1-GROUP.mkv", 55 * 1024 * 1024 * 1024},
				{"Inception.2010.1080p.BluRay.x264.DTS-GROUP.mkv", 11 * 1024 * 1024 * 1024},
			},
		},
		// Dune - 2 versions: 4K DV HDR10 combo and 1080p WEB-DL
		{
			title: "Dune",
			year:  2021,
			files: []struct {
				name string
				size int64
			}{
				{"Dune.2021.2160p.UHD.BluRay.Remux.DV.HDR10.HEVC.TrueHD.7.1.Atmos-GROUP.mkv", 70 * 1024 * 1024 * 1024},
				{"Dune.2021.1080p.HMAX.WEB-DL.DDP5.1.Atmos.H.264-FLUX.mkv", 10 * 1024 * 1024 * 1024},
			},
		},
		// Pulp Fiction - single WEB-DL 1080p (upgradable with Bluray-1080p cutoff)
		{
			title: "Pulp Fiction",
			year:  1994,
			files: []struct {
				name string
				size int64
			}{
				{"Pulp.Fiction.1994.1080p.WEB-DL.x264.DDP5.1-GROUP.mkv", 8 * 1024 * 1024 * 1024},
			},
		},
		// Fight Club - single 720p BluRay (upgradable with Bluray-1080p cutoff)
		{
			title: "Fight Club",
			year:  1999,
			files: []struct {
				name string
				size int64
			}{
				{"Fight.Club.1999.720p.BluRay.x264.DTS-GROUP.mkv", 5 * 1024 * 1024 * 1024},
			},
		},
	}

	// Movies WITHOUT files (missing/wanted) - for search/download testing
	// These exist in metadata mock but have no files - user can search & download
	// Dune Part Two (693134), Oppenheimer (872585), Barbie (346698),
	// The Dark Knight (155), Interstellar (157336), Avatar (19995), etc.
	// These are intentionally NOT created in the virtual filesystem

	for _, m := range availableMovies {
		folderName := m.title + " (" + itoa(m.year) + ")"
		folderPath := "/mock/movies/" + folderName
		vfs.createDirectory(folderPath)

		for _, f := range m.files {
			vfs.createFile(folderPath+"/"+f.name, f.size)
		}
	}
}

func (vfs *VirtualFS) populateTV() {
	// TV shows with multi-version support for slot testing
	// Some seasons have multiple quality tiers, others have single quality

	// Multi-version shows: seasons with multiple quality files per episode
	multiVersionShows := []struct {
		title   string
		seasons []struct {
			num       int
			episodes  int
			qualities []string // Multiple quality tiers per episode
		}
	}{
		// Breaking Bad - Season 1 has both 4K and 1080p for multi-slot testing
		// S2-S3 are 1080p BluRay (available), S4-S5 are WEB-DL (upgradable)
		// TVDB: 81189
		{
			title: "Breaking Bad",
			seasons: []struct {
				num       int
				episodes  int
				qualities []string
			}{
				{1, 7, []string{
					"2160p.UHD.BluRay.Remux.HEVC.DTS-HD.MA.5.1",
					"1080p.BluRay.x264.DTS-HD.MA.5.1",
				}},
				{2, 13, []string{"1080p.BluRay.x264.DTS-HD.MA.5.1"}},
				{3, 13, []string{"1080p.BluRay.x264.DTS-HD.MA.5.1"}},
				{4, 13, []string{"1080p.AMZN.WEB-DL.DDP5.1.H.264"}},
				{5, 16, []string{"1080p.AMZN.WEB-DL.DDP5.1.H.264"}},
			},
		},
		// Game of Thrones - Season 1 has 4K HDR10 and 1080p for multi-slot testing
		// Seasons 2-3 are 720p BluRay (upgradable), seasons 4-8 missing
		// TVDB: 121361
		{
			title: "Game of Thrones",
			seasons: []struct {
				num       int
				episodes  int
				qualities []string
			}{
				{1, 10, []string{
					"2160p.UHD.BluRay.HDR10.HEVC.TrueHD.7.1.Atmos",
					"1080p.BluRay.x264.DTS-HD.MA.5.1",
				}},
				{2, 10, []string{"720p.BluRay.x264.DTS-GROUP"}},
				{3, 10, []string{"720p.BluRay.x264.DTS-GROUP"}},
			},
		},
	}

	// Single-version shows: only one quality tier per season
	singleVersionShows := []struct {
		title   string
		seasons []struct {
			num      int
			episodes int
			quality  string
		}
	}{
		// Stranger Things - all 1080p WEB-DL (upgrade to 2160p via indexer)
		// Season 4 partial (5 of 9 episodes)
		// TVDB: 305288
		{
			title: "Stranger Things",
			seasons: []struct {
				num      int
				episodes int
				quality  string
			}{
				{1, 8, "1080p.NF.WEB-DL.DDP5.1.H.264"},
				{2, 9, "1080p.NF.WEB-DL.DDP5.1.H.264"},
				{3, 8, "1080p.NF.WEB-DL.DDP5.1.H.264"},
				{4, 5, "1080p.NF.WEB-DL.DDP5.1.H.264"}, // Only 5 of 9 episodes
			},
		},
		// The Mandalorian - seasons 1-2 at 1080p, season 3 missing
		// TVDB: 361753
		{
			title: "The Mandalorian",
			seasons: []struct {
				num      int
				episodes int
				quality  string
			}{
				{1, 8, "1080p.DSNP.WEB-DL.DDP5.1.Atmos.H.264"},
				{2, 8, "1080p.DSNP.WEB-DL.DDP5.1.Atmos.H.264"},
				// Season 3 missing - available in indexer
			},
		},
	}

	// TV shows WITHOUT files - for search/grab testing
	// The Boys (TVDB: 355567), The Simpsons (TVDB: 71663)
	// These exist in metadata/indexer mocks but have no files in VFS

	for _, show := range multiVersionShows {
		vfs.populateMultiVersionShow(show.title, show.seasons)
	}

	for _, show := range singleVersionShows {
		vfs.populateSingleVersionShow(show.title, show.seasons)
	}
}

func (vfs *VirtualFS) populateMultiVersionShow(title string, seasons []struct {
	num       int
	episodes  int
	qualities []string
}) {
	showPath := "/mock/tv/" + title
	vfs.createDirectory(showPath)

	for _, season := range seasons {
		seasonPath := showPath + "/Season " + padNumber(season.num)
		vfs.createDirectory(seasonPath)

		for ep := 1; ep <= season.episodes; ep++ {
			for _, quality := range season.qualities {
				filename := title + " - S" + padNumber(season.num) + "E" + padNumber(ep) + " - " + quality + "-GROUP.mkv"
				fileSize := episodeFileSize(quality, ep)
				vfs.createFile(seasonPath+"/"+filename, fileSize)
			}
		}
	}
}

func (vfs *VirtualFS) populateSingleVersionShow(title string, seasons []struct {
	num      int
	episodes int
	quality  string
}) {
	showPath := "/mock/tv/" + title
	vfs.createDirectory(showPath)

	for _, season := range seasons {
		seasonPath := showPath + "/Season " + padNumber(season.num)
		vfs.createDirectory(seasonPath)

		for ep := 1; ep <= season.episodes; ep++ {
			filename := title + " - S" + padNumber(season.num) + "E" + padNumber(ep) + " - " + season.quality + "-GROUP.mkv"
			fileSize := int64((2 + ep%3) * 1024 * 1024 * 1024)
			vfs.createFile(seasonPath+"/"+filename, fileSize)
		}
	}
}

func episodeFileSize(quality string, ep int) int64 {
	if strings.Contains(quality, "2160p") {
		return int64((8 + ep%3) * 1024 * 1024 * 1024)
	}
	return int64((2 + ep%3) * 1024 * 1024 * 1024)
}

func (vfs *VirtualFS) createDirectory(path string) {
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	current := vfs.root

	for i, part := range parts {
		if part == "" || part == "mock" && i == 0 {
			continue
		}

		if current.Children == nil {
			current.Children = make(map[string]*VirtualFile)
		}

		child, exists := current.Children[part]
		if !exists {
			fullPath := "/" + strings.Join(parts[:i+1], "/")
			child = &VirtualFile{
				Name:     part,
				Path:     fullPath,
				Type:     FileTypeDirectory,
				ModTime:  time.Now(),
				Children: make(map[string]*VirtualFile),
			}
			current.Children[part] = child
		}
		current = child
	}
}

func (vfs *VirtualFS) createFile(path string, size int64) {
	dir := filepath.Dir(path)
	vfs.createDirectory(dir)

	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	current := vfs.root

	for i, part := range parts[:len(parts)-1] {
		if part == "" || part == "mock" && i == 0 {
			continue
		}
		current = current.Children[part]
	}

	filename := parts[len(parts)-1]
	if current.Children == nil {
		current.Children = make(map[string]*VirtualFile)
	}

	current.Children[filename] = &VirtualFile{
		Name:    filename,
		Path:    path,
		Type:    FileTypeVideo,
		Size:    size,
		ModTime: time.Now(),
	}
}

func (vfs *VirtualFS) getNode(path string) *VirtualFile {
	if path == "/mock" || path == "/mock/" {
		return vfs.root
	}

	path = strings.TrimSuffix(path, "/")
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	current := vfs.root

	for i, part := range parts {
		if part == "" || (part == "mock" && i == 0) {
			continue
		}
		if current.Children == nil {
			return nil
		}
		child, exists := current.Children[part]
		if !exists {
			return nil
		}
		current = child
	}

	return current
}

func (vfs *VirtualFS) Exists(path string) bool {
	vfs.mu.RLock()
	defer vfs.mu.RUnlock()
	return vfs.getNode(path) != nil
}

func (vfs *VirtualFS) IsDirectory(path string) bool {
	vfs.mu.RLock()
	defer vfs.mu.RUnlock()
	node := vfs.getNode(path)
	return node != nil && node.Type == FileTypeDirectory
}

func (vfs *VirtualFS) ListDirectory(path string) ([]*VirtualFile, error) {
	vfs.mu.RLock()
	defer vfs.mu.RUnlock()

	node := vfs.getNode(path)
	if node == nil {
		return nil, nil
	}

	if node.Type != FileTypeDirectory {
		return nil, nil
	}

	entries := make([]*VirtualFile, 0, len(node.Children))
	for _, child := range node.Children {
		entries = append(entries, child)
	}

	sort.Slice(entries, func(i, j int) bool {
		return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name)
	})

	return entries, nil
}

func (vfs *VirtualFS) GetFile(path string) *VirtualFile {
	vfs.mu.RLock()
	defer vfs.mu.RUnlock()
	return vfs.getNode(path)
}

func (vfs *VirtualFS) WalkDir(root string, fn func(path string, file *VirtualFile) error) error {
	vfs.mu.RLock()
	defer vfs.mu.RUnlock()
	return vfs.walkDirInternal(root, fn)
}

func (vfs *VirtualFS) walkDirInternal(path string, fn func(path string, file *VirtualFile) error) error {
	node := vfs.getNode(path)
	if node == nil {
		return nil
	}

	if err := fn(node.Path, node); err != nil {
		return err
	}

	if node.Type == FileTypeDirectory && node.Children != nil {
		names := make([]string, 0, len(node.Children))
		for name := range node.Children {
			names = append(names, name)
		}
		sort.Strings(names)

		for _, name := range names {
			child := node.Children[name]
			if err := vfs.walkDirInternal(child.Path, fn); err != nil {
				return err
			}
		}
	}

	return nil
}

func (vfs *VirtualFS) AddFile(path string, size int64) {
	vfs.mu.Lock()
	defer vfs.mu.Unlock()
	vfs.createFile(path, size)
}

func (vfs *VirtualFS) AddDirectory(path string) {
	vfs.mu.Lock()
	defer vfs.mu.Unlock()
	vfs.createDirectory(path)
}

func (vfs *VirtualFS) Remove(path string) bool {
	vfs.mu.Lock()
	defer vfs.mu.Unlock()

	dir := filepath.Dir(path)
	name := filepath.Base(path)

	parent := vfs.getNode(dir)
	if parent == nil || parent.Children == nil {
		return false
	}

	if _, exists := parent.Children[name]; exists {
		delete(parent.Children, name)
		return true
	}
	return false
}

func IsMockPath(path string) bool {
	return strings.HasPrefix(path, MockPathPrefix) || path == "/mock"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	digits := ""
	for n > 0 {
		digits = string(rune('0'+n%10)) + digits
		n /= 10
	}
	return digits
}

func padNumber(n int) string {
	if n < 10 {
		return "0" + itoa(n)
	}
	return itoa(n)
}
