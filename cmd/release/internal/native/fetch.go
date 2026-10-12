package native

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"slices"
)

// Fetch makes the layer present in the cache and returns its directory. It
// downloads the asset beside the final directory, checks its SHA-256 against
// the lock, extracts it and renames it into place, so the cache never holds
// a layer that is partial or does not match the lock. A present layer costs
// no request.
func Fetch(ctx context.Context, cacheRoot string, l Layer) (string, error) {
	dir := LayerDir(cacheRoot, l)
	if _, err := os.Stat(dir); err == nil {
		return dir, nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(dir), 0o755); err != nil {
		return "", err
	}
	tmp, err := os.MkdirTemp(filepath.Dir(dir), ".fetching-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmp)

	archive := filepath.Join(tmp, l.Asset)
	f, err := os.Create(archive)
	if err != nil {
		return "", err
	}
	sum, err := download(ctx, l.URL, f)
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return "", err
	}
	if sum != l.SHA256 {
		return "", fmt.Errorf("%s has SHA-256 %s, %s pins %s", l.URL, sum, LockPath, l.SHA256)
	}
	extracted := filepath.Join(tmp, "layer")
	if err := extract(archive, extracted); err != nil {
		return "", err
	}
	if err := os.Rename(extracted, dir); err != nil {
		// A concurrent fetch of the same layer renamed its copy first.
		if _, statErr := os.Stat(dir); statErr == nil {
			return dir, nil
		}
		return "", err
	}
	return dir, nil
}

// Pin hashes the published asset of each layer under base.
func Pin(ctx context.Context, base string, ids []LayerID) (Lock, error) {
	lock := Lock{Layers: []Layer{}}
	for _, id := range ids {
		url := id.URL(base)
		sum, err := download(ctx, url, io.Discard)
		if err != nil {
			return Lock{}, err
		}
		lock.Layers = append(lock.Layers, Layer{id, id.AssetName(), url, sum})
	}
	return lock, nil
}

// noRedirects stops at GitHub's redirect for a release asset. The signed
// URL it redirects to accepts only GET.
var noRedirects = &http.Client{
	CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
}

func Published(ctx context.Context, base string, id LayerID) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, id.URL(base), nil)
	if err != nil {
		return false, err
	}
	resp, err := noRedirects.Do(req)
	if err != nil {
		return false, err
	}
	resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK, http.StatusFound:
		return true, nil
	case http.StatusNotFound:
		return false, nil
	}
	return false, fmt.Errorf("HEAD %s: %s", req.URL, resp.Status)
}

func download(ctx context.Context, url string, w io.Writer) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusNotFound:
		return "", fmt.Errorf("%s is not published; %s", url, PublishHint)
	default:
		return "", fmt.Errorf("GET %s: %s", url, resp.Status)
	}
	h := sha256.New()
	if _, err := io.Copy(io.MultiWriter(w, h), resp.Body); err != nil {
		return "", fmt.Errorf("GET %s: %w", url, err)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// Bundle is a target's fetched layers as the lock pins them for the current
// inputs and recipe: the libs layer, whose digest names the whole bundle,
// and the environment that builds against it with the toolchain layer.
type Bundle struct {
	Libs Layer
	Env  []EnvVar
}

func FetchedBundle(root, cacheRoot string, in Inputs, t Target) (Bundle, error) {
	if !slices.Contains(t.Layers, KindLibs) {
		return Bundle{}, fmt.Errorf("target %s lists no %s layer in %s", t.Name, KindLibs, InputsPath)
	}
	lock, err := LoadLock(filepath.Join(root, LockPath))
	if err != nil {
		return Bundle{}, err
	}
	recipe := os.DirFS(filepath.Join(root, RecipeDir))
	fetched := func(k Kind) (Layer, string, error) {
		digest, err := layerDigests[k](in, t, recipe)
		if err != nil {
			return Layer{}, "", err
		}
		layer, err := lock.Pinned(LayerID{k, t.Name, digest})
		if err != nil {
			return Layer{}, "", err
		}
		dir := LayerDir(cacheRoot, layer)
		if _, err := os.Stat(dir); errors.Is(err, fs.ErrNotExist) {
			return Layer{}, "", fmt.Errorf("the %s layer for %s is not fetched: run scripts/native-deps fetch -target %s", k, t.Name, t.Name)
		}
		return layer, dir, nil
	}
	_, toolchainDir, err := fetched(KindToolchain)
	if err != nil {
		return Bundle{}, err
	}
	libs, libsDir, err := fetched(KindLibs)
	if err != nil {
		return Bundle{}, err
	}
	cc, err := ReadCompilers(toolchainDir)
	if err != nil {
		return Bundle{}, err
	}
	md, err := ReadMetadata(libsDir)
	if err != nil {
		return Bundle{}, err
	}
	return Bundle{libs, CgoEnv(libsDir, md, cc, t.GoTags)}, nil
}
