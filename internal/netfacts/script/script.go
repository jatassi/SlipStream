// Package script is the scripted readers' file: what a Node under the Fake
// world's scripted-network divergence reads as its interfaces, egress,
// listeners' scope and firewall in place of the OS, and the Controller's
// table of where each Node link appears to come from. fake-gateway writes
// it, and the Node reads it again whenever it changes.
package script

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"os"
	"path/filepath"
)

type File struct {
	Interfaces []Interface `json:"interfaces"`
	// Egress is the name of the interface traffic to the internet leaves
	// by.
	Egress string `json:"egress"`
	// Gateway is the egress route's gateway, the zero address when the
	// route has none.
	Gateway netip.Addr `json:"gateway"`
	// Bind is what SlipStream's own web listener accepts.
	Bind Bind `json:"bind"`
	// Firewall is the host firewall's answer per port and family. A port
	// and family it doesn't list reads unknown.
	Firewall []Rule `json:"firewall,omitempty"`
	// Links are the addresses each other Node's link appears to come from,
	// in the Controller's file only.
	Links []Link `json:"links,omitempty"`
}

type Interface struct {
	Name      string    `json:"name"`
	Tunnel    bool      `json:"tunnel,omitempty"`
	Addresses []Address `json:"addresses"`
}

type Address struct {
	Prefix     netip.Prefix `json:"prefix"`
	Temporary  bool         `json:"temporary,omitempty"`
	Deprecated bool         `json:"deprecated,omitempty"`
}

type Bind string

const (
	// BindAll accepts connections on every interface.
	BindAll Bind = "all"
	// BindLocal accepts connections from this machine only.
	BindLocal Bind = "local"
)

type Family string

const (
	IPv4 Family = "ipv4"
	IPv6 Family = "ipv6"
)

type Policy string

const (
	Allows  Policy = "allows"
	Blocks  Policy = "blocks"
	Unknown Policy = "unknown"
	// Docker is a port Docker publishes, which the firewall reader can't
	// judge.
	Docker Policy = "docker"
)

// Rule is the firewall's answer for one port and family. Web stands for
// SlipStream's own web listener, whichever port it bound, since the Fake
// world's ports are drawn at launch; Port is zero then.
type Rule struct {
	Port   uint16 `json:"port,omitempty"`
	Web    bool   `json:"web,omitempty"`
	Family Family `json:"family"`
	Policy Policy `json:"policy"`
}

type Link struct {
	Node    string     `json:"node"`
	Address netip.Addr `json:"address"`
}

// Read reads the file at path strictly.
func Read(path string) (File, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return File{}, fmt.Errorf("network script: %w", err)
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var f File
	if err := dec.Decode(&f); err != nil {
		return File{}, fmt.Errorf("network script %s: %w", path, err)
	}
	return f, nil
}

// Write replaces the file at path with f in one rename, so a reader sees
// the old file or the new one, never a part.
func Write(path string, f File) error {
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return fmt.Errorf("network script: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*")
	if err != nil {
		return fmt.Errorf("network script: %w", err)
	}
	_, werr := tmp.Write(append(data, '\n'))
	cerr := tmp.Close()
	if werr != nil || cerr != nil {
		_ = os.Remove(tmp.Name())
		return fmt.Errorf("network script %s: %w", path, errors.Join(werr, cerr))
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		_ = os.Remove(tmp.Name())
		return fmt.Errorf("network script: %w", err)
	}
	return nil
}
