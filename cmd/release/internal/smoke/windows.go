package smoke

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/install"
)

const runKey = `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`

func exe(name string) string { return name + ".exe" }

func (r *Run) stopWindowsNodes(ctx context.Context) {
	r.tidy(ctx, "taskkill", "/F", "/IM", exe(install.Trampoline))
	r.tidy(ctx, "taskkill", "/F", "/IM", exe(install.Executable))
}

func (r *Run) powershell(ctx context.Context, script string) (string, error) {
	return r.must(ctx, nil, "powershell", "-NoProfile", "-NonInteractive", "-Command", script)
}

func (r *Run) launch(ctx context.Context, trampoline string) error {
	_, err := r.must(ctx, nil, trampoline)
	return err
}

func (r *Run) windowless(ctx context.Context) error {
	out, err := r.powershell(ctx, `Get-Process -Name slipstream,ffmpeg,ffprobe -ErrorAction SilentlyContinue | Where-Object { $_.MainWindowHandle -ne 0 } | ForEach-Object { $_.ProcessName }; exit 0`)
	if err != nil {
		return err
	}
	if names := strings.TrimSpace(out); names != "" {
		return fmt.Errorf("these processes show a window: %s", names)
	}
	return nil
}

func (r *Run) browserOpened(ctx context.Context, url string) error {
	script := fmt.Sprintf(`@(Get-CimInstance Win32_Process | Where-Object { $_.CommandLine -like '*%s*' }).Count`, url)
	deadline := time.Now().Add(waitFor)
	for {
		out, err := r.powershell(ctx, script)
		if err == nil && strings.TrimSpace(out) != "0" {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("no process's command line holds %s within %s", url, waitFor)
		}
		time.Sleep(time.Second)
	}
}

func (r *Run) consoleExitCodes(ctx context.Context, slipstream string) error {
	dir, err := os.MkdirTemp("", "slipstream-console-")
	if err != nil {
		return fmt.Errorf("create the console script: %w", err)
	}
	defer func() { _ = os.RemoveAll(dir) }()
	batch := filepath.Join(dir, "stop.cmd")
	if err := os.WriteFile(batch, []byte("@echo off\r\n\""+slipstream+"\" stop\r\necho errorlevel=%ERRORLEVEL%\r\n"), 0o600); err != nil {
		return fmt.Errorf("write the console script: %w", err)
	}
	out, err := r.must(ctx, nil, "cmd", "/d", "/c", batch)
	if err != nil {
		return err
	}
	if !strings.Contains(out, "errorlevel=1") {
		return fmt.Errorf("a second stop under cmd printed %q, want errorlevel=1", out)
	}
	out, err = r.powershell(ctx, fmt.Sprintf(`& '%s' stop; "lastexitcode=$LASTEXITCODE"`, slipstream))
	if err != nil {
		return err
	}
	if !strings.Contains(out, "lastexitcode=1") {
		return fmt.Errorf("a second stop under PowerShell printed %q, want lastexitcode=1", out)
	}
	return nil
}

func (r *Run) regValue(ctx context.Context, key, name string) (string, bool) {
	res, err := r.exec(ctx, nil, "reg", "query", key, "/v", name)
	if err != nil || res.code != 0 {
		return "", false
	}
	return res.stdout, true
}

func smokeMSI(ctx context.Context, r *Run, files []string) error {
	msi, err := r.path(files[0])
	if err != nil {
		return err
	}
	l := layout("windows")
	local := l.Getenv("LOCALAPPDATA")
	programDir := filepath.Join(local, filepath.FromSlash(strings.ReplaceAll(install.WindowsProgramDir, `\`, "/")))
	dataDir, err := install.MSI.DefaultDataDir(l)
	if err != nil {
		return err
	}
	r.stopWindowsNodes(ctx)
	if err := os.RemoveAll(dataDir); err != nil {
		return fmt.Errorf("remove the earlier data dir: %w", err)
	}
	if _, err := r.must(ctx, nil, "msiexec", "/i", msi, "/qn", "/norestart"); err != nil {
		return err
	}
	slipstream := filepath.Join(programDir, exe(install.Executable))
	trampoline := filepath.Join(programDir, exe(install.Trampoline))
	shortcut := filepath.Join(l.Getenv("APPDATA"), "Microsoft", "Windows", "Start Menu", "Programs", install.ProductName+".lnk")
	if err := r.checkMSIInstall(ctx, programDir, shortcut); err != nil {
		return err
	}
	if err := r.launch(ctx, trampoline); err != nil {
		return err
	}
	ready, err := awaitReady(ctx, dataDir, readFile(filepath.Join(dataDir, "logs", "slipstream.log")))
	if err != nil {
		return err
	}
	if err := r.windowless(ctx); err != nil {
		return err
	}
	st, err := r.appStatusOf(ctx, slipstream, "status")
	if err != nil {
		return err
	}
	if err := st.wantApp(install.MSI, true); err != nil {
		return err
	}
	if err := r.browserOpened(ctx, "/setup#code="+ready.setupCode); err != nil {
		return err
	}
	if err := r.launch(ctx, trampoline); err != nil {
		return fmt.Errorf("a second launch: %w", err)
	}
	again, err := r.statusOf(ctx, slipstream, "status")
	if err != nil {
		return err
	}
	if again.PID != st.PID {
		return fmt.Errorf("after a second launch the Node is PID %d, want %d", again.PID, st.PID)
	}
	if _, err := r.must(ctx, nil, slipstream, "stop"); err != nil {
		return err
	}
	if err := r.consoleExitCodes(ctx, slipstream); err != nil {
		return err
	}
	if err := r.upgradesKeepStartAtSignIn(ctx, msi, trampoline); err != nil {
		return err
	}
	if _, err := r.must(ctx, nil, "msiexec", "/x", msi, "/qn", "/norestart"); err != nil {
		return err
	}
	return r.checkMSIUninstall(ctx, dataDir, slipstream, trampoline, shortcut)
}

// upgradesKeepStartAtSignIn upgrades to a copy of the MSI under a new
// ProductCode with start at sign-in off, then back to the MSI with it on,
// and checks each upgrade kept the choice.
func (r *Run) upgradesKeepStartAtSignIn(ctx context.Context, msi, trampoline string) error {
	upgrade, err := r.upgradeCopy(ctx, msi)
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(filepath.Dir(upgrade)) }()
	if _, err := r.must(ctx, nil, "reg", "delete", runKey, "/v", install.WindowsRunValue, "/f"); err != nil {
		return err
	}
	if _, err := r.must(ctx, nil, "msiexec", "/i", upgrade, "/qn", "/norestart"); err != nil {
		return err
	}
	if run, ok := r.regValue(ctx, runKey, install.WindowsRunValue); ok {
		return fmt.Errorf("an upgrade turned start at sign-in back on: %s", run)
	}
	if err := mustExist(trampoline); err != nil {
		return fmt.Errorf("after the upgrade: %w", err)
	}
	if _, err := r.must(ctx, nil, "reg", "add", runKey, "/v", install.WindowsRunValue, "/t", "REG_SZ", "/d", `"`+trampoline+`"`, "/f"); err != nil {
		return err
	}
	if _, err := r.must(ctx, nil, "msiexec", "/i", msi, "/qn", "/norestart"); err != nil {
		return err
	}
	if run, ok := r.regValue(ctx, runKey, install.WindowsRunValue); !ok || !strings.Contains(run, `"`+trampoline+`"`) {
		return fmt.Errorf("after an upgrade with start at sign-in on, the Run value is %q, want the quoted trampoline %s", run, trampoline)
	}
	return nil
}

// upgradeCopyScript copies {msi} to {upgrade} under a new ProductCode and
// package code, which Windows Installer treats as a newer build of the same
// product. Each ~ becomes the backtick MSI SQL quotes names with, which a Go
// raw string cannot hold.
const upgradeCopyScript = `$ErrorActionPreference = 'Stop'
Copy-Item -LiteralPath '{msi}' -Destination '{upgrade}'
$installer = New-Object -ComObject WindowsInstaller.Installer
$db = $installer.GetType().InvokeMember('OpenDatabase', 'InvokeMethod', $null, $installer, @('{upgrade}', 1))
$query = 'UPDATE ~Property~ SET ~Value~ = ''' + [guid]::NewGuid().ToString('B').ToUpper() + ''' WHERE ~Property~ = ''ProductCode'''
$view = $db.GetType().InvokeMember('OpenView', 'InvokeMethod', $null, $db, @($query))
[void]$view.GetType().InvokeMember('Execute', 'InvokeMethod', $null, $view, $null)
[void]$view.GetType().InvokeMember('Close', 'InvokeMethod', $null, $view, $null)
$info = $db.GetType().InvokeMember('SummaryInformation', 'GetProperty', $null, $db, @(1))
[void]$info.GetType().InvokeMember('Property', 'SetProperty', $null, $info, @(9, [guid]::NewGuid().ToString('B').ToUpper()))
[void]$info.GetType().InvokeMember('Persist', 'InvokeMethod', $null, $info, $null)
[void]$db.GetType().InvokeMember('Commit', 'InvokeMethod', $null, $db, $null)`

func (r *Run) upgradeCopy(ctx context.Context, msi string) (string, error) {
	dir, err := os.MkdirTemp("", "slipstream-upgrade-")
	if err != nil {
		return "", fmt.Errorf("create the upgrade copy's folder: %w", err)
	}
	upgrade := filepath.Join(dir, filepath.Base(msi))
	script := strings.NewReplacer("{msi}", msi, "{upgrade}", upgrade, "~", "`").Replace(upgradeCopyScript)
	if _, err := r.powershell(ctx, script); err != nil {
		_ = os.RemoveAll(dir)
		return "", err
	}
	return upgrade, nil
}

func (r *Run) checkMSIUninstall(ctx context.Context, dataDir string, gone ...string) error {
	for _, p := range gone {
		if exists(p) {
			return fmt.Errorf("uninstall left %s", p)
		}
	}
	if run, ok := r.regValue(ctx, runKey, install.WindowsRunValue); ok {
		return fmt.Errorf("uninstall left the Run value: %s", run)
	}
	return mustExist(dataDir)
}

func (r *Run) checkMSIInstall(ctx context.Context, programDir, shortcut string) error {
	trampoline := filepath.Join(programDir, exe(install.Trampoline))
	files := []string{shortcut, filepath.Join(programDir, install.MarkerName)}
	for _, name := range []string{install.Executable, install.Trampoline, "ffmpeg", "ffprobe", "dovi_tool"} {
		files = append(files, filepath.Join(programDir, exe(name)))
	}
	if err := mustExist(files...); err != nil {
		return err
	}
	if run, ok := r.regValue(ctx, runKey, install.WindowsRunValue); !ok || !strings.Contains(run, `"`+trampoline+`"`) {
		return fmt.Errorf("the Run value %s is %q, want the quoted trampoline %s", install.WindowsRunValue, run, trampoline)
	}
	if path, ok := r.regValue(ctx, `HKCU\Environment`, "Path"); !ok || !strings.Contains(path, programDir) {
		return fmt.Errorf("the user PATH is %q, want it to hold %s", path, programDir)
	}
	return nil
}

func smokeZIP(ctx context.Context, r *Run, files []string) error {
	zip, err := r.path(files[0])
	if err != nil {
		return err
	}
	dir := filepath.Join(os.TempDir(), "ss-zip")
	r.stopWindowsNodes(ctx)
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("remove the earlier ZIP: %w", err)
	}
	r.tidy(ctx, "reg", "delete", runKey, "/v", install.WindowsRunValue, "/f")
	if _, err := r.powershell(ctx, fmt.Sprintf(`Expand-Archive -LiteralPath '%s' -DestinationPath '%s'`, zip, dir)); err != nil {
		return err
	}
	folder := filepath.Join(dir, install.ProductName)
	slipstream := filepath.Join(folder, exe(install.Executable))
	if err := r.launch(ctx, filepath.Join(folder, exe(install.Trampoline))); err != nil {
		return err
	}
	dataDir := filepath.Join(folder, "data")
	if _, err := awaitReady(ctx, dataDir, readFile(filepath.Join(dataDir, "logs", "slipstream.log"))); err != nil {
		return err
	}
	if run, ok := r.regValue(ctx, runKey, install.WindowsRunValue); ok {
		return fmt.Errorf("the ZIP turned on start at sign-in: %s", run)
	}
	st, err := r.appStatusOf(ctx, slipstream, "status")
	if err != nil {
		return err
	}
	if err := st.wantApp(install.ZIP, false); err != nil {
		return err
	}
	_, err = r.must(ctx, nil, slipstream, "stop")
	return err
}
