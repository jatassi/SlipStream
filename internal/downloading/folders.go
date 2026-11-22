package downloading

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
)

var unsafeInName = strings.NewReplacer(`\`, "_", "/", "_", ":", "_", "*", "_", "?", "_", `"`, "_", "<", "_", ">", "_", "|", "_")

// folderName is a Content Type's name as a folder name on every OS.
func folderName(name string) string { return unsafeInName.Replace(name) }

// builtinFolder is the folder the built-in client keeps torrent h of
// Content Type ct in, under its downloads folder dir.
func builtinFolder(dir, ct string, h bencode.InfoHash) string {
	return filepath.Join(dir, folderName(ct), h.String())
}

// errNoClient is a Download Client this Node runs no actor for, or whose
// download folder its actor doesn't know yet.
var errNoClient = errors.New("this Node doesn't know the Download Client's folder yet")

// clientBase is the download folder of a Download Client on this Node:
// the built-in client's downloads folder when client is nil, else the
// external client's as this Node sees it.
func (n *Node) clientBase(client *[16]byte) (string, error) {
	if client == nil {
		return n.builtin.Dir(), nil
	}
	id, err := storage.UUIDFromBytes(client[:])
	if err != nil {
		return "", err
	}
	a := n.actor(id)
	if a == nil {
		return "", errNoClient
	}
	f, ok := a.Folder()
	if !ok {
		return "", errNoClient
	}
	return f, nil
}

// StageDownload is the folder a Transfer stages in for the torrent's
// Download Client: a staging folder in the client's download folder, so
// the payload's place into the client's folder stays on one volume.
func (n *Node) StageDownload(_ context.Context, client *[16]byte, _ string, _ bencode.InfoHash, transfer storage.UUID) (string, error) {
	base, err := n.clientBase(client)
	if err != nil {
		return "", err
	}
	folder := filepath.Join(base, storing.StagingDir, transfer.String())
	if err := os.MkdirAll(folder, 0o750); err != nil {
		return "", fmt.Errorf("make the staging folder: %w", err)
	}
	return folder, nil
}

// Arriving is where a payload file a Move placed in a Download Client's
// folder lies on this Node: under the built-in client's folder for the
// torrent, or under the external client's Content Type subfolder.
func (n *Node) Arriving(a storing.Arriving) (string, error) {
	base, err := n.clientBase(a.Client)
	if err != nil {
		return "", err
	}
	folder := filepath.Join(base, folderName(a.ContentType))
	if a.Client == nil {
		folder = builtinFolder(base, a.ContentType, a.InfoHash)
	}
	return filepath.Join(folder, filepath.FromSlash(a.Path.String())), nil
}
