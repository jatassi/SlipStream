package cardigann

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// HTMLSelector provides CSS selector-based extraction from HTML documents.
type HTMLSelector struct {
	doc *goquery.Document
}

// NewHTMLSelector creates a new HTML selector from raw HTML bytes.
func NewHTMLSelector(html []byte) (*HTMLSelector, error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(html))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}
	return &HTMLSelector{doc: doc}, nil
}

// NewHTMLSelectorFromString creates a new HTML selector from an HTML string.
func NewHTMLSelectorFromString(html string) (*HTMLSelector, error) {
	return NewHTMLSelector([]byte(html))
}

// Select returns the first element matching the CSS selector.
func (s *HTMLSelector) Select(selector string) *goquery.Selection {
	return s.doc.Find(selector).First()
}

// SelectAll returns all elements matching the CSS selector.
func (s *HTMLSelector) SelectAll(selector string) *goquery.Selection {
	return s.doc.Find(selector)
}

// SelectFrom returns elements matching the selector within a given selection.
func (s *HTMLSelector) SelectFrom(sel *goquery.Selection, selector string) *goquery.Selection {
	return sel.Find(selector)
}

// ExtractText extracts text content from a selection.
// If attribute is specified, extracts that attribute instead.
func ExtractText(sel *goquery.Selection, attribute string) string {
	if sel == nil || sel.Length() == 0 {
		return ""
	}

	if attribute != "" {
		val, exists := sel.Attr(attribute)
		if exists {
			return strings.TrimSpace(val)
		}
		return ""
	}

	return strings.TrimSpace(sel.Text())
}

// ExtractAttribute extracts an attribute value from a selection.
func ExtractAttribute(sel *goquery.Selection, attr string) string {
	if sel == nil || sel.Length() == 0 {
		return ""
	}
	val, _ := sel.Attr(attr)
	return strings.TrimSpace(val)
}

// RemoveElements removes elements matching the selector from the selection.
func RemoveElements(sel *goquery.Selection, removeSelector string) *goquery.Selection {
	if removeSelector == "" {
		return sel
	}
	sel.Find(removeSelector).Remove()
	return sel
}

// ExtractField extracts a value from a selection based on a Field definition.
func ExtractField(sel *goquery.Selection, field Field, ctx *TemplateContext) (string, error) {
	// Handle static text
	if field.Text != "" {
		engine := NewTemplateEngine()
		return engine.Evaluate(field.Text, ctx)
	}

	// Handle selector-based extraction
	var targetSel *goquery.Selection
	if field.Selector != "" {
		targetSel = sel.Find(field.Selector).First()
	} else {
		targetSel = sel
	}

	// Check if we found the element
	if targetSel.Length() == 0 {
		if field.Optional {
			return field.Default, nil
		}
		if field.Default != "" {
			return field.Default, nil
		}
		return "", nil
	}

	// Remove unwanted elements before extraction
	if field.Remove != "" {
		targetSel = targetSel.Clone()
		targetSel.Find(field.Remove).Remove()
	}

	// Extract the value
	var value string
	if field.Attribute != "" {
		value = ExtractAttribute(targetSel, field.Attribute)
	} else {
		value = strings.TrimSpace(targetSel.Text())
	}

	// Handle case mapping
	if len(field.Case) > 0 {
		if mapped, ok := field.Case[value]; ok {
			value = mapped
		} else if defaultVal, ok := field.Case["*"]; ok {
			value = defaultVal
		}
	}

	// Apply filters
	if len(field.Filters) > 0 {
		filtered, err := ApplyFilters(value, field.Filters)
		if err != nil {
			return "", err
		}
		value = filtered
	}

	// Use default if value is empty
	if value == "" && field.Default != "" {
		value = field.Default
	}

	return value, nil
}

// ExtractRows finds all result rows in the document.
func (s *HTMLSelector) ExtractRows(rowSelector RowSelector) []*goquery.Selection {
	var rows []*goquery.Selection

	sel := s.doc.Find(rowSelector.Selector)

	// Remove elements if specified
	if rowSelector.Remove != "" {
		sel.Find(rowSelector.Remove).Remove()
	}

	sel.Each(func(i int, row *goquery.Selection) {
		// Skip header rows
		if i < rowSelector.After {
			return
		}
		rows = append(rows, row)
	})

	return rows
}

// GetDocument returns the underlying goquery document.
func (s *HTMLSelector) GetDocument() *goquery.Document {
	return s.doc
}

// FindText finds and returns the text content of the first matching element.
func (s *HTMLSelector) FindText(selector string) string {
	return ExtractText(s.Select(selector), "")
}

// FindAttr finds and returns an attribute value of the first matching element.
func (s *HTMLSelector) FindAttr(selector, attr string) string {
	return ExtractAttribute(s.Select(selector), attr)
}

// Exists returns true if at least one element matches the selector.
func (s *HTMLSelector) Exists(selector string) bool {
	return s.doc.Find(selector).Length() > 0
}

// Count returns the number of elements matching the selector.
func (s *HTMLSelector) Count(selector string) int {
	return s.doc.Find(selector).Length()
}

// OuterHTML returns the outer HTML of the first matching element.
func (s *HTMLSelector) OuterHTML(selector string) (string, error) {
	sel := s.Select(selector)
	if sel.Length() == 0 {
		return "", nil
	}
	return goquery.OuterHtml(sel)
}

// InnerHTML returns the inner HTML of the first matching element.
func (s *HTMLSelector) InnerHTML(selector string) (string, error) {
	sel := s.Select(selector)
	if sel.Length() == 0 {
		return "", nil
	}
	return sel.Html()
}
