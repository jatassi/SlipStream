package clients

import (
	"errors"
	"fmt"
	"strings"
)

// Folder is one Client folder: a folder as the client reports it, paired
// with the path of the same folder on the client's Node.
type Folder struct {
	Client string
	Node   string
}

// Folders are a client's Client folders. Only ParseFolders makes them.
type Folders struct{ pairs []pair }

type pair struct {
	Folder
	client path
}

var (
	ErrClientFolderNotAbsolute = errors.New("the folder as the client reports it isn't an absolute path")
	ErrNodeFolderNotAbsolute   = errors.New("the folder on the Node isn't an absolute path")
	ErrClientFolderTwice       = errors.New("another pair has the same folder as the client reports it")
)

// FolderError is why one pair can't be used, with Index its place in
// what ParseFolders was given.
type FolderError struct {
	Index int
	Err   error
}

func (e *FolderError) Error() string {
	return fmt.Sprintf("pair %d of the Client folders: %v", e.Index+1, e.Err)
}
func (e *FolderError) Unwrap() error { return e.Err }

// ParseFolders checks each pair, its sides trimmed of spaces: both sides
// absolute, POSIX or Windows, and no client folder twice.
func ParseFolders(folders []Folder) (Folders, error) {
	var f Folders
	for i, raw := range folders {
		folder := Folder{Client: strings.TrimSpace(raw.Client), Node: strings.TrimSpace(raw.Node)}
		client, ok := parsePath(folder.Client)
		if !ok {
			return Folders{}, &FolderError{Index: i, Err: ErrClientFolderNotAbsolute}
		}
		if _, ok := parsePath(folder.Node); !ok {
			return Folders{}, &FolderError{Index: i, Err: ErrNodeFolderNotAbsolute}
		}
		for _, other := range f.pairs {
			if other.client.within(client) && client.within(other.client) {
				return Folders{}, &FolderError{Index: i, Err: ErrClientFolderTwice}
			}
		}
		f.pairs = append(f.pairs, pair{Folder: folder, client: client})
	}
	return f, nil
}

func (f Folders) Pairs() []Folder {
	out := make([]Folder, len(f.pairs))
	for i, p := range f.pairs {
		out[i] = p.Folder
	}
	return out
}

// OnNode is client path p as the Node sees it, with the client folder
// that holds it in the most path segments replaced by its Node folder and
// the rest written with sep. It answers the pair it used, the zero Folder
// with no pairs, when p is used as it is. ok is false when there are pairs
// and none holds p.
func (f Folders) OnNode(p string, sep byte) (onNode string, used Folder, ok bool) {
	if len(f.pairs) == 0 {
		return p, Folder{}, true
	}
	target, ok := parsePath(p)
	if !ok {
		return "", Folder{}, false
	}
	best := -1
	for i, pr := range f.pairs {
		if target.within(pr.client) && (best < 0 || len(pr.client.names) > len(f.pairs[best].client.names)) {
			best = i
		}
	}
	if best < 0 {
		return "", Folder{}, false
	}
	pr := f.pairs[best]
	rest := target.names[len(pr.client.names):]
	if len(rest) == 0 {
		return pr.Node, pr.Folder, true
	}
	s := string(sep)
	return strings.TrimRight(pr.Node, `/\`) + s + strings.Join(rest, s), pr.Folder, true
}

// path is an absolute path split into its root and the names under it,
// with "." and ".." resolved. windows is a path with a drive letter, a
// share or a backslash, whose names compare without case.
type path struct {
	root    string
	names   []string
	windows bool
}

// parsePath reads p with / and \ both separators. Its root is "/", a
// drive such as "D:" or a share such as "//server/share".
func parsePath(p string) (path, bool) {
	out := path{windows: strings.Contains(p, `\`)}
	s := strings.ReplaceAll(p, `\`, "/")
	var rest string
	switch {
	case strings.HasPrefix(s, "//"):
		server, after, _ := strings.Cut(s[2:], "/")
		share, r, _ := strings.Cut(after, "/")
		if server == "" || share == "" {
			return path{}, false
		}
		out.root, rest, out.windows = "//"+server+"/"+share, r, true
	case len(s) >= 3 && s[1] == ':' && s[2] == '/' && isLetter(s[0]):
		out.root, rest, out.windows = strings.ToUpper(s[:2]), s[3:], true
	case strings.HasPrefix(s, "/"):
		out.root, rest = "/", s[1:]
	default:
		return path{}, false
	}
	for name := range strings.SplitSeq(rest, "/") {
		switch name {
		case "", ".":
		case "..":
			out.names = out.names[:max(len(out.names)-1, 0)]
		default:
			out.names = append(out.names, name)
		}
	}
	return out, true
}

func isLetter(b byte) bool { return b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' }

// within is whether p is folder or under it, segment by segment, without
// case when p is a Windows path.
func (p path) within(folder path) bool {
	same := func(a, b string) bool { return a == b }
	if p.windows {
		same = strings.EqualFold
	}
	if !same(p.root, folder.root) || len(folder.names) > len(p.names) {
		return false
	}
	for i, name := range folder.names {
		if !same(p.names[i], name) {
			return false
		}
	}
	return true
}
