package setting

import (
	"encoding/json"
	"net/netip"
	"strings"
)

// IPFamily is the family of address an IP setting takes.
type IPFamily string

const (
	AnyFamily IPFamily = "any"
	IPv6Only  IPFamily = "ipv6"
)

// IP is an IP address of Family with no zone, empty by default.
type IP struct {
	Key, Label string
	Family     IPFamily
}

// Value is the address, the zero Addr when it is empty.
func (i IP) Value(override *netip.Addr) netip.Addr {
	if override == nil {
		return netip.Addr{}
	}
	return *override
}

// Parse accepts an address of the setting's family; nil or blank empties
// it.
func (i IP) Parse(text *string) (Stored[netip.Addr], error) {
	if text == nil || strings.TrimSpace(*text) == "" {
		return Stored[netip.Addr]{}, nil
	}
	typed := strings.TrimSpace(*text)
	example := map[IPFamily]string{AnyFamily: "an IP address, such as 192.168.1.20", IPv6Only: "an IPv6 address, such as 2001:db8::20"}[i.Family]
	a, err := netip.ParseAddr(typed)
	switch {
	case err != nil:
		return Stored[netip.Addr]{}, refuse(i.Key, "%q isn't an IP address. Use %s.", typed, example)
	case a.Zone() != "":
		return Stored[netip.Addr]{}, refuse(i.Key, "Leave out the zone, the part from %%%s.", a.Zone())
	case i.Family == IPv6Only && a.Unmap().Is4():
		return Stored[netip.Addr]{}, refuse(i.Key, "%s is an IPv4 address. Use %s.", a.Unmap(), example)
	}
	return override(a.Unmap()), nil
}

func (i IP) ParseJSON(raw json.RawMessage) (Stored[netip.Addr], error) {
	return parseJSON(raw, i.Key, "An IP address, as a string.", i.Parse)
}

func (i IP) Field(override *netip.Addr) IPField {
	f := IPField{About: about(i.Key, i.Label, override == nil), Kind: "ip", Family: string(i.Family)}
	if override != nil {
		s := override.String()
		f.Value, f.Running = &s, &s
	}
	return f
}

// IPField is an IP on the wire.
type IPField struct {
	About
	Kind    string  `json:"kind" enum:"ip"`
	Value   *string `json:"value" doc:"The address, or null when it is empty"`
	Running *string `json:"running" doc:"The address, or null when it is empty"`
	Default *string `json:"default" doc:"Always null: the setting is empty by default"`
	Family  string  `json:"family" enum:"any,ipv6" doc:"Which addresses the setting takes: any IP address, or IPv6 only"`
}
