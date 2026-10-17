package node

import (
	"os"
	"path/filepath"

	"example.com/app/internal/atomicfile"
	"example.com/app/internal/node/internal/config"
)

func save(dir string, data []byte) error {
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), data, 0o600); err != nil {
		return err
	}
	return atomicfile.Write(dir+"/config.yaml", data)
}

func saveBySource(dir string, data []byte) error {
	return os.WriteFile(filepath.Join(dir, string(config.SourceYAML)), data, 0o600)
}

func read(dir string) ([]byte, error) {
	return os.ReadFile(filepath.Join(dir, "config.yaml"))
}

func logs(dir string) error {
	f, err := os.Create(filepath.Join(dir, "slipstream.log"))
	if err != nil {
		return err
	}
	return f.Close()
}
