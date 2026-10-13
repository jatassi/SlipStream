package rules

import (
	"github.com/jatassi/slipstream/cmd/release/internal/lint"
	"github.com/jatassi/slipstream/cmd/release/internal/native"
)

func init() {
	lint.Register(lint.Rule{
		Name:  "native-lock",
		Doc:   "native/lock.json pins the layers the native inputs and recipe build and the tool archives they list (release deps verify, spec #126)",
		Check: checkNativeLock,
	})
}

func checkNativeLock(repo *lint.Repo) ([]lint.Finding, error) {
	stale, err := native.StaleLock(repo.Root)
	if err != nil {
		return nil, err
	}
	var findings []lint.Finding
	for _, msg := range stale {
		findings = append(findings, lint.Finding{Path: native.LockPath, Line: 1, Message: msg})
	}
	return findings, nil
}
