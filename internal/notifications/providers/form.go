package providers

import (
	"bytes"
	"fmt"
	"mime/multipart"
	"net/textproto"
)

// boundary is fixed so a build is the same each time. A poster's bytes
// holding it by chance is the only way it breaks, which is vanishingly
// unlikely at this length.
const boundary = "slipstream-notification-7f3c9a1e5b2d4086"

// part is one multipart/form-data field, a file when file is set.
type part struct {
	name  string
	file  string
	ctype string
	body  []byte
}

func posterPart(name string, p *Poster) part {
	return part{name: name, file: p.Name, ctype: p.Type, body: p.Bytes}
}

// multipartBody is parts as multipart/form-data, with its Content-Type.
func multipartBody(parts []part) (body []byte, contentType string) {
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	if err := w.SetBoundary(boundary); err != nil {
		panic(fmt.Sprintf("providers: %v", err))
	}
	for _, p := range parts {
		h := textproto.MIMEHeader{}
		if p.file != "" {
			h.Set("Content-Disposition", fmt.Sprintf(`form-data; name=%q; filename=%q`, p.name, p.file))
			h.Set("Content-Type", p.ctype)
		} else {
			h.Set("Content-Disposition", fmt.Sprintf(`form-data; name=%q`, p.name))
		}
		pw, err := w.CreatePart(h)
		if err != nil {
			panic(fmt.Sprintf("providers: %v", err))
		}
		_, _ = pw.Write(p.body)
	}
	if err := w.Close(); err != nil {
		panic(fmt.Sprintf("providers: %v", err))
	}
	return b.Bytes(), w.FormDataContentType()
}
