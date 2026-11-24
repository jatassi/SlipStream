package netfacts

import (
	"context"
	"crypto/rand"
	"embed"
	"errors"
	"fmt"
	"net/netip"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/netfacts/gateway"
	"github.com/jatassi/slipstream/internal/netfacts/internal/db"
	"github.com/jatassi/slipstream/internal/netmodel"
	"github.com/jatassi/slipstream/internal/storage"
)

//go:embed schema/*.sql
var schemaFiles embed.FS

// Schema is netfacts' set in every Node's local.db: the router mappings
// the Node asked for.
func Schema() storage.Schema[storage.Local] {
	return storage.NewSchema[storage.Local]("netfacts", schemaFiles)
}

// Want is a port the network intent asks this Node to map: TCP from
// External on its router to Internal here.
type Want struct{ External, Internal uint16 }

// mapper keeps this Node's router mappings to its wants. Its rows in
// local.db are what it holds, so a restart still deletes what it mapped.
type mapper struct {
	local  *storage.DB[storage.Local]
	client *gateway.Client
	// description names this Node's UPnP entries, by which it knows its
	// own when local.db lost their rows.
	description string
	timings     Timings
	// at is the router a row's gateway address names.
	at func(ctx context.Context, gw netip.AddrPort) (gateway.Target, error)
}

var methods = map[netmodel.MapProtocol]string{netmodel.MapPCP: "pcp", netmodel.MapNATPMP: "nat-pmp", netmodel.MapUPnP: "upnp"}

var refusals = map[netmodel.MapRefusal]string{netmodel.OtherPort: "other-port", netmodel.NotAuthorized: "not-authorized", netmodel.NoAnswer: "refused"}

func methodOf(s string) netmodel.MapProtocol {
	for m, name := range methods {
		if name == s {
			return m
		}
	}
	panic("netfacts: a mapping row's method is " + s)
}

func refusalOf(s *string) netmodel.MapRefusal {
	if s == nil {
		return netmodel.RefusedNone
	}
	for r, name := range refusals {
		if name == *s {
			return r
		}
	}
	panic("netfacts: a mapping row's refusal is " + *s)
}

// reconcile maps each wanted port on gw, by the first protocol speaks
// names, renews what is due, and deletes every other mapping this Node
// holds: each row's at the router it went to, and each UPnP entry in
// listed that carries this Node's description and has no row. With gw nil
// the router is unknown, so it only deletes.
func (m *mapper) reconcile(ctx context.Context, gw *gateway.Target, speaks netmodel.Protocols, want []Want, listed []netmodel.Forward) error {
	rows, err := m.rows(ctx)
	if err != nil {
		return err
	}
	held, errs := m.keep(ctx, rows, gw, want)
	if gw == nil {
		return errors.Join(errs...)
	}
	for _, w := range want {
		if err := m.hold(ctx, *gw, speaks, w, held); err != nil {
			errs = append(errs, err)
		}
	}
	for _, f := range listed {
		if f.Description != m.description || slices.ContainsFunc(rows, func(r db.NetfactsMapping) bool { return r.ExternalPort == int64(f.External) }) ||
			slices.ContainsFunc(want, func(w Want) bool { return w.External == f.External }) {
			continue
		}
		if err := m.client.Unmap(ctx, *gw, gateway.Ask{Method: netmodel.MapUPnP, External: f.External, Internal: f.Internal}); err != nil {
			errs = append(errs, fmt.Errorf("delete the router's entry for port %d: %w", f.External, err))
		}
	}
	return errors.Join(errs...)
}

// keep drops each row that isn't a want on gw, and answers the rows still
// held by port: the kept ones and those whose drop failed.
func (m *mapper) keep(ctx context.Context, rows []db.NetfactsMapping, gw *gateway.Target, want []Want) (map[uint16]db.NetfactsMapping, []error) {
	var errs []error
	held := map[uint16]db.NetfactsMapping{}
	for _, row := range rows {
		w := Want{External: uint16(row.ExternalPort), Internal: uint16(row.Port)} //nolint:gosec // G115: CHECKs keep both ports in 1-65535
		if gw != nil && row.Gateway == gw.PCP.String() && slices.Contains(want, w) {
			held[w.Internal] = row
			continue
		}
		if err := m.drop(ctx, row); err != nil {
			errs = append(errs, err)
			held[w.Internal] = row
		}
	}
	return held, errs
}

// hold maps w on gw unless a row already holds it and isn't due, or a row
// for its port waits to be deleted elsewhere. A row asking or granted
// keeps its protocol and nonce; anything else asks by the first protocol
// the router speaks.
func (m *mapper) hold(ctx context.Context, gw gateway.Target, speaks netmodel.Protocols, w Want, held map[uint16]db.NetfactsMapping) error {
	row, ok := held[w.Internal]
	if ok && (row.Gateway != gw.PCP.String() || row.ExternalPort != int64(w.External) || !m.due(row)) {
		return nil
	}
	a := gateway.Ask{External: w.External, Internal: w.Internal, Lifetime: m.timings.Mapping, Description: m.description}
	if ok && row.Refused == nil {
		a.Method = methodOf(row.Method)
		copy(a.Nonce[:], row.Nonce)
	} else {
		a.Method = first(speaks)
		_, _ = rand.Read(a.Nonce[:])
	}
	if a.Method == 0 {
		return nil
	}
	return m.ask(ctx, gw, a)
}

// first is the protocol a new mapping asks by: PCP, then NAT-PMP, then
// UPnP, of those the router answered; 0 when it answered none.
func first(speaks netmodel.Protocols) netmodel.MapProtocol {
	switch {
	case speaks.PCP == netmodel.Yes:
		return netmodel.MapPCP
	case speaks.NATPMP == netmodel.Yes:
		return netmodel.MapNATPMP
	case speaks.UPnP == netmodel.Yes:
		return netmodel.MapUPnP
	}
	return 0
}

// due is whether row's mapping is asked again: a row still asking, a
// grant past its renewal point, or a refusal as old as a renewal.
func (m *mapper) due(row db.NetfactsMapping) bool {
	now := time.Now()
	granted, expires := row.GrantedAt.Time(), row.ExpiresAt.Time()
	switch {
	case row.Refused != nil:
		return !now.Before(row.AskedAt.Time().Add(time.Duration(float64(m.timings.Mapping) * m.timings.RenewFraction)))
	case granted.IsZero():
		return true
	case expires.IsZero():
		return false
	}
	return !now.Before(granted.Add(time.Duration(float64(expires.Sub(granted)) * m.timings.RenewFraction)))
}

// ask maps a on gw, its row written asking before the call and set
// granted or refused after it.
func (m *mapper) ask(ctx context.Context, gw gateway.Target, a gateway.Ask) error {
	var nonce []byte
	if a.Method == netmodel.MapPCP {
		nonce = a.Nonce[:]
	}
	err := m.local.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Local]) error {
		return db.New(tx).AskMapping(ctx, db.AskMappingParams{
			Port: int64(a.Internal), Method: methods[a.Method], Gateway: gw.PCP.String(), ExternalPort: int64(a.External),
			Nonce: nonce, AskedAt: storage.FromTime(time.Now()),
		})
	})
	if err != nil {
		return fmt.Errorf("record the mapping of port %d: %w", a.External, err)
	}
	got, err := m.client.Map(ctx, gw, a)
	if err != nil {
		return fmt.Errorf("map port %d: %w", a.External, err)
	}
	return m.local.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Local]) error {
		q := db.New(tx)
		if got.Refused != netmodel.RefusedNone {
			return q.RefuseMapping(ctx, db.RefuseMappingParams{Refused: new(refusals[got.Refused]), Port: int64(a.Internal)})
		}
		return q.GrantMapping(ctx, db.GrantMappingParams{GrantedAt: storage.FromTime(time.Now()), ExpiresAt: storage.FromTime(got.Expires), Port: int64(a.Internal)})
	})
}

// drop deletes row's mapping at the router it went to, then the row. A
// refused row holds nothing there.
func (m *mapper) drop(ctx context.Context, row db.NetfactsMapping) error {
	if row.Refused == nil {
		at, err := netip.ParseAddrPort(row.Gateway)
		if err != nil {
			return fmt.Errorf("read a mapping's router: %w", err)
		}
		gw, err := m.at(ctx, at)
		if err != nil {
			return err
		}
		a := gateway.Ask{Method: methodOf(row.Method), External: uint16(row.ExternalPort), Internal: uint16(row.Port)} //nolint:gosec // G115: CHECKs keep both ports in 1-65535
		copy(a.Nonce[:], row.Nonce)
		if err := m.client.Unmap(ctx, gw, a); err != nil {
			return fmt.Errorf("delete the mapping of port %d: %w", a.External, err)
		}
	}
	return m.local.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Local]) error {
		return db.New(tx).DeleteMapping(ctx, row.Port)
	})
}

func (m *mapper) rows(ctx context.Context) ([]db.NetfactsMapping, error) {
	var rows []db.NetfactsMapping
	err := m.local.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Local]) error {
		var err error
		rows, err = db.New(r).ListMappings(ctx)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("read this Node's router mappings: %w", err)
	}
	return rows, nil
}

// mappings is what this Node holds as facts: each grant, each refusal,
// and a mapping still asking as one the router didn't answer.
func (m *mapper) mappings(ctx context.Context) ([]netmodel.Mapping, error) {
	rows, err := m.rows(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]netmodel.Mapping, 0, len(rows))
	for _, row := range rows {
		mp := netmodel.Mapping{Protocol: methodOf(row.Method), External: uint16(row.ExternalPort), Internal: uint16(row.Port), Refused: refusalOf(row.Refused)} //nolint:gosec // G115: CHECKs keep both ports in 1-65535
		granted, expires := row.GrantedAt.Time(), row.ExpiresAt.Time()
		switch {
		case granted.IsZero() && mp.Refused == netmodel.RefusedNone:
			mp.Refused = netmodel.NoAnswer
		case !expires.IsZero():
			mp.Lifetime, mp.Expires = expires.Sub(granted), expires
		}
		out = append(out, mp)
	}
	return out, nil
}
