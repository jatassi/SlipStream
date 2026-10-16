package rules

import (
	"path"
	"strconv"
	"strings"

	"github.com/jatassi/slipstream/cmd/release/internal/lint"
)

func init() {
	lint.Register(lint.Rule{
		Name:  "webauthn-confined",
		Doc:   "go-webauthn is imported only by internal/identity/passkey (spec #137, Architecture checks)",
		Check: checkWebauthnConfined,
	})
}

const (
	passkeyPackage = "internal/identity/passkey"
	goWebauthn     = "github.com/go-webauthn/webauthn"
)

func checkWebauthnConfined(repo *lint.Repo) ([]lint.Finding, error) {
	files, err := repo.GoSyntax()
	if err != nil {
		return nil, err
	}
	var findings []lint.Finding
	for _, f := range files {
		if path.Dir(f.Path) == passkeyPackage {
			continue
		}
		for _, imp := range f.Syntax.Imports {
			if p, _ := strconv.Unquote(imp.Path.Value); p == goWebauthn || strings.HasPrefix(p, goWebauthn+"/") {
				findings = append(findings, lint.Finding{
					Path:    f.Path,
					Line:    repo.Fset.Position(imp.Pos()).Line,
					Message: "go-webauthn imported outside internal/identity/passkey; run ceremonies through passkey.Ceremonies",
				})
			}
		}
	}
	return findings, nil
}
