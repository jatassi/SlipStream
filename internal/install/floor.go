package install

// The oldest glibc and kernel the bundled ffmpeg runs on (#93 Q18). The
// deb and rpm refuse an older host; a tarball or AppImage Node warns.
const (
	GlibcFloor  = "2.28"
	KernelFloor = "4.18"
)
