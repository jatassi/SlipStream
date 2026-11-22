package archive

import (
	_ "embed"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
)

// words.txt is age's word list, BIP39's 2048 English words (internal/notices/age).
//
//go:embed words.txt
var wordsFile string

var wordlist = strings.Fields(wordsFile)

const passphraseWords = 10

// Passphrase is 10 words from age's list, about 110 bits. It never prints:
// String and LogValue answer "[passphrase]".
type Passphrase struct{ words string }

// NewPassphrase draws each word from two bytes of rand. 2048 divides
// 65536, so every word is equally likely.
func NewPassphrase(rand io.Reader) Passphrase {
	words := make([]string, passphraseWords)
	buf := make([]byte, 2)
	for i := range words {
		if _, err := io.ReadFull(rand, buf); err != nil {
			panic(fmt.Sprintf("archive: draw a passphrase word: %v", err))
		}
		words[i] = wordlist[int(binary.BigEndian.Uint16(buf))%len(wordlist)]
	}
	return Passphrase{words: strings.Join(words, " ")}
}

var errEmptyPassphrase = errors.New("the passphrase is empty")

// ParsePassphrase trims, folds case and collapses spaces, so a passphrase
// typed from a printed sheet matches the one generated.
func ParsePassphrase(typed string) (Passphrase, error) {
	words := strings.Join(strings.Fields(strings.ToLower(typed)), " ")
	if words == "" {
		return Passphrase{}, errEmptyPassphrase
	}
	return Passphrase{words: words}, nil
}

// Reveal is the one accessor. Its callers are the reconfirmed API and the
// recovery sheet.
func (p Passphrase) Reveal() string { return p.words }

func (p Passphrase) String() string { return "[passphrase]" }

func (p Passphrase) GoString() string { return "[passphrase]" }

func (p Passphrase) LogValue() slog.Value { return slog.StringValue("[passphrase]") }
