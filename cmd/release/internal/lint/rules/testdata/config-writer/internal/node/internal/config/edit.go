package config

import (
	"path/filepath"

	"example.com/app/internal/atomicfile"
)

func Edit(dir string, data []byte) error {
	return atomicfile.Write(filepath.Join(dir, "config.yaml"), data)
}
