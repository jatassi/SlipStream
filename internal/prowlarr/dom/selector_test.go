package dom

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
	doc, err := ParseHTML(testPage)
	if err != nil {
		t.Fatal(err)
	}
	return doc
}

func mustSelector(t *testing.T, css string) *Selector {
	t.Helper()
	s, err := Compile(css)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func texts(nodes []*html.Node) []string {
	out := make([]string, len(nodes))
	for i, n := range nodes {
		out[i] = strings.TrimSpace(TextContent(n))
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
		if _, err := Compile(css); err == nil {
			t.Errorf("Compile(%q): want an error", css)
		}
	}
}
