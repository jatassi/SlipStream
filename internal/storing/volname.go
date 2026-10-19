package storing

import "strings"

// isUNC is a Windows \\server\share path, including the \\?\UNC\ long
// form; \\?\C:\ and the \\.\ device paths are not.
func isUNC(p string) bool {
	p = strings.ReplaceAll(p, "/", `\`)
	if strings.HasPrefix(strings.ToUpper(p), `\\?\UNC\`) {
		return true
	}
	return strings.HasPrefix(p, `\\`) && !strings.HasPrefix(p, `\\?\`) && !strings.HasPrefix(p, `\\.\`)
}

// displayRoot is a Windows volume root as an Admin writes it: "C:",
// `\\nas\media` or `C:\mnt\disk`, without the long prefix or the trailing
// separator GetVolumePathNameW answers with.
func displayRoot(root string) string {
	switch {
	case strings.HasPrefix(strings.ToUpper(root), `\\?\UNC\`):
		root = `\\` + root[len(`\\?\UNC\`):]
	case strings.HasPrefix(root, `\\?\`):
		root = root[len(`\\?\`):]
	}
	return strings.TrimSuffix(root, `\`)
}

// volumeName names a Windows volume "D: (Media)", or `\\nas\media` for a
// share, whose label says less than its name.
func volumeName(root, label string) string {
	d := displayRoot(root)
	if label == "" || strings.HasPrefix(d, `\\`) {
		return d
	}
	return d + " (" + label + ")"
}
