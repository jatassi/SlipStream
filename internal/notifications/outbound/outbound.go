// Package outbound writes what a provider sends as canonical text, so a
// provider's golden files and notify-sink's captures compare as equals.
package outbound

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/jatassi/slipstream/internal/notifications/providers"
)

// masked stands for a secret value.
const masked = "[set]"

// Dump is r as canonical text: the method and the URL's path and query,
// since the host differs between a provider and notify-sink, then the
// headers sorted with any RFC 2047 value decoded, then the body decoded.
// JSON is indented, a form is one field per line in order, and a
// multipart part is its name, then its text, or its file name, type, size
// and SHA-256. Every secret value is
// replaced by [set], and so is an Authorization header's credential.
func Dump(r providers.Request, secrets []string) string {
	var b strings.Builder
	u, err := url.Parse(r.URL)
	if err != nil {
		fmt.Fprintf(&b, "%s %s\n", r.Method, masked)
	} else {
		fmt.Fprintf(&b, "%s %s\n", r.Method, u.RequestURI())
	}
	names := make([]string, 0, len(r.Header))
	for name := range r.Header {
		names = append(names, http.CanonicalHeaderKey(name))
	}
	slices.Sort(names)
	for _, name := range names {
		value := strings.Join(r.Header.Values(name), ", ")
		switch name {
		case "Authorization":
			value = authorization(value)
		case "Content-Type":
			value = mediaType(value)
		default:
			if decoded, err := new(mime.WordDecoder).DecodeHeader(value); err == nil {
				value = decoded
			}
		}
		fmt.Fprintf(&b, "%s: %s\n", name, value)
	}
	body(&b, r.Header.Get("Content-Type"), r.Body)
	return mask(b.String(), secrets)
}

// DumpOutbound is o as canonical text: a Post as Dump writes it, a Script's
// command, event and event file, or each Mail's address, subject, parts
// and inline images.
func DumpOutbound(o providers.Outbound, secrets []string) string {
	switch {
	case o.Post != nil:
		return Dump(*o.Post, secrets)
	case o.Run != nil:
		var b strings.Builder
		fmt.Fprintf(&b, "RUN %s\n", strings.Join(append([]string{o.Run.Path}, o.Run.Args...), " "))
		fmt.Fprintf(&b, "Event: %s\nStop: %s\n", o.Run.Event, o.Run.Stop)
		body(&b, "application/json", o.Run.EventFile)
		return mask(b.String(), secrets)
	}
	var b strings.Builder
	for i, m := range o.Mail {
		if i > 0 {
			b.WriteString("\n")
		}
		fmt.Fprintf(&b, "MAIL %s\nSubject: %s\n--- text\n%s\n--- html\n%s\n", m.To, m.Subject, m.Text, m.HTML)
		for _, p := range m.Inline {
			fmt.Fprintf(&b, "--- inline %s %s %s\n%s\n", p.ContentID, p.Name, p.Type, digest(p.Body))
		}
	}
	return mask(b.String(), secrets)
}

func body(b *strings.Builder, contentType string, raw []byte) {
	if len(raw) == 0 {
		return
	}
	b.WriteString("\n")
	kind, params, _ := mime.ParseMediaType(contentType)
	switch {
	case kind == "application/json":
		b.WriteString(indent(raw))
	case kind == "application/x-www-form-urlencoded":
		form(b, raw)
	case kind == "multipart/form-data":
		parts(b, raw, params["boundary"])
	case strings.HasPrefix(kind, "text/"):
		b.Write(raw)
		b.WriteString("\n")
	default:
		b.WriteString(digest(raw) + "\n")
	}
}

func indent(raw []byte) string {
	var out bytes.Buffer
	if err := json.Indent(&out, raw, "", "  "); err != nil {
		return "unreadable JSON: " + digest(raw) + "\n"
	}
	return out.String() + "\n"
}

func form(b *strings.Builder, raw []byte) {
	for pair := range strings.SplitSeq(string(raw), "&") {
		k, v, _ := strings.Cut(pair, "=")
		k, _ = url.QueryUnescape(k)
		v, _ = url.QueryUnescape(v)
		fmt.Fprintf(b, "%s: %s\n", k, v)
	}
}

func parts(b *strings.Builder, raw []byte, boundary string) {
	r := multipart.NewReader(bytes.NewReader(raw), boundary)
	for {
		p, err := r.NextPart()
		if err == io.EOF {
			return
		}
		if err != nil {
			fmt.Fprintf(b, "unreadable multipart: %v\n", err)
			return
		}
		data, _ := io.ReadAll(p)
		if p.FileName() != "" {
			fmt.Fprintf(b, "--- part %s %s %s\n%s\n", p.FormName(), p.FileName(), p.Header.Get("Content-Type"), digest(data))
			continue
		}
		fmt.Fprintf(b, "--- part %s\n", p.FormName())
		if json.Valid(data) && len(data) > 0 && (data[0] == '{' || data[0] == '[') {
			b.WriteString(indent(data))
		} else {
			b.Write(data)
			b.WriteString("\n")
		}
	}
}

func digest(data []byte) string {
	sum := sha256.Sum256(data)
	return fmt.Sprintf("%d bytes, sha256 %s", len(data), hex.EncodeToString(sum[:]))
}

// mediaType drops a Content-Type's parameters, since a multipart boundary
// is no part of what was sent.
func mediaType(v string) string {
	kind, _, err := mime.ParseMediaType(v)
	if err != nil {
		return v
	}
	return kind
}

// authorization keeps a credential's scheme and a Basic user name, and
// masks the rest.
func authorization(v string) string {
	scheme, cred, ok := strings.Cut(v, " ")
	if !ok {
		return masked
	}
	if strings.EqualFold(scheme, "Basic") {
		if raw, err := base64.StdEncoding.DecodeString(cred); err == nil {
			if user, _, ok := strings.Cut(string(raw), ":"); ok {
				return scheme + " " + user + ":" + masked
			}
		}
	}
	return scheme + " " + masked
}

// mask replaces each secret as written, path-escaped and query-escaped,
// longest first so a secret inside another doesn't leave part of it.
func mask(s string, secrets []string) string {
	forms := make([]string, 0, 3*len(secrets))
	for _, v := range secrets {
		if v != "" {
			forms = append(forms, v, url.PathEscape(v), url.QueryEscape(v))
		}
	}
	slices.SortFunc(forms, func(a, b string) int { return len(b) - len(a) })
	for _, v := range forms {
		s = strings.ReplaceAll(s, v, masked)
	}
	return s
}
