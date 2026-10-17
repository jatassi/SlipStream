package rules

import (
	"path"
	"strconv"
	"strings"

	"github.com/jatassi/slipstream/cmd/release/internal/lint"
)

func init() {
	lint.Register(lint.Rule{
		Name:  "mail-confined",
		Doc:   "go-mail is imported only by internal/mail (spec #137, Architecture checks)",
		Check: checkMailConfined,
	})
}

const (
	mailPackage = "internal/mail"
	goMail      = "github.com/wneessen/go-mail"
)

func checkMailConfined(repo *lint.Repo) ([]lint.Finding, error) {
	files, err := repo.GoSyntax()
	if err != nil {
		return nil, err
	}
	var findings []lint.Finding
	for _, f := range files {
		if path.Dir(f.Path) == mailPackage {
			continue
		}
		for _, imp := range f.Syntax.Imports {
			if p, _ := strconv.Unquote(imp.Path.Value); p == goMail || strings.HasPrefix(p, goMail+"/") {
				findings = append(findings, lint.Finding{
					Path:    f.Path,
					Line:    repo.Fset.Position(imp.Pos()).Line,
					Message: "go-mail imported outside internal/mail; send through a mail.Sender",
				})
			}
		}
	}
	return findings, nil
}
