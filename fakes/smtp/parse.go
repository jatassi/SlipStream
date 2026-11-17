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
	"net/textproto"
	"strings"
)

type parts struct {
	subject string
	text    string
	html    string
	inline  []inline
}

// inline is an image part a message shows through its Content-ID.
type inline struct {
	contentID string
	media     string
	body      []byte
}

func parse(raw []byte) (parts, error) {
	m, err := netmail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		return parts{}, fmt.Errorf("unreadable message: %w", err)
	}
	var p parts
	if p.subject, err = new(mime.WordDecoder).DecodeHeader(m.Header.Get("Subject")); err != nil {
		return parts{}, fmt.Errorf("unreadable subject: %w", err)
	}
	if err := p.walk(textproto.MIMEHeader(m.Header), m.Body); err != nil {
		return parts{}, err
	}
	return p, nil
}

func (p *parts) walk(h textproto.MIMEHeader, body io.Reader) error {
	contentType, encoding := h.Get("Content-Type"), h.Get("Content-Transfer-Encoding")
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
			if err := p.walk(part.Header, part); err != nil {
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
	switch {
	case media == "text/plain":
		p.text = string(b)
	case media == "text/html":
		p.html = string(b)
	case strings.HasPrefix(media, "image/"):
		p.inline = append(p.inline, inline{contentID: strings.Trim(h.Get("Content-ID"), "<>"), media: media, body: b})
	}
	return nil
}
