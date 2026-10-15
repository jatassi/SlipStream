package pack

import (
	"bytes"
	"embed"
	"encoding/xml"
	"strings"
	"text/template"

	"github.com/jatassi/slipstream/internal/install"
)

//go:embed templates
var templateFS embed.FS

var templates = template.Must(template.New("").Funcs(template.FuncMap{"xml": xmlEscape}).ParseFS(templateFS, "templates/*"))

func xmlEscape(s string) string {
	var b strings.Builder
	_ = xml.EscapeText(&b, []byte(s))
	return b.String()
}

// names are the install names every template may use.
type names struct {
	Product    string
	Executable string
	Trampoline string
	BundleID   string
	User       string
}

var installNames = names{
	Product:    install.ProductName,
	Executable: install.Executable,
	Trampoline: install.Trampoline,
	BundleID:   install.BundleID,
	User:       install.SystemUser,
}

func render(name string, data any) string {
	var b bytes.Buffer
	if err := templates.ExecuteTemplate(&b, name, data); err != nil {
		panic("pack: render " + name + ": " + err.Error())
	}
	return b.String()
}
