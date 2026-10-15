package smoke

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/jatassi/slipstream/internal/install"
)

func smokeApp(ctx context.Context, r *Run, files []string) error {
	dmg, err := r.path(files[0])
	if err != nil {
		return err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("find the home directory: %w", err)
	}
	app := filepath.Join(home, "Applications", install.ProductName+".app")
	slipstream := filepath.Join(app, "Contents", "MacOS", install.Executable)
	agent := filepath.Join(home, "Library", "LaunchAgents", install.BundleID+".plist")
	dataDir, err := install.App.DefaultDataDir(layout("darwin"))
	if err != nil {
		return err
	}
	uid := strconv.Itoa(os.Getuid())
	removeApp := func(ctx context.Context) error {
		r.tidy(ctx, "pkill", "-f", slipstream)
		r.tidy(ctx, "launchctl", "bootout", "gui/"+uid+"/"+install.BundleID)
		for _, p := range []string{agent, app} {
			if err := os.RemoveAll(p); err != nil {
				return fmt.Errorf("remove %s: %w", p, err)
			}
		}
		return nil
	}
	if err := removeApp(ctx); err != nil {
		return err
	}
	if err := os.RemoveAll(dataDir); err != nil {
		return fmt.Errorf("remove the earlier data dir: %w", err)
	}
	if err := r.installApp(ctx, dmg, app); err != nil {
		return err
	}
	if _, err := r.must(ctx, nil, "open", app); err != nil {
		return err
	}
	ready, err := awaitReady(ctx, dataDir, readFile(filepath.Join(dataDir, "logs", "slipstream.log")))
	if err != nil {
		return err
	}
	if !strings.Contains(ready.setupURL, "/setup#code="+ready.setupCode) {
		return fmt.Errorf("the setup page record holds %q, not the ready line's code %s", ready.setupURL, ready.setupCode)
	}
	st, err := r.appStatusOf(ctx, slipstream, "status")
	if err != nil {
		return err
	}
	if err := st.wantApp(install.App, true); err != nil {
		return err
	}
	if err := mustExist(agent); err != nil {
		return err
	}
	if _, err := r.must(ctx, nil, slipstream, "stop"); err != nil {
		return err
	}
	return removeApp(ctx)
}

func (r *Run) installApp(ctx context.Context, dmg, app string) error {
	mount, err := os.MkdirTemp("", "slipstream-dmg-")
	if err != nil {
		return fmt.Errorf("create the mount point: %w", err)
	}
	if _, err := r.must(ctx, nil, "hdiutil", "attach", "-nobrowse", "-readonly", "-mountpoint", mount, dmg); err != nil {
		return err
	}
	_, copyErr := r.must(ctx, nil, "ditto", filepath.Join(mount, install.ProductName+".app"), app)
	if _, err := r.must(ctx, nil, "hdiutil", "detach", mount); err != nil || copyErr != nil {
		return errors.Join(copyErr, err)
	}
	_, err = r.must(ctx, nil, "codesign", "--verify", "--deep", "--strict", app)
	return err
}
