package fake

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/jatassi/slipstream/fakes/events"
)

// Capture writes one declared capture's records to the run's event log.
type Capture struct {
	spec CaptureSpec
	env  Env
}

// File is a file a capture keeps with a record, such as a mail's
// attachment.
type File struct {
	Name string
	Data []byte
}

// Capture is the writer of the capture name, which the fake's Spec must
// declare.
func (e Env) Capture(name string) (*Capture, error) {
	i := slices.IndexFunc(e.captures, func(c CaptureSpec) bool { return c.Name == name })
	if i < 0 {
		return nil, fmt.Errorf("fake %s declares no capture %s", e.Name, name)
	}
	if e.sink == nil {
		return nil, fmt.Errorf("fake %s runs with no event log for capture %s", e.Name, name)
	}
	return &Capture{spec: e.captures[i], env: e}, nil
}

// Record appends a record of the capture's kind with data, from the origin
// ctx carries. Each file lands under captures/<name>/ in the run dir, and
// the record names it.
func (c *Capture) Record(ctx context.Context, data any, files ...File) {
	if err := c.record(ctx, data, files); err != nil {
		c.env.Log.Error("capture", "capture", c.spec.Name, "err", err.Error())
	}
}

func (c *Capture) record(ctx context.Context, data any, files []File) error {
	raw, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("capture %s: %w", c.spec.Name, err)
	}
	rec := events.Record{Kind: c.spec.Kind, Node: OriginOf(ctx), Data: raw}
	if len(files) > 0 {
		dir := filepath.Join(c.env.Files, "captures", c.spec.Name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("capture %s: %w", c.spec.Name, err)
		}
		rec.Files = map[string]string{}
		for _, f := range files {
			path, err := keep(dir, f)
			if err != nil {
				return fmt.Errorf("capture %s: %w", c.spec.Name, err)
			}
			rel, err := filepath.Rel(c.env.Files, path)
			if err != nil {
				return fmt.Errorf("capture %s: %w", c.spec.Name, err)
			}
			rec.Files[f.Name] = filepath.ToSlash(rel)
		}
	}
	if _, err := c.env.sink.Append(rec); err != nil {
		return fmt.Errorf("capture %s: %w", c.spec.Name, err)
	}
	return nil
}

// keep writes f under dir with a name no other file there has.
func keep(dir string, f File) (string, error) {
	out, err := os.CreateTemp(dir, "*-"+filepath.Base(f.Name))
	if err != nil {
		return "", err //nolint:wrapcheck // the caller names the capture
	}
	_, werr := out.Write(f.Data)
	if err := errors.Join(werr, out.Close()); err != nil {
		return "", err
	}
	return out.Name(), nil
}
