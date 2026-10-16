package smtp

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	netmail "net/mail"
	"strings"
)

type parts struct {
	subject string
	text    string
	html    string
}

// parse reads a message's subject and its text and HTML parts, at any
// depth of multipart nesting.
func parse(raw []byte) (parts, error) {
	m, err := netmail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		return parts{}, fmt.Errorf("unreadable message: %w", err)
	}
	var p parts
	if p.subject, err = new(mime.WordDecoder).DecodeHeader(m.Header.Get("Subject")); err != nil {
		return parts{}, fmt.Errorf("unreadable subject: %w", err)
	}
	if err := p.walk(m.Header.Get("Content-Type"), m.Header.Get("Content-Transfer-Encoding"), m.Body); err != nil {
		return parts{}, err
	}
	return p, nil
}

func (p *parts) walk(contentType, encoding string, body io.Reader) error {
	if contentType == "" {
		contentType = "text/plain"
	}
	media, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return fmt.Errorf("content type %q: %w", contentType, err)
	}
	if strings.HasPrefix(media, "multipart/") {
		r := multipart.NewReader(body, params["boundary"])
		for {
			part, err := r.NextRawPart()
			if err == io.EOF {
				return nil
			}
			if err != nil {
				return fmt.Errorf("%s part: %w", media, err)
			}
			if err := p.walk(part.Header.Get("Content-Type"), part.Header.Get("Content-Transfer-Encoding"), part); err != nil {
				return err
			}
		}
	}
	switch strings.ToLower(encoding) {
	case "quoted-printable":
		body = quotedprintable.NewReader(body)
	case "base64":
		body = base64.NewDecoder(base64.StdEncoding, body)
	}
	b, err := io.ReadAll(body)
	if err != nil {
		return fmt.Errorf("%s body: %w", media, err)
	}
	switch media {
	case "text/plain":
		p.text = string(b)
	case "text/html":
		p.html = string(b)
	}
	return nil
}
