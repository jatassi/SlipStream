package cardigann

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strings"

	"golang.org/x/net/html"
)

// parseXML parses an XML response into the same node tree as HTML, so
// selectors run over both. An HTML parser would empty RSS's <link>, so XML
// gets its own tree. Element and attribute names are lowercased, as the
// selector engine compares them in lowercase.
func parseXML(body string) (*html.Node, error) {
	dec := xml.NewDecoder(strings.NewReader(body))
	dec.Strict = false
	dec.CharsetReader = func(_ string, r io.Reader) (io.Reader, error) { return r, nil }
	doc := &html.Node{Type: html.DocumentNode}
	cur := doc
	for {
		tok, err := dec.RawToken()
		if errors.Is(err, io.EOF) {
			return doc, nil
		}
		if err != nil {
			return nil, fmt.Errorf("parse the XML response: %w", err)
		}
		cur = addXMLToken(cur, tok)
	}
}

func addXMLToken(cur *html.Node, tok xml.Token) *html.Node {
	switch t := tok.(type) {
	case xml.StartElement:
		el := &html.Node{Type: html.ElementNode, Data: xmlName(t.Name)}
		for _, a := range t.Attr {
			el.Attr = append(el.Attr, html.Attribute{Key: xmlName(a.Name), Val: a.Value})
		}
		cur.AppendChild(el)
		return el
	case xml.EndElement:
		name := xmlName(t.Name)
		for n := cur; n != nil && n.Type == html.ElementNode; n = n.Parent {
			if n.Data == name {
				return n.Parent
			}
		}
	case xml.CharData:
		if cur.Type == html.ElementNode {
			cur.AppendChild(&html.Node{Type: html.TextNode, Data: string(t)})
		}
	case xml.Comment:
		cur.AppendChild(&html.Node{Type: html.CommentNode, Data: string(t)})
	}
	return cur
}

func xmlName(n xml.Name) string {
	if n.Space != "" {
		return strings.ToLower(n.Space + ":" + n.Local)
	}
	return strings.ToLower(n.Local)
}
