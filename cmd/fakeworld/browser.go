package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"

	"github.com/jatassi/slipstream/fakes/world"
)

// cliConfig is a playwright cli config: Chromium reaching every Node by
// name, one shell's viewport and device, and the run's browser output.
type cliConfig struct {
	Browser   cliBrowser `json:"browser"`
	OutputDir string     `json:"outputDir"`
}

type cliBrowser struct {
	BrowserName    string            `json:"browserName"`
	LaunchOptions  cliLaunchOptions  `json:"launchOptions"`
	ContextOptions cliContextOptions `json:"contextOptions"`
}

type cliLaunchOptions struct {
	Args []string `json:"args"`
}

type cliContextOptions struct {
	Viewport          cliViewport `json:"viewport"`
	DeviceScaleFactor float64     `json:"deviceScaleFactor"`
	IsMobile          bool        `json:"isMobile"`
	HasTouch          bool        `json:"hasTouch"`
}

type cliViewport struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

func browserCommand(args []string) int {
	if len(args) != 1 {
		return usage(errors.New("browser: takes one run dir"))
	}
	runDir := args[0]
	ctx, stop := interrupted()
	defer stop()
	return journal(ctx, runDir, "browser", nil, func(context.Context, world.Controls, world.Step) (int, map[string]string) {
		m, err := world.ReadManifest(runDir)
		if err != nil {
			return failed("browser", err), nil
		}
		files := map[string]string{}
		for _, shell := range slices.Sorted(maps.Keys(m.Browser.Config)) {
			rel := m.Browser.Config[shell]
			dst := filepath.Join(m.RunDir, filepath.FromSlash(rel))
			if err := writeCLIConfig(dst, cliConfigFor(m, shell)); err != nil {
				return failed("browser", err), files
			}
			files[shell] = rel
			fmt.Println(dst)
		}
		return exitOK, files
	})
}

func cliConfigFor(m world.Manifest, shell string) cliConfig {
	p := m.Browser.Presets[shell]
	return cliConfig{
		Browser: cliBrowser{
			BrowserName:   "chromium",
			LaunchOptions: cliLaunchOptions{Args: []string{"--host-resolver-rules=" + m.Browser.HostResolverRules}},
			ContextOptions: cliContextOptions{
				Viewport:          cliViewport{Width: p.Width, Height: p.Height},
				DeviceScaleFactor: p.DeviceScaleFactor,
				IsMobile:          p.IsMobile,
				HasTouch:          p.HasTouch,
			},
		},
		OutputDir: filepath.Join(m.RunDir, "browser", "output"),
	}
}

func writeCLIConfig(path string, c cliConfig) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("browser config: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("browser config: %w", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("browser config: %w", err)
	}
	return nil
}
