package mail

import (
	"bytes"
	"embed"
	"fmt"
	htmltemplate "html/template"
	"io/fs"
	"net/url"
	"regexp"
	"strings"
	texttemplate "text/template"
)

//go:embed layout.html.tmpl layout.txt.tmpl
var layoutFiles embed.FS

// Template is one kind of email on the shared layout. Its owner's files
// are <name>.html.tmpl and <name>.txt.tmpl. The text file defines
// "subject" and "body", the HTML file "body", and either may define
// "footer". Both call the layout's blocks: {{button "Label" .URL}} and
// {{callout "text"}}.
type Template[D any] struct {
	html *htmltemplate.Template
	text *texttemplate.Template
}

// MustTemplate parses name's two files from files on the layout, and
// panics on a missing file or define.
func MustTemplate[D any](files fs.FS, name string) Template[D] {
	html := htmltemplate.Must(htmltemplate.New("").Funcs(htmlBlocks).ParseFS(layoutFiles, "layout.html.tmpl"))
	html = htmltemplate.Must(html.ParseFS(files, name+".html.tmpl"))
	text := texttemplate.Must(texttemplate.New("").Funcs(textBlocks).ParseFS(layoutFiles, "layout.txt.tmpl"))
	text = texttemplate.Must(text.ParseFS(files, name+".txt.tmpl"))
	if html.Lookup("body") == nil || text.Lookup("body") == nil || text.Lookup("subject") == nil {
		panic("mail: template " + name + " needs body in both files and subject in its text file")
	}
	return Template[D]{html: html, text: text}
}

func (t Template[D]) Render(to Address, data D) (Message, error) {
	var subject, html, text bytes.Buffer
	if err := t.text.ExecuteTemplate(&subject, "subject", data); err != nil {
		return Message{}, fmt.Errorf("mail subject: %w", err)
	}
	if err := t.html.ExecuteTemplate(&html, "layout", data); err != nil {
		return Message{}, fmt.Errorf("mail html: %w", err)
	}
	if err := t.text.ExecuteTemplate(&text, "layout", data); err != nil {
		return Message{}, fmt.Errorf("mail text: %w", err)
	}
	return Message{
		To:      to,
		Subject: strings.Join(strings.Fields(subject.String()), " "),
		HTML:    html.String(),
		Text:    strings.TrimSpace(blankRuns.ReplaceAllString(text.String(), "\n\n")) + "\n",
	}, nil
}

// blankRuns are the blank lines a template's conditionals leave, which the
// text part folds into one.
var blankRuns = regexp.MustCompile(`\n(?:[ \t]*\n)+`)

var htmlBlocks = htmltemplate.FuncMap{"button": htmlButton, "callout": htmlCallout}

var textBlocks = texttemplate.FuncMap{
	"button":  func(label, link string) string { return label + ": " + link },
	"callout": func(text string) string { return text },
}

// htmlButton is a bulletproof capsule: VML for Outlook and a styled link
// everywhere else. It writes the markup itself, since html/template drops
// the conditional comments Outlook reads.
func htmlButton(label, link string) (htmltemplate.HTML, error) {
	u, err := url.Parse(link)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") {
		return "", fmt.Errorf("button link %q is not an http or https URL", link)
	}
	l, h := htmltemplate.HTMLEscapeString(label), htmltemplate.HTMLEscapeString(u.String())
	return htmltemplate.HTML(`<table role="presentation" cellpadding="0" cellspacing="0" border="0" style="margin:8px 0 24px;"><tr><td>` + //nolint:gosec // label and link are escaped above
		`<!--[if mso]><v:roundrect xmlns:v="urn:schemas-microsoft-com:vml" xmlns:w="urn:schemas-microsoft-com:office:word" href="` + h + `" style="height:44px;v-text-anchor:middle;width:240px;" arcsize="50%" stroke="f" fillcolor="#1d1d1f"><w:anchorlock/><center style="color:#ffffff;font-family:Helvetica,Arial,sans-serif;font-size:16px;font-weight:bold;">` + l + `</center></v:roundrect><![endif]-->` +
		`<!--[if !mso]><!--><a href="` + h + `" target="_blank" style="display:inline-block;background-color:#1d1d1f;border-radius:22px;color:#ffffff;font-family:Helvetica,Arial,sans-serif;font-size:16px;font-weight:bold;line-height:44px;padding:0 28px;text-decoration:none;">` + l + `</a><!--<![endif]-->` +
		`</td></tr></table>`), nil
}

func htmlCallout(text string) htmltemplate.HTML {
	return htmltemplate.HTML(`<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="margin:0 0 16px;"><tr>` + //nolint:gosec // text is escaped
		`<td bgcolor="#fff4dc" style="background-color:#fff4dc;border-left:3px solid #e0a020;padding:12px 16px;color:#5c3d00;font-family:Helvetica,Arial,sans-serif;font-size:15px;line-height:22px;">` +
		htmltemplate.HTMLEscapeString(text) + `</td></tr></table>`)
}
