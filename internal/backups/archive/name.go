package archive

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// Name is the file name: slipstream-20261114T040012Z-2.1.0-scheduled-7k3m.tar.zst.age.
// ParseName is the only way a name from outside (a Node's report, an
// upload, a destination folder) becomes a Name, so a path never crosses.
type Name struct{ s string }

const (
	maxName    = 80
	namePrefix = "slipstream-"
	nameSuffix = ".tar.zst.age"
	stampForm  = "20060102T150405Z"
	crockford  = "0123456789abcdefghjkmnpqrstvwxyz"
)

var namePattern = regexp.MustCompile(`^slipstream-[0-9]{8}T[0-9]{6}Z-[0-9A-Za-z.-]+-(scheduled|manual|update|before-restore|before-import|before-handover|superseded|restored)-[0-9a-hjkmnp-tv-z]{4}\.tar\.zst\.age$`)

var errName = errors.New("not a Backup file name")

// NameOf drops each character of the version outside 0-9, A-Z, a-z, "."
// and "-", and shortens the version so the name stays within 80
// characters. The id suffix keeps two Backups' names apart.
func NameOf(m Manifest) Name {
	version := strings.Map(func(r rune) rune {
		if r < 0x80 && (r == '.' || r == '-' || r >= '0' && r <= '9' || r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z') {
			return r
		}
		return -1
	}, m.Version)
	tail := "-" + string(m.Trigger) + "-" + suffix(m.BackupID) + nameSuffix
	head := namePrefix + m.TakenAt.UTC().Format(stampForm) + "-"
	if room := maxName - len(head) - len(tail); len(version) > room {
		version = version[:room]
	}
	return Name{s: head + version + tail}
}

// ParseName refuses a separator, any character outside the pattern and a
// name over 80 characters.
func ParseName(s string) (Name, error) {
	if len(s) > maxName || !namePattern.MatchString(s) {
		return Name{}, errName
	}
	return Name{s: s}, nil
}

func (n Name) String() string { return n.s }

// suffix is the last four characters of the id in Crockford base32, its
// low 20 bits.
func suffix(id ID) string {
	low := uint32(id[13]&0x0f)<<16 | uint32(id[14])<<8 | uint32(id[15])
	b := make([]byte, 4)
	for i := 3; i >= 0; i-- {
		b[i] = crockford[low&0x1f]
		low >>= 5
	}
	return string(b)
}

func (n Name) MarshalText() ([]byte, error) { return []byte(n.s), nil }

// UnmarshalText is ParseName, so a stored name is held to the pattern too.
func (n *Name) UnmarshalText(b []byte) error {
	parsed, err := ParseName(string(b))
	if err != nil {
		return fmt.Errorf("%q: %w", b, err)
	}
	*n = parsed
	return nil
}
