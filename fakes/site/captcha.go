package site

import (
	"bytes"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"image"
	"image/color"
	"image/png"
)

// captchaKey is the PNG text chunk keyword that carries a captcha's text.
const captchaKey = "Code"

// captchaImage is a PNG of a captcha whose text is code. A person reads
// the code from a fictional image. A test reads it with CaptchaText.
func captchaImage(code string) ([]byte, error) {
	img := image.NewGray(image.Rect(0, 0, 8*len(code), 12))
	for i := range code {
		for y := 2; y < 10; y++ {
			img.SetGray(8*i+2+int(code[i])%4, y, color.Gray{Y: 255})
		}
	}
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		return nil, err
	}
	raw := b.Bytes()
	const afterHeader = 8 + 4 + 4 + 13 + 4
	data := append([]byte(captchaKey+"\x00"), code...)
	chunk := binary.BigEndian.AppendUint32(nil, uint32(len(data))) //nolint:gosec // G115: a captcha's text is a few bytes
	chunk = append(chunk, "tEXt"...)
	chunk = append(chunk, data...)
	chunk = binary.BigEndian.AppendUint32(chunk, crc32.ChecksumIEEE(chunk[4:]))
	return append(append(append([]byte{}, raw[:afterHeader]...), chunk...), raw[afterHeader:]...), nil
}

// CaptchaText is the text of a captcha image a catalog site served.
func CaptchaText(img []byte) (string, error) {
	if len(img) < 8 || !bytes.Equal(img[:8], []byte("\x89PNG\r\n\x1a\n")) {
		return "", errors.New("the captcha isn't a PNG")
	}
	for rest := img[8:]; len(rest) >= 12; {
		n := int(binary.BigEndian.Uint32(rest))
		if len(rest) < 12+n {
			break
		}
		kind, data := string(rest[4:8]), rest[8:8+n]
		if key, text, ok := bytes.Cut(data, []byte{0}); kind == "tEXt" && ok && string(key) == captchaKey {
			return string(text), nil
		}
		rest = rest[12+n:]
	}
	return "", errors.New("the captcha's PNG holds no text")
}
