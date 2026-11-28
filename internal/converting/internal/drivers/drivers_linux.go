package drivers

import (
	"bufio"
	"cmp"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// vendors are the PCI vendor ids of the GPU makers ffmpeg has a hardware
// path for.
var vendors = map[string]string{"0x8086": "Intel", "0x1002": "AMD", "0x10de": "NVIDIA"}

// Local reads the NVIDIA module's version for each NVIDIA GPU, and the
// VA-API driver string ffmpeg prints at verbose level for each render
// node that opens as a VA-API device.
func Local(ffmpeg string, environ []string) Reader {
	return func(ctx context.Context) ([]Adapter, error) {
		out, err := nvidia()
		if err != nil {
			return nil, err
		}
		nodes, err := filepath.Glob("/dev/dri/renderD*")
		if err != nil {
			return nil, fmt.Errorf("list the render nodes: %w", err)
		}
		for _, node := range nodes {
			cmd := exec.CommandContext(ctx, ffmpeg, "-hide_banner", "-v", "verbose", "-init_hw_device", "vaapi=va:"+node) //nolint:gosec // the Node's ffmpeg on a render node
			cmd.Env = environ
			// ffmpeg exits non-zero with no output named, after it opened
			// the device or failed to.
			printed, err := cmd.CombinedOutput()
			if exit := (*exec.ExitError)(nil); err != nil && !errors.As(err, &exit) {
				return nil, fmt.Errorf("read %s's VA-API driver: %w", node, err)
			}
			driver := vaapiDriver(string(printed))
			if driver == "" {
				continue
			}
			vendor, _ := os.ReadFile("/sys/class/drm/" + filepath.Base(node) + "/device/vendor")
			id := strings.TrimSpace(string(vendor))
			out = append(out, Adapter{Name: filepath.Base(node), Vendor: cmp.Or(vendors[id], id), Driver: driver})
		}
		return out, nil
	}
}

// nvidia is an adapter per GPU the NVIDIA module drives, named by its
// model, with the module's version.
func nvidia() ([]Adapter, error) {
	version, err := os.ReadFile("/sys/module/nvidia/version")
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read the NVIDIA module's version: %w", err)
	}
	infos, err := filepath.Glob("/proc/driver/nvidia/gpus/*/information")
	if err != nil {
		return nil, fmt.Errorf("list the NVIDIA GPUs: %w", err)
	}
	var out []Adapter
	for _, info := range infos {
		text, err := os.ReadFile(info) //nolint:gosec // an entry of /proc/driver/nvidia/gpus
		if err != nil {
			return nil, fmt.Errorf("read an NVIDIA GPU's information: %w", err)
		}
		out = append(out, Adapter{Name: field(string(text), "Model:"), Vendor: "NVIDIA", Driver: strings.TrimSpace(string(version))})
	}
	return out, nil
}

// vaapiDriver is the driver string of the "VAAPI driver: ..." line ffmpeg
// prints once it opens a VA-API device, empty when it printed none.
func vaapiDriver(printed string) string {
	return strings.TrimSuffix(field(printed, "VAAPI driver:"), ".")
}

// field is what follows the first line's key, trimmed, empty when no line
// holds key.
func field(text, key string) string {
	lines := bufio.NewScanner(strings.NewReader(text))
	for lines.Scan() {
		if _, value, ok := strings.Cut(lines.Text(), key); ok {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
