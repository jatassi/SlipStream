package dom

import (
	"slices"
	"strings"
	"testing"
)

const sitePage = `<html><body>
<table align=center><tr><td>header</td></tr></table><br>
<table><tbody>
<tr class="head"><th>Cat</th><th>Name</th></tr>
<tr class="row"><td><a href="browse.php?cat=4"><img src="/cat.png"></a></td><td><a href="details.php?id=42">Show.S01 <b>NEW</b></a> 1.37 GB<br>more</td><td><font color="green">FL</font></td><td class="peers">12 / </td></tr>
<tr class="row"><td><a href="browse.php?cat=5"></a></td><td><a href="details.php?id=43">Film</a></td><td><font color="green">F</font></td><td class="peers"> 3 / </td></tr>
</tbody></table>
<span>Freeleech until: <span class="datetime" data-timestamp="1762353000">soon</span></span>
<!-- a comment -->
</body></html>`

func siteRows(t *testing.T) (doc *Node, rows []*Node) {
	t.Helper()
	doc = Parse(sitePage)
	rows = doc.QuerySelectorAll("table[align=center] + br + table > tbody > tr.row")
	if len(rows) != 2 {
		t.Fatalf("BitHDTV's row selector found %d rows, want 2", len(rows))
	}
	return doc, rows
}

func TestNodeTree(t *testing.T) {
	_, rows := siteRows(t)
	row := rows[0]
	if got := row.Children()[1].QuerySelector("a").TextContent(); got != "Show.S01 NEW" {
		t.Errorf("Children()[1] a = %q", got)
	}
	if got := row.FirstElementChild().FirstElementChild().GetAttribute("HREF"); got != "browse.php?cat=4" {
		t.Errorf("FirstElementChild().FirstElementChild() href = %q", got)
	}
	if got := len(row.Children()); got != 4 {
		t.Errorf("len(Children()) = %d, want 4", got)
	}
	peers := row.QuerySelectorAll(".peers")[0].FirstChild()
	if peers.NodeName() != "#text" || peers.NodeType() != Text || strings.TrimSpace(peers.NodeValue()) != "12 /" {
		t.Errorf("BakaBT's peers text node = %q %v %q", peers.NodeName(), peers.NodeType(), peers.NodeValue())
	}
	title := row.QuerySelector(`a[href^="details.php?id="]`)
	if got := title.FirstChild().TextContent(); got != "Show.S01 " {
		t.Errorf("TorrentBytes' FirstChild().TextContent() = %q", got)
	}
	if got := title.ParentElement().NodeName(); got != "TD" {
		t.Errorf("ParentElement().NodeName() = %q", got)
	}
	if got := strings.TrimSpace(strings.Split(title.ParentElement().InnerHTML(), "<br")[0]); !strings.HasSuffix(got, "</a> 1.37 GB") {
		t.Errorf("RevolutionTT's InnerHTML = %q", got)
	}
}

func TestNodeAttributes(t *testing.T) {
	_, rows := siteRows(t)
	row := rows[0]
	if !row.Matches("tr.row:has(a[href*=\"details.php?id=\"])") || row.Matches("tr.head") {
		t.Error("Matches disagrees with the row's selector")
	}
	attrs := row.QuerySelector("img").Attributes()
	if len(attrs) != 1 || attrs[0] != (Attribute{Name: "src", Value: "/cat.png"}) {
		t.Errorf("Attributes() = %v", attrs)
	}
	if row.QuerySelector("img").HasAttribute("alt") || !row.QuerySelector("img").HasAttribute("src") {
		t.Error("HasAttribute is wrong")
	}
}

func TestSiteSelectors(t *testing.T) {
	doc, rows := siteRows(t)
	var fl []bool
	for _, r := range rows {
		fl = append(fl, r.QuerySelector(`font[color="green"]:contains("F"):contains("L")`) != nil)
	}
	if !slices.Equal(fl, []bool{true, false}) {
		t.Errorf("TorrentBytes' freeleech selector = %v, want [true false]", fl)
	}
	if doc.QuerySelector(`span:contains("Freeleech until:"):has(span.datetime)`) == nil {
		t.Error("Shazbat's freeleech selector found nothing")
	}
	if doc.QuerySelector(`span:contains("freeleech until:")`) != nil {
		t.Error(":contains ignored case")
	}
}

func TestChildNodesAndRemove(t *testing.T) {
	doc, rows := siteRows(t)
	var kinds []string
	for _, c := range doc.QuerySelector("body").ChildNodes() {
		kinds = append(kinds, c.NodeName())
	}
	if !slices.Contains(kinds, "#comment") || kinds[1] != "TABLE" {
		t.Errorf("ChildNodes() = %v", kinds)
	}
	if got := doc.QuerySelector("body").LastChild().NodeName(); got != "#text" {
		t.Errorf("LastChild() = %q", got)
	}
	rows[1].Remove()
	if got := len(doc.QuerySelectorAll("tr.row")); got != 1 {
		t.Errorf("after Remove, %d rows", got)
	}
}

func TestNilNode(t *testing.T) {
	missing := Parse(sitePage).QuerySelector("div.none")
	if missing != nil {
		t.Fatal("QuerySelector found div.none")
	}
	if missing.TextContent() != "" || missing.GetAttribute("href") != "" || missing.QuerySelector("a") != nil ||
		missing.QuerySelectorAll("a") != nil || missing.FirstChild() != nil || missing.Children() != nil ||
		missing.ParentElement() != nil || missing.Matches("a") || missing.InnerHTML() != "" || missing.NodeName() != "" {
		t.Error("a nil Node gave a value")
	}
}

func TestBadSelectorPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("QuerySelector with a bad selector didn't panic")
		}
	}()
	Parse(sitePage).QuerySelector("a >")
}

func TestDocumentText(t *testing.T) {
	doc := Parse("<p>x</p>")
	if doc.TextContent() != "" || doc.NodeType() != Document || doc.NodeName() != "#document" {
		t.Errorf("document: %q %v %q", doc.TextContent(), doc.NodeType(), doc.NodeName())
	}
	if got := doc.QuerySelector("p").TextContent(); got != "x" {
		t.Errorf("p = %q", got)
	}
}
