package converting

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os/exec"
	"slices"

	"github.com/jatassi/slipstream/internal/converting/internal/drivers"
	"github.com/jatassi/slipstream/internal/proctree"
)

// readSignature runs ffmpeg -version and reads the Node's adapters, and
// signs the two.
func readSignature(ctx context.Context, ffmpeg string, environ []string, read drivers.Reader) (Signature, []Device, error) {
	cmd := exec.CommandContext(ctx, ffmpeg, "-version")
	cmd.Env = environ
	version, err := proctree.Output(cmd)
	if err != nil {
		return "", nil, fmt.Errorf("run ffmpeg -version: %w", err)
	}
	adapters, err := read(ctx)
	if err != nil {
		return "", nil, err
	}
	devices := make([]Device, len(adapters))
	for i, a := range adapters {
		devices[i] = Device(a)
	}
	return sign(string(version), devices), devices, nil
}

// sign hashes ffmpeg's version text and the devices' driver versions in
// sorted order, so the order the OS lists adapters in doesn't change it.
func sign(version string, devices []Device) Signature {
	versions := make([]string, len(devices))
	for i, d := range devices {
		versions[i] = d.Driver
	}
	slices.Sort(versions)
	h := sha256.New()
	_, _ = fmt.Fprintf(h, "%q", append([]string{version}, versions...))
	return Signature(hex.EncodeToString(h.Sum(nil)))
}
