// Ported from Prowlarr's src/NzbDrone.Core/Prowlarr.Core.csproj
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

// Package dom is AngleSharp's HTML parser and DOM as Prowlarr's sites use
// them, at the AngleSharp version Prowlarr.Core.csproj pins. A site's
// `parser.ParseDocument(html)` is dom.Parse(html), and its
// `row.QuerySelector("td")?.TextContent` is
// row.QuerySelector("td").TextContent(): every method on a nil *Node
// returns the zero value, as C#'s ?. does. A selector that doesn't compile
// panics, because it is a constant in the site's code.
package dom

import (
	"fmt"
	"strings"

	"golang.org/x/net/html"
)

// ParseHTML parses an HTML document, as AngleSharp's HtmlParser does.
func ParseHTML(body string) (*html.Node, error) {
	doc, err := html.Parse(strings.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("parse the HTML response: %w", err)
	}
	return doc, nil
}

// TextContent is the text of n's descendant text nodes, as AngleSharp's
// TextContent of an element.
func TextContent(n *html.Node) string {
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			b.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return b.String()
}

// Attr is the value of n's attribute, named in any case.
func Attr(n *html.Node, name string) (string, bool) {
	for _, a := range n.Attr {
		if strings.EqualFold(a.Key, name) {
			return a.Val, true
		}
	}
	return "", false
}

// Node is AngleSharp's INode, IElement and IHtmlDocument in one type.
type Node struct {
	n *html.Node
}

// NodeType is AngleSharp's NodeType.
type NodeType int

const (
	Element  NodeType = 1
	Text     NodeType = 3
	Comment  NodeType = 8
	Document NodeType = 9
	Doctype  NodeType = 10
)

// Parse is HtmlParser.ParseDocument. Reading a string can't fail, so
// neither does Parse.
func Parse(body string) *Node {
	doc, err := ParseHTML(body)
	if err != nil {
		panic(err)
	}
	return wrap(doc)
}

func wrap(n *html.Node) *Node {
	if n == nil {
		return nil
	}
	return &Node{n: n}
}

func wrapAll(ns []*html.Node) []*Node {
	out := make([]*Node, len(ns))
	for i, n := range ns {
		out[i] = wrap(n)
	}
	return out
}

func (x *Node) QuerySelector(css string) *Node {
	if x == nil {
		return nil
	}
	return wrap(MustCompile(css).First(x.n))
}

func (x *Node) QuerySelectorAll(css string) []*Node {
	if x == nil {
		return nil
	}
	return wrapAll(MustCompile(css).All(x.n))
}

func (x *Node) Matches(css string) bool {
	return x != nil && x.n.Type == html.ElementNode && MustCompile(css).Match(x.n)
}

// TextContent is an element's text, a text node's or comment's data, and
// for a document, where AngleSharp gives null, "".
func (x *Node) TextContent() string {
	switch {
	case x == nil:
		return ""
	case x.n.Type == html.TextNode || x.n.Type == html.CommentNode:
		return x.n.Data
	case x.n.Type == html.ElementNode:
		return TextContent(x.n)
	}
	return ""
}

// NodeValue is a text node's or comment's data, and "" where AngleSharp
// gives null.
func (x *Node) NodeValue() string {
	if x == nil || x.n.Type != html.TextNode && x.n.Type != html.CommentNode {
		return ""
	}
	return x.n.Data
}

// NodeName is an element's tag name in uppercase, "#text", "#comment",
// "#document" or a doctype's name.
func (x *Node) NodeName() string {
	if x == nil {
		return ""
	}
	switch x.n.Type {
	case html.ElementNode:
		return strings.ToUpper(x.n.Data)
	case html.TextNode:
		return "#text"
	case html.CommentNode:
		return "#comment"
	case html.DocumentNode:
		return "#document"
	case html.DoctypeNode, html.ErrorNode, html.RawNode:
	}
	return x.n.Data
}

func (x *Node) NodeType() NodeType {
	if x == nil {
		return 0
	}
	switch x.n.Type {
	case html.ElementNode:
		return Element
	case html.TextNode:
		return Text
	case html.CommentNode:
		return Comment
	case html.DocumentNode:
		return Document
	case html.DoctypeNode:
		return Doctype
	case html.ErrorNode, html.RawNode:
	}
	return 0
}

// GetAttribute is the attribute's value, named in any case, and "" where
// AngleSharp gives null.
func (x *Node) GetAttribute(name string) string {
	v, _ := x.attribute(name)
	return v
}

func (x *Node) HasAttribute(name string) bool {
	_, ok := x.attribute(name)
	return ok
}

func (x *Node) attribute(name string) (string, bool) {
	if x == nil || x.n.Type != html.ElementNode {
		return "", false
	}
	return Attr(x.n, name)
}

// Attribute is AngleSharp's IAttr.
type Attribute struct {
	Name, Value string
}

func (x *Node) Attributes() []Attribute {
	if x == nil {
		return nil
	}
	out := make([]Attribute, len(x.n.Attr))
	for i, a := range x.n.Attr {
		out[i] = Attribute{Name: a.Key, Value: a.Val}
	}
	return out
}

func (x *Node) Children() []*Node {
	var out []*Node
	for c := x.FirstElementChild(); c != nil; c = c.NextElementSibling() {
		out = append(out, c)
	}
	return out
}

// ChildNodes is every child, text and comments included.
func (x *Node) ChildNodes() []*Node {
	if x == nil {
		return nil
	}
	var out []*Node
	for c := x.n.FirstChild; c != nil; c = c.NextSibling {
		out = append(out, wrap(c))
	}
	return out
}

func (x *Node) FirstChild() *Node {
	if x == nil {
		return nil
	}
	return wrap(x.n.FirstChild)
}

func (x *Node) LastChild() *Node {
	if x == nil {
		return nil
	}
	return wrap(x.n.LastChild)
}

func (x *Node) FirstElementChild() *Node {
	if x == nil {
		return nil
	}
	return wrap(nextElement(x.n.FirstChild))
}

func (x *Node) NextElementSibling() *Node {
	if x == nil {
		return nil
	}
	return wrap(nextElement(x.n.NextSibling))
}

func nextElement(n *html.Node) *html.Node {
	for ; n != nil; n = n.NextSibling {
		if n.Type == html.ElementNode {
			return n
		}
	}
	return nil
}

func (x *Node) ParentElement() *Node {
	if x == nil || x.n.Parent == nil || x.n.Parent.Type != html.ElementNode {
		return nil
	}
	return wrap(x.n.Parent)
}

func (x *Node) InnerHTML() string {
	if x == nil {
		return ""
	}
	var b strings.Builder
	for c := x.n.FirstChild; c != nil; c = c.NextSibling {
		if err := html.Render(&b, c); err != nil {
			panic(err)
		}
	}
	return b.String()
}

func (x *Node) Remove() {
	if x != nil && x.n.Parent != nil {
		x.n.Parent.RemoveChild(x.n)
	}
}
