package cardigann

import (
	"strings"
	"testing"

	"golang.org/x/net/html"
)

const testPage = `<html><body>
<table id="list" border=1>
<tr class="row" data-id="1"><td class="title"><a href="/t/1" class="dl">Show Seeders <span>NEW</span></a></td><td class="seeders">12</td><td><img src="/free.gif"></td></tr>
<tr class="row" data-id="2"><td class="title"><a href="/t/2">show seeders</a></td><td class="seeders"> 3 </td><td></td></tr>
</table>
<div id="content"><div class="poststuff">P1</div><div class="entry"><a class="download">D</a></div><div class="poststuff">P2</div><div class="entry">none</div></div>
<div class="footer"><a class="logout" href="/logout">Log out</a></div>
</body></html>`

func testDoc(t *testing.T) *html.Node {
	t.Helper()
	doc, err := parseHTML(testPage)
	if err != nil {
		t.Fatal(err)
	}
	return doc
}

func mustSelector(t *testing.T, css string) *selector {
	t.Helper()
	s, err := compileSelector(css)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func texts(nodes []*html.Node) []string {
	out := make([]string, len(nodes))
	for i, n := range nodes {
		out[i] = strings.TrimSpace(textContent(n))
	}
	return out
}

func TestSelectorDialect(t *testing.T) {
	doc := testDoc(t)
	tests := []struct {
		css  string
		want []string
	}{
		{`td.title a:contains("Seeders")`, []string{"Show Seeders NEW"}},
		{`td.title a:contains('seeders')`, []string{"show seeders"}},
		{`td.title a:contains(Show Seeders)`, []string{"Show Seeders NEW"}},
		{`td:contains("(x")`, []string{}},
		{`table[border=1] td.seeders`, []string{"12", "3"}},
		{`tr[data-id=2] td.seeders`, []string{"3"}},
		{`a:not(.dl)a`, []string{"show seeders", "D", "Log out"}},
		{`td.title a[href]a:not(:contains("show"))`, []string{"Show Seeders NEW"}},
		{`div#content > div.poststuff:has(~ div.entry a.download)`, []string{"P1"}},
		{`div.poststuff:has(+ div.entry a.download), div.footer`, []string{"P1", "Log out"}},
		{`tr:has(> td img)`, []string{"Show Seeders NEW12"}},
	}
	for _, tt := range tests {
		got := texts(mustSelector(t, tt.css).All(doc))
		if strings.Join(got, "|") != strings.Join(tt.want, "|") {
			t.Errorf("%s: got %q, want %q", tt.css, got, tt.want)
		}
	}
}

func TestSelectorScope(t *testing.T) {
	doc := testDoc(t)
	row := mustSelector(t, "tr.row").First(doc)
	got := texts(mustSelector(t, ":scope > td.seeders").All(row))
	if len(got) != 1 || got[0] != "12" {
		t.Errorf(":scope > td.seeders = %q", got)
	}
	if !mustSelector(t, "tr.row").Match(row) {
		t.Error("the row doesn't match its own selector")
	}
	if len(row.Attr) != 2 {
		t.Errorf("the row kept a marker attribute: %v", row.Attr)
	}
}

func TestSelectorErrors(t *testing.T) {
	for _, css := range []string{`td[src$=\"/x\"]`, `td:contains("x`, `td:contains(x`, `div:has(~)`, `a >`} {
		if _, err := compileSelector(css); err == nil {
			t.Errorf("compileSelector(%q): want an error", css)
		}
	}
}

func TestQuerySelectorRoot(t *testing.T) {
	doc := testDoc(t)
	row := mustSelector(t, "tr.row").First(doc)
	n, err := querySelector(row, ":root div.footer a")
	if err != nil || n == nil || textContent(n) != "Log out" {
		t.Errorf(":root div.footer a = %v, %v", n, err)
	}
	if n, _ := querySelector(row, "div.footer a"); n != nil {
		t.Error("div.footer a found outside the row")
	}
}

func block(t *testing.T, sel, attribute string) *selectorBlock {
	t.Helper()
	b := &selectorBlock{Attribute: attribute}
	if sel != "" {
		tmpl, err := parseTemplate(sel)
		if err != nil {
			t.Fatal(err)
		}
		b.Selector = tmpl
	}
	return b
}

func TestSelectHTML(t *testing.T) {
	e := testEnv(t)
	row := func() *html.Node { return mustSelector(t, "tr.row").First(testDoc(t)) }
	tests := []struct {
		name string
		b    *selectorBlock
		want string
	}{
		{"text", block(t, "td.seeders", ""), "12"},
		{"attribute", block(t, "td.title a", "href"), "/t/1"},
		{"row itself", block(t, "tr.row", "data-id"), "1"},
		{"templated", block(t, "tr[data-id={{ .Config.x }}]", "data-id"), "1"},
		{"no selector reads the row", block(t, "", "class"), "row"},
	}
	for _, tt := range tests {
		got, err := tt.b.Select(row(), e, true)
		if err != nil || got.String() != tt.want {
			t.Errorf("%s: got %q, %v; want %q", tt.name, got.String(), err, tt.want)
		}
	}

	if _, err := block(t, "td.missing", "").Select(row(), e, true); err == nil {
		t.Error("a required selector that matches nothing: want an error")
	}
	if v, err := block(t, "td.missing", "").Select(row(), e, false); err != nil || !v.IsNull() {
		t.Errorf("an optional selector that matches nothing = %#v, %v", v, err)
	}
	if v, err := block(t, "td.title a", "title").Select(row(), e, false); err != nil || !v.IsNull() {
		t.Errorf("an optional missing attribute = %#v, %v", v, err)
	}

	text, _ := parseTemplate("{{ .Config.x }}")
	b := &selectorBlock{Text: text, Filters: filterBlocks{mustFilter(t, "append", arg("!"))}}
	if v, err := b.Select(row(), e, true); err != nil || v.String() != "1!" {
		t.Errorf("text = %q, %v", v.String(), err)
	}
}

func mustFilter(t *testing.T, name string, a filterArgs) filterBlock {
	t.Helper()
	f, err := compileFilter(name, a)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func TestSelectRemoveAndCase(t *testing.T) {
	e := testEnv(t)
	r := mustSelector(t, "tr.row").First(testDoc(t))
	b := block(t, "td.title a", "")
	b.Remove = "span"
	if v, err := b.Select(r, e, true); err != nil || v.String() != "Show Seeders" {
		t.Errorf("remove = %q, %v", v.String(), err)
	}
	if v, _ := block(t, "td.title a", "").Select(r, e, true); v.String() != "Show Seeders" {
		t.Errorf("remove didn't change the shared document: %q", v.String())
	}

	one, _ := parseTemplate("1")
	zero, _ := parseTemplate("0")
	c := block(t, "", "")
	c.Case = keyedList[*template]{{Key: `img[src="/free.gif"]`, Value: one}, {Key: "*", Value: zero}}
	if v, err := c.Select(r, e, true); err != nil || v.String() != "1" {
		t.Errorf("case on the first row = %q, %v", v.String(), err)
	}
	second := mustSelector(t, "tr.row").All(testDoc(t))[1]
	if v, err := c.Select(second, e, true); err != nil || v.String() != "0" {
		t.Errorf("case on the second row = %q, %v", v.String(), err)
	}
}

func TestSelectorFieldMatch(t *testing.T) {
	doc := testDoc(t)
	sel, _ := parseTemplate(":root a.logout")
	f := &selectorField{Selector: sel, Attribute: "href"}
	v, ok, err := f.Match(doc, testEnv(t))
	if err != nil || !ok || v.String() != "/logout" {
		t.Errorf("Match = %q, %v, %v", v.String(), ok, err)
	}
	sel, _ = parseTemplate("a.none")
	if _, ok, err := (&selectorField{Selector: sel}).Match(doc, testEnv(t)); ok || err != nil {
		t.Errorf("a selector that matches nothing = %v, %v", ok, err)
	}
	sel, _ = parseTemplate("a.logout")
	if _, _, err := (&selectorField{Selector: sel, Attribute: "title"}).Match(doc, testEnv(t)); err == nil {
		t.Error("a missing attribute: want an error")
	}
}

func TestParseXML(t *testing.T) {
	doc, err := parseXML(`<?xml version="1.0" encoding="windows-1251"?>
<rss xmlns:torznab="http://torznab.com/schemas/2015/feed"><Channel>
<Item><Title>A &amp; B</Title><link>https://x/1</link><Enclosure URL="https://x/1.torrent" length="5"/>
<torznab:attr name="seeders" value="7"/><description><![CDATA[<b>bold</b>]]></description></Item>
</Channel></rss>`)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct{ css, attribute, want string }{
		{"item > title", "", "A & B"},
		{"item link", "", "https://x/1"},
		{"item enclosure", "url", "https://x/1.torrent"},
		{`item torznab\:attr[name="seeders"]`, "value", "7"},
		{"item description", "", "<b>bold</b>"},
	}
	for _, tt := range tests {
		n := mustSelector(t, tt.css).First(doc)
		if n == nil {
			t.Errorf("%s matches nothing", tt.css)
			continue
		}
		got := textContent(n)
		if tt.attribute != "" {
			got, _ = attr(n, tt.attribute)
		}
		if got != tt.want {
			t.Errorf("%s = %q, want %q", tt.css, got, tt.want)
		}
	}
	if _, err := parseXML("<a><b></a"); err == nil {
		t.Error("truncated XML: want an error")
	}
}
