package smoke

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"fmt"
	"net/netip"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/install"
	"github.com/jatassi/slipstream/internal/joincode"
)

// joinCode is a well-formed Join Code for a Controller nobody runs. A host
// command refuses a foreign data dir before it dials anything.
func joinCode() (string, error) {
	key, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", fmt.Errorf("make a Join Code: %w", err)
	}
	c, err := joincode.New(key, [joincode.SecretSize]byte{}, []netip.AddrPort{netip.MustParseAddrPort("127.0.0.1:9")})
	if err != nil {
		return "", fmt.Errorf("make a Join Code: %w", err)
	}
	return c.String(), nil
}

// sudo prefixes a command a person runs as root.
func sudo(args ...string) []string {
	if os.Geteuid() == 0 {
		return args
	}
	return append([]string{"sudo"}, args...)
}

// system is where a deb or rpm smoke runs its commands: this runner, or
// a container with systemd as PID 1.
type system struct {
	// root and as run a command as root and as another user.
	root func(args ...string) []string
	as   func(user string, args ...string) []string
	// install and remove the package with the distribution's own tool.
	install, remove []string
}

// smokeService proves a system package: installing it leaves a running
// service owned by the slipstream user, a root host command is refused,
// and removing it stops the service and keeps the data.
func (r *Run) smokeService(ctx context.Context, s system, k install.Kind) error {
	// The journal outlives the package, so only this install's records count.
	since := "@" + strconv.FormatInt(time.Now().Unix(), 10)
	if _, err := r.must(ctx, nil, s.root(s.install...)...); err != nil {
		return err
	}
	res, err := r.exec(ctx, nil, s.root("systemctl", "is-active", install.UnitName)...)
	if err != nil {
		return err
	}
	if state := strings.TrimSpace(res.stdout); state != "active" {
		return fmt.Errorf("after install, %s is %q, want active", install.UnitName, state)
	}
	if _, err := awaitReady(ctx, "journalctl -u slipstream", func() (string, error) {
		res, err := r.exec(ctx, nil, s.root("journalctl", "-u", install.UnitName, "--since", since, "-o", "cat", "--no-pager")...)
		return res.stdout, err
	}); err != nil {
		return err
	}
	st, err := r.statusOf(ctx, s.as(install.SystemUser, install.Executable, "status")...)
	if err != nil {
		return err
	}
	if err := st.want(k, "run"); err != nil {
		return err
	}
	code, err := joinCode()
	if err != nil {
		return err
	}
	res, err = r.exec(ctx, nil, s.root(install.Executable, "join", code)...)
	if err != nil {
		return err
	}
	owner := "data dir /var/lib/slipstream belongs to slipstream; run: sudo -u slipstream slipstream join ..."
	if res.code != 3 || !strings.Contains(res.stderr, owner) {
		return fmt.Errorf("join as root: %s; want exit 3 and %q", res, owner)
	}
	if _, err := r.must(ctx, nil, s.root(s.remove...)...); err != nil {
		return err
	}
	if res, _ := r.exec(ctx, nil, s.root("systemctl", "is-active", install.UnitName)...); strings.TrimSpace(res.stdout) == "active" {
		return fmt.Errorf("after remove, %s is still active", install.UnitName)
	}
	if res, err := r.exec(ctx, nil, s.root("test", "-d", "/var/lib/slipstream")...); err != nil || res.code != 0 {
		return errors.New("removing the package deleted /var/lib/slipstream")
	}
	return nil
}

func smokeDeb(ctx context.Context, r *Run, files []string) error {
	deb, err := r.path(files[0])
	if err != nil {
		return err
	}
	if res, _ := r.exec(ctx, nil, "dpkg", "-s", install.Executable); res.code == 0 {
		r.tidy(ctx, sudo("apt-get", "remove", "-y", install.Executable)...)
	}
	r.tidy(ctx, sudo("rm", "-rf", "/var/lib/slipstream")...)
	return r.smokeService(ctx, system{
		root:    sudo,
		as:      func(user string, args ...string) []string { return append([]string{"sudo", "-u", user}, args...) },
		install: []string{"env", "DEBIAN_FRONTEND=noninteractive", "apt-get", "install", "-y", deb},
		remove:  []string{"env", "DEBIAN_FRONTEND=noninteractive", "apt-get", "remove", "-y", install.Executable},
	}, install.Deb)
}

// The rpm smoke runs in a Rocky Linux 8 container with systemd as PID 1,
// which also proves the glibc 2.28 floor.
const (
	rpmImage     = "slipstream-smoke-rockylinux8"
	rpmContainer = "slipstream-smoke-rpm"
)

const rpmDockerfile = `FROM rockylinux:8
RUN dnf -y install systemd sudo procps-ng && dnf clean all
CMD ["/usr/sbin/init"]
`

func smokeRPM(ctx context.Context, r *Run, files []string) error {
	rpm, err := r.path(files[0])
	if err != nil {
		return err
	}
	r.tidy(ctx, "docker", "rm", "-f", rpmContainer)
	build := exec.CommandContext(ctx, "docker", "build", "-t", rpmImage, "-")
	build.Stdin = strings.NewReader(rpmDockerfile)
	build.Stdout, build.Stderr = r.Log, r.Log
	if err := build.Run(); err != nil {
		return fmt.Errorf("build the Rocky Linux 8 image: %w", err)
	}
	if _, err := r.must(ctx, nil, "docker", "run", "-d", "--name", rpmContainer, "--privileged", "--cgroupns=host",
		"-v", "/sys/fs/cgroup:/sys/fs/cgroup:rw", "-v", filepath.Dir(rpm)+":/pkg:ro", rpmImage); err != nil {
		return err
	}
	defer r.tidy(context.WithoutCancel(ctx), "docker", "rm", "-f", rpmContainer)
	in := func(args ...string) []string { return append([]string{"docker", "exec", rpmContainer}, args...) }
	if err := r.awaitSystemd(ctx, in); err != nil {
		return err
	}
	return r.smokeService(ctx, system{
		root: in,
		as: func(user string, args ...string) []string {
			return in(append([]string{"sudo", "-u", user}, args...)...)
		},
		install: []string{"dnf", "install", "-y", "/pkg/" + filepath.Base(rpm)},
		remove:  []string{"dnf", "remove", "-y", install.Executable},
	}, install.RPM)
}

// awaitSystemd waits for the container's systemd to finish booting. Its
// bus is missing for the first moments, so is-system-running --wait alone
// can fail before it waits.
func (r *Run) awaitSystemd(ctx context.Context, in func(args ...string) []string) error {
	deadline := time.Now().Add(waitFor)
	for {
		res, err := r.exec(ctx, nil, in("systemctl", "is-system-running", "--wait")...)
		if err != nil {
			return err
		}
		// A container's missing devices leave systemd degraded, which is enough.
		state := strings.TrimSpace(res.stdout)
		if state == "running" || state == "degraded" {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("systemd in the container is %q after %s, want running", state, waitFor)
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("wait for systemd in the container: %w", ctx.Err())
		case <-time.After(time.Second):
		}
	}
}

// userSession gives the smoke what a person's login session has: a
// systemd user manager, kept by linger, and XDG_RUNTIME_DIR naming it.
func (r *Run) userSession(ctx context.Context) ([]string, error) {
	u, err := user.Current()
	if err != nil {
		return nil, fmt.Errorf("find the current user: %w", err)
	}
	if _, err := r.must(ctx, nil, sudo("loginctl", "enable-linger", u.Username)...); err != nil {
		return nil, err
	}
	runtime := "/run/user/" + u.Uid
	deadline := time.Now().Add(waitFor)
	for !exists(filepath.Join(runtime, "bus")) {
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("no systemd user manager for %s: %s/bus is missing after enable-linger", u.Username, runtime)
		}
		time.Sleep(time.Second)
	}
	return []string{"XDG_RUNTIME_DIR=" + runtime}, nil
}

// removeUserService clears what an earlier user service smoke left.
func (r *Run) removeUserService(ctx context.Context, env []string, k install.Kind) error {
	_, _ = r.exec(ctx, env, "systemctl", "--user", "disable", "--now", install.UnitName)
	unit, err := install.UserUnitPath(k, layout("linux"))
	if err != nil {
		return err
	}
	if err := os.Remove(unit); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove the earlier unit: %w", err)
	}
	_, _ = r.exec(ctx, env, "systemctl", "--user", "daemon-reload")
	dataDir, err := k.DefaultDataDir(layout("linux"))
	if err != nil {
		return err
	}
	if err := os.RemoveAll(dataDir); err != nil {
		return fmt.Errorf("remove the earlier data dir: %w", err)
	}
	return nil
}

// smokeUserService proves service install --user on exe: the unit runs,
// linger is on, a rerun keeps the PID, and uninstall removes the unit.
func (r *Run) smokeUserService(ctx context.Context, env []string, exe string, k install.Kind) error {
	out, err := r.must(ctx, env, exe, "service", "install", "--user")
	if err != nil {
		return err
	}
	if !strings.HasPrefix(out, "slipstream service running (PID ") {
		return fmt.Errorf("service install --user printed %q", out)
	}
	if out, _ := r.must(ctx, env, "systemctl", "--user", "is-active", install.UnitName); strings.TrimSpace(out) != "active" {
		return fmt.Errorf("after service install --user, %s is %q, want active", install.UnitName, strings.TrimSpace(out))
	}
	u, err := user.Current()
	if err != nil {
		return fmt.Errorf("find the current user: %w", err)
	}
	if out, _ := r.must(ctx, env, "loginctl", "show-user", u.Username, "--property=Linger"); strings.TrimSpace(out) != "Linger=yes" {
		return fmt.Errorf("linger is %q, want Linger=yes", strings.TrimSpace(out))
	}
	dataDir, err := k.DefaultDataDir(layout("linux"))
	if err != nil {
		return err
	}
	if _, err := awaitReady(ctx, dataDir, readFile(filepath.Join(dataDir, "logs", "slipstream.log"))); err != nil {
		return err
	}
	first, err := r.statusOf(ctx, exe, "status")
	if err != nil {
		return err
	}
	if err := first.want(k, "run"); err != nil {
		return err
	}
	if _, err := r.must(ctx, env, exe, "service", "install", "--user"); err != nil {
		return err
	}
	second, err := r.statusOf(ctx, exe, "status")
	if err != nil {
		return err
	}
	if second.PID != first.PID {
		return fmt.Errorf("a second service install --user restarted the Node: PID %d, then %d", first.PID, second.PID)
	}
	if _, err := r.must(ctx, env, exe, "service", "uninstall", "--user"); err != nil {
		return err
	}
	unit, err := install.UserUnitPath(k, layout("linux"))
	if err != nil {
		return err
	}
	if exists(unit) {
		return fmt.Errorf("service uninstall --user left %s", unit)
	}
	return nil
}

func smokeTarball(ctx context.Context, r *Run, files []string) error {
	archive, err := r.path(files[0])
	if err != nil {
		return err
	}
	env, err := r.userSession(ctx)
	if err != nil {
		return err
	}
	if err := r.removeUserService(ctx, env, install.Tarball); err != nil {
		return err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("find the home directory: %w", err)
	}
	dir := filepath.Join(home, "slipstream-smoke")
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("remove the earlier tarball: %w", err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create %s: %w", dir, err)
	}
	if _, err := r.must(ctx, nil, "tar", "-xzf", archive, "-C", dir); err != nil {
		return err
	}
	exe := filepath.Join(dir, install.Executable+"-"+r.Manifest.Version, install.Executable)
	return r.smokeUserService(ctx, env, exe, install.Tarball)
}

func smokeAppImage(ctx context.Context, r *Run, files []string) error {
	image, err := r.path(files[0])
	if err != nil {
		return err
	}
	if _, err := exec.LookPath("fusermount3"); err != nil {
		if _, err := exec.LookPath("fusermount"); err != nil {
			return errors.New("an AppImage mounts itself through FUSE: install fuse3")
		}
	}
	env, err := r.userSession(ctx)
	if err != nil {
		return err
	}
	if err := r.removeUserService(ctx, env, install.AppImage); err != nil {
		return err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("find the home directory: %w", err)
	}
	placed := filepath.Join(home, "Applications", filepath.Base(image))
	if err := copyExecutable(image, placed); err != nil {
		return err
	}
	if err := r.smokeUserServiceUnit(ctx, env, placed); err != nil {
		return err
	}
	return r.smokeBrowserOpen(ctx, placed)
}

// smokeUserServiceUnit runs the user service smoke on an AppImage, whose
// unit must start the image itself rather than its mount under /tmp.
func (r *Run) smokeUserServiceUnit(ctx context.Context, env []string, image string) error {
	if err := r.smokeUserService(ctx, env, image, install.AppImage); err != nil {
		return err
	}
	if _, err := r.must(ctx, env, image, "service", "install", "--user"); err != nil {
		return err
	}
	unitPath, err := install.UserUnitPath(install.AppImage, layout("linux"))
	if err != nil {
		return err
	}
	dataDir, err := install.AppImage.DefaultDataDir(layout("linux"))
	if err != nil {
		return err
	}
	unit, err := os.ReadFile(unitPath)
	if err != nil {
		return fmt.Errorf("read the AppImage's unit: %w", err)
	}
	if want := install.UserUnit(image, dataDir); string(unit) != want {
		return fmt.Errorf("the AppImage's unit is %q, want %q", unit, want)
	}
	_, err = r.must(ctx, env, image, "service", "uninstall", "--user")
	return err
}

// smokeBrowserOpen proves the first-run browser open on Linux: app opens
// the setup page record's URL through $BROWSER, which a stub records.
func (r *Run) smokeBrowserOpen(ctx context.Context, exe string) error {
	stubDir, err := os.MkdirTemp("", "slipstream-browser-")
	if err != nil {
		return fmt.Errorf("create the browser stub: %w", err)
	}
	defer func() { _ = os.RemoveAll(stubDir) }()
	opened := filepath.Join(stubDir, "opened")
	stub := filepath.Join(stubDir, "browser")
	if err := os.WriteFile(stub, []byte("#!/bin/sh\nprintf '%s\\n' \"$@\" >> "+strconv.Quote(opened)+"\n"), 0o755); err != nil {
		return fmt.Errorf("write the browser stub: %w", err)
	}
	dataDir, err := install.AppImage.DefaultDataDir(layout("linux"))
	if err != nil {
		return err
	}
	if err := os.RemoveAll(dataDir); err != nil {
		return fmt.Errorf("remove the earlier data dir: %w", err)
	}
	app := exec.CommandContext(ctx, exe, "app")
	app.Env = append(os.Environ(), "DISPLAY=:99", "BROWSER="+stub)
	app.Stdout, app.Stderr = r.Log, r.Log
	r.logf("+ %s app\n", exe)
	if err := app.Start(); err != nil {
		return fmt.Errorf("start %s app: %w", exe, err)
	}
	// An app that exits early fails the waits at once, naming its exit.
	running, exitedEarly := context.WithCancelCause(ctx)
	defer exitedEarly(nil)
	exited := make(chan error, 1)
	go func() {
		err := app.Wait()
		exited <- err
		cause := fmt.Errorf("%s app exited 0 before the smoke stopped it", exe)
		if err != nil {
			cause = fmt.Errorf("%s app exited: %w", exe, err)
		}
		exitedEarly(cause)
	}()
	l, err := awaitReady(running, dataDir, readFile(filepath.Join(dataDir, "logs", "slipstream.log")))
	if err == nil {
		err = awaitFile(running, opened, l.setupURL)
	}
	if err == nil && !strings.Contains(l.setupURL, "/setup#code="+l.setupCode) {
		err = fmt.Errorf("the setup page record holds %q, not the ready line's code %s", l.setupURL, l.setupCode)
	}
	if _, stopErr := r.must(ctx, nil, exe, "stop"); err == nil {
		err = stopErr
	}
	select {
	case waitErr := <-exited:
		if err == nil && waitErr != nil {
			err = fmt.Errorf("%s app after stop: %w", exe, waitErr)
		}
	case <-time.After(waitFor):
		_ = app.Process.Kill()
		if err == nil {
			err = fmt.Errorf("%s app still runs %s after stop", exe, waitFor)
		}
	}
	return err
}

func copyExecutable(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("read %s: %w", src, err)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("create %s: %w", filepath.Dir(dst), err)
	}
	if err := os.WriteFile(dst, data, 0o755); err != nil {
		return fmt.Errorf("write %s: %w", dst, err)
	}
	return nil
}

const (
	dockerContainer = "slipstream-smoke"
	dockerVolume    = "slipstream-smoke-config"
	dockerUID       = "1001"
)

func smokeDocker(ctx context.Context, r *Run, files []string) error {
	dir, err := r.path(files[0])
	if err != nil {
		return err
	}
	r.tidy(ctx, "docker", "rm", "-f", dockerContainer)
	r.tidy(ctx, "docker", "volume", "rm", dockerVolume)
	image := "slipstream-smoke:" + r.Manifest.Version
	if _, err := r.must(ctx, nil, "docker", "build", "-t", image, dir); err != nil {
		return err
	}
	if _, err := r.must(ctx, nil, "docker", "run", "-d", "--name", dockerContainer, "-e", "PUID="+dockerUID, "-e", "PGID="+dockerUID,
		"-v", dockerVolume+":/config", image); err != nil {
		return err
	}
	defer func() {
		cleanup := context.WithoutCancel(ctx)
		r.tidy(cleanup, "docker", "rm", "-f", dockerContainer)
		r.tidy(cleanup, "docker", "volume", "rm", dockerVolume)
	}()
	if _, err := awaitReady(ctx, "docker logs", func() (string, error) {
		res, err := r.exec(ctx, nil, "docker", "logs", dockerContainer)
		return res.stdout + res.stderr, err
	}); err != nil {
		return err
	}
	owner, err := r.must(ctx, nil, "docker", "exec", dockerContainer, "stat", "-c", "%u", "/config/node.key")
	if err != nil {
		return err
	}
	if strings.TrimSpace(owner) != dockerUID {
		return fmt.Errorf("/config/node.key belongs to %s, want PUID %s", strings.TrimSpace(owner), dockerUID)
	}
	hint := "docker exec -u " + dockerUID + " <container> slipstream status"
	if err := r.expect(ctx, 3, hint, "docker", "exec", dockerContainer, install.Executable, "status"); err != nil {
		return err
	}
	st, err := r.statusOf(ctx, "docker", "exec", "-u", dockerUID, dockerContainer, install.Executable, "status")
	if err != nil {
		return err
	}
	if err := st.want(install.Docker, "run"); err != nil {
		return err
	}
	if _, err := r.must(ctx, nil, "docker", "stop", dockerContainer); err != nil {
		return err
	}
	code, err := r.must(ctx, nil, "docker", "inspect", "-f", "{{.State.ExitCode}}", dockerContainer)
	if err != nil {
		return err
	}
	if strings.TrimSpace(code) != "0" {
		return fmt.Errorf("docker stop left exit code %s, want 0", strings.TrimSpace(code))
	}
	return nil
}
