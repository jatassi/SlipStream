package engine

import (
	"errors"
	"fmt"
	"net/netip"
	"slices"

	"google.golang.org/protobuf/proto"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/link"
	tv1 "github.com/jatassi/slipstream/internal/pb/slipstream/transfer/v1"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/transfers/source"
)

// spec is one active Transfer as this Node runs it: an intent's Active,
// parsed.
type spec struct {
	id    ID
	side  tv1.Side
	peer  link.Key
	addrs link.Addresses
	route tv1.Route
	// root and place are the receiver's destination, backup is the
	// Backup folder and name it renames its one file to, download is the
	// download folder it stages in, or scratch is the name under the
	// scratch folder it renames its one file to.
	root     storage.UUID
	place    int64
	backup   *tv1.BackupPlace
	download *Download
	scratch  storing.Rel
	files    []file
}

type file struct {
	position int
	// wire is the source as Open names it.
	wire *tv1.Source
	ref  source.Ref
}

// sameRun reports whether o runs as s does: the same Transfer, side, peer
// and destination. A route change is handed to the run; a shorter file
// list is the Controller catching up on files already reported.
func (s spec) sameRun(o spec) bool {
	return s.id == o.id && s.side == o.side && s.peer == o.peer && s.addrs == o.addrs && s.root == o.root && s.place == o.place &&
		s.scratch == o.scratch && proto.Equal(s.backup, o.backup) && sameDownload(s.download, o.download)
}

// receivesDropped reports whether s receives a file live doesn't list: a
// file its purpose dropped, whose staged bytes go.
func (s spec) receivesDropped(live map[int]bool) bool {
	return s.side == tv1.Side_SIDE_RECEIVE && slices.ContainsFunc(s.files, func(f file) bool { return !live[f.position] })
}

func (s spec) file(position int, src *tv1.Source) (file, bool) {
	i := slices.IndexFunc(s.files, func(f file) bool { return f.position == position })
	if i < 0 || !proto.Equal(s.files[i].wire, src) {
		return file{}, false
	}
	return s.files[i], true
}

func parseActive(a *tv1.Active) (spec, error) {
	u, err := storage.UUIDFromBytes(a.GetTransferId())
	if err != nil {
		return spec{}, fmt.Errorf("transfer id: %w", err)
	}
	s := spec{id: ID{UUID: u}, side: a.GetSide(), route: a.GetRoute()}
	if s.side != tv1.Side_SIDE_SEND && s.side != tv1.Side_SIDE_RECEIVE {
		return spec{}, fmt.Errorf("transfer %s: side %s", s.id, s.side)
	}
	if s.route < tv1.Route_ROUTE_RECEIVER_DIALS || s.route > tv1.Route_ROUTE_BOTH_DIAL {
		return spec{}, fmt.Errorf("transfer %s: route %s", s.id, s.route)
	}
	if len(a.GetPeer().GetKey()) != len(link.Key{}) {
		return spec{}, fmt.Errorf("transfer %s: a peer key of %d bytes", s.id, len(a.GetPeer().GetKey()))
	}
	s.peer = link.Key(a.GetPeer().GetKey())
	if s.addrs, err = parseAddresses(a.GetPeer().GetAddresses()); err != nil {
		return spec{}, fmt.Errorf("transfer %s: %w", s.id, err)
	}
	if s.side == tv1.Side_SIDE_RECEIVE {
		if s, err = s.receiving(a.GetDestination()); err != nil {
			return spec{}, err
		}
	}
	for _, f := range a.GetFiles() {
		ref, err := parseSource(f.GetSource())
		if err != nil {
			return spec{}, fmt.Errorf("transfer %s file %d: %w", s.id, f.GetPosition(), err)
		}
		s.files = append(s.files, file{position: int(f.GetPosition()), wire: f.GetSource(), ref: ref})
	}
	if s.backup != nil && len(s.files) > 1 {
		return spec{}, fmt.Errorf("transfer %s: a Backup of %d files", s.id, len(s.files))
	}
	if s.scratch != (storing.Rel{}) && len(s.files) > 1 {
		return spec{}, fmt.Errorf("transfer %s: a scratch file of %d files", s.id, len(s.files))
	}
	return s, nil
}

// receiving is s with the receiver's destination d.
func (s spec) receiving(d *tv1.Destination) (spec, error) {
	switch k := d.GetKind().(type) {
	case *tv1.Destination_Root:
		var err error
		if s.root, err = storage.UUIDFromBytes(k.Root.GetRootId()); err != nil {
			return spec{}, fmt.Errorf("transfer %s: root id: %w", s.id, err)
		}
		s.place = k.Root.GetPlace()
	case *tv1.Destination_Backup:
		s.backup, s.place = k.Backup, backupPlace
	case *tv1.Destination_BackupsFolder:
		s.backup, s.place = k.BackupsFolder, backupPlace
	case *tv1.Destination_DownloadFolder:
		d, err := parseDownload(k.DownloadFolder)
		if err != nil {
			return spec{}, fmt.Errorf("transfer %s: %w", s.id, err)
		}
		s.download, s.place = &d, backupPlace
	case *tv1.Destination_Scratch:
		var err error
		if s.scratch, err = storing.ParseRel(k.Scratch); err != nil {
			return spec{}, fmt.Errorf("transfer %s: scratch name: %w", s.id, err)
		}
		s.place = backupPlace
	default:
		return spec{}, fmt.Errorf("transfer %s: no destination", s.id)
	}
	return s, nil
}

// backupPlace is the place of a Backup folder, which never moves.
const backupPlace = 1

func parseAddresses(in []string) (link.Addresses, error) {
	if len(in) == 0 {
		return link.Addresses{}, nil
	}
	aps := make([]netip.AddrPort, len(in))
	for i, a := range in {
		ap, err := netip.ParseAddrPort(a)
		if err != nil {
			return link.Addresses{}, fmt.Errorf("peer address: %w", err)
		}
		aps[i] = ap
	}
	return link.ParseAddresses(aps)
}

func parseSource(s *tv1.Source) (source.Ref, error) {
	switch r := s.GetRef().(type) {
	case *tv1.Source_Torrent:
		t := r.Torrent
		if len(t.GetInfoHash()) != len(bencode.InfoHash{}) {
			return nil, fmt.Errorf("an info hash of %d bytes", len(t.GetInfoHash()))
		}
		return source.Torrent{Hash: bencode.InfoHash(t.GetInfoHash()), File: int(t.GetFileIndex())}, nil
	case *tv1.Source_Path:
		if r.Path.GetAbs() == "" {
			return nil, errors.New("an empty source path")
		}
		return source.Path{Abs: r.Path.GetAbs()}, nil
	case *tv1.Source_Backup:
		return source.Backup{Folder: r.Backup.GetFolder(), Name: r.Backup.GetName()}, nil
	case *tv1.Source_Library:
		if len(r.Library.GetRoot()) != len([16]byte{}) || r.Library.GetRel() == "" {
			return nil, errors.New("a library source with no root or path")
		}
		return source.Library{Root: [16]byte(r.Library.GetRoot()), Rel: r.Library.GetRel()}, nil
	case *tv1.Source_Scratch:
		return source.Scratch{Name: r.Scratch}, nil
	}
	return nil, errors.New("a source of no kind this build reads")
}

// Download is a download folder destination: the folder a Download Client
// on this Node keeps the torrent with InfoHash in. Client is nil for the
// built-in client.
type Download struct {
	Client      *[16]byte
	ContentType string
	InfoHash    bencode.InfoHash
}

func parseDownload(w *tv1.DownloadFolder) (Download, error) {
	if len(w.GetInfoHash()) != len(bencode.InfoHash{}) {
		return Download{}, fmt.Errorf("a download folder with an info hash of %d bytes", len(w.GetInfoHash()))
	}
	d := Download{ContentType: w.GetContentType(), InfoHash: bencode.InfoHash(w.GetInfoHash())}
	switch c := w.GetClientId(); len(c) {
	case 0:
	case len([16]byte{}):
		d.Client = (*[16]byte)(c)
	default:
		return Download{}, fmt.Errorf("a download folder with a client id of %d bytes", len(c))
	}
	return d, nil
}

func sameDownload(a, b *Download) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.ContentType == b.ContentType && a.InfoHash == b.InfoHash && (a.Client == nil) == (b.Client == nil) && (a.Client == nil || *a.Client == *b.Client)
}
