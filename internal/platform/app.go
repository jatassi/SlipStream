package platform

type AppConfig struct {
	ServerURL string
	DataPath  string
	Port      int
	NoTray    bool
	OnQuit    func()
}

type App interface {
	Run() error
	OpenBrowser(url string) error
	Stop()
}

func IsFirstRun(dbPath string) bool {
	return !fileExists(dbPath)
}
