package install

const (
	// ProductName is the app's display name, and SlipStream.app's name.
	ProductName = "SlipStream"
	// BundleID is the macOS bundle id and the LaunchAgent label. v1 used
	// com.slipstream.*, so neither collides with it.
	BundleID = "io.github.jatassi.slipstream"
	// Executable is the Node binary's base name, before any .exe.
	Executable = "slipstream"
	// Trampoline is the desktop launcher's base name, before any .exe. On
	// Windows it is the GUI-subsystem program shortcuts start; on macOS it
	// is CFBundleExecutable.
	Trampoline = "slipstream-app"
	// WindowsRegistryKey is the HKCU key v2 writes under, beside v1's.
	WindowsRegistryKey = `Software\` + DataName
	// WindowsRunValue names the HKCU Run value that starts the app at sign-in.
	WindowsRunValue = DataName
	// WindowsProgramDir is the MSI's install folder under %LOCALAPPDATA%.
	// It is fixed because firewall rules key on it.
	WindowsProgramDir = `Programs\` + ProductName
)
