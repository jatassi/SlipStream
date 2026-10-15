package main

import (
	"slices"
	"testing"

	"github.com/jatassi/slipstream/cmd/release/internal/native"
	"github.com/jatassi/slipstream/cmd/release/internal/pack"
	"github.com/jatassi/slipstream/cmd/release/internal/smoke"
	"github.com/jatassi/slipstream/internal/buildinfo"
	"github.com/jatassi/slipstream/internal/install"
)

func TestEveryKindHasAPackageAndASmoke(t *testing.T) {
	in, err := native.LoadInputs("../../" + native.InputsPath)
	if err != nil {
		t.Fatal(err)
	}
	targets := pack.Targets()
	for _, k := range install.Kinds() {
		if k == install.Dev {
			if _, ok := targets[k]; ok || slices.Contains(smoke.Kinds(), k) {
				t.Errorf("dev has a recipe; a dev build is never packaged")
			}
			continue
		}
		if len(targets[k]) == 0 {
			t.Errorf("%s has no package recipe", k)
		}
		if !slices.Contains(smoke.Kinds(), k) {
			t.Errorf("%s has no smoke recipe", k)
		}
		for _, name := range targets[k] {
			target, err := in.Target(name)
			if err != nil {
				t.Errorf("%s packages on %v", k, err)
			} else if target.GOOS != k.GOOS() {
				t.Errorf("%s packages on %s, a %s target", k, name, target.GOOS)
			}
		}
	}
}

func TestStampVersion(t *testing.T) {
	linux := native.Target{Name: "linux-amd64"}
	for _, tc := range []struct {
		stamp buildinfo.Stamp
		tag   string
		want  pack.Version
		err   string
	}{
		{stamp: buildinfo.Stamp{Version: "v1.2.0", Fingerprint: "clean", Target: "linux-amd64"}, tag: "v1.2.0", want: "1.2.0"},
		{stamp: buildinfo.Stamp{Version: "v1.2.0-beta.1", Fingerprint: "clean", Target: "linux-amd64"}, tag: "v1.2.0-beta.1", want: "1.2.0-beta.1"},
		{stamp: buildinfo.Stamp{Version: "v1.2.0-3-gabc1234", Fingerprint: "e3b0", Target: "linux-amd64"}, want: "1.2.0-3-gabc1234"},
		{stamp: buildinfo.Stamp{Version: "abc1234", Fingerprint: "clean", Target: "linux-amd64"}, want: "0.0.0-gabc1234"},
		{stamp: buildinfo.Stamp{Version: "v1.2.0", Fingerprint: "e3b0", Target: "linux-amd64"}, tag: "v1.2.0",
			err: "refusing to package v1.2.0 from a build stamped v1.2.0 with fingerprint e3b0; a release takes a clean build of its tag"},
		{stamp: buildinfo.Stamp{Version: "v1.2.0-3-gabc1234", Fingerprint: "clean", Target: "linux-amd64"}, tag: "v1.2.0",
			err: "refusing to package v1.2.0 from a build stamped v1.2.0-3-gabc1234 with fingerprint clean; a release takes a clean build of its tag"},
		{stamp: buildinfo.Stamp{Version: "v1.2.0", Fingerprint: "clean", Target: "linux-arm64"},
			err: "dist holds a build for linux-arm64, and this runner packages linux-amd64"},
	} {
		got, err := stampVersion(tc.stamp, linux, tc.tag)
		if tc.err != "" {
			if err == nil || err.Error() != tc.err {
				t.Errorf("stampVersion(%+v, %q) = %q, %v; want %s", tc.stamp, tc.tag, got, err, tc.err)
			}
			continue
		}
		if err != nil || got != tc.want {
			t.Errorf("stampVersion(%+v, %q) = %q, %v; want %q", tc.stamp, tc.tag, got, err, tc.want)
		}
	}
}
