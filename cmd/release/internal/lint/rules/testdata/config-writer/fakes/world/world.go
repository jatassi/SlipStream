package world

import "os"

func pin(dir string, data []byte) error {
	return os.WriteFile(dir+"/config.yaml", data, 0o600)
}
