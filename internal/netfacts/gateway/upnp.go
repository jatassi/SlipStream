package gateway

import (
	"bufio"
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/netmodel"
)

const (
	igdDevice  = "urn:schemas-upnp-org:device:InternetGatewayDevice:1"
	ipService  = "urn:schemas-upnp-org:service:WANIPConnection:1"
	pppService = "urn:schemas-upnp-org:service:WANPPPConnection:1"
	// descriptionLimit bounds a router's description and each SOAP answer,
	// far above any real router's few kilobytes.
	descriptionLimit = 1 << 20
)

// igdTypes are the search targets an answer may name to count as an IGD.
var igdTypes = []string{igdDevice, ipService, pppService, "urn:schemas-upnp-org:device:InternetGatewayDevice:2", "urn:schemas-upnp-org:service:WANIPConnection:2"}

type igdRead struct {
	speaks   netmodel.Tri
	wan      netip.Addr
	forwards []netmodel.Forward
}

// upnp searches for the router's IGD and reads its WAN address and its
// TCP forwards from every WAN connection service, deduped by external
// port, since WANIPConnection and WANPPPConnection often list the same.
func (c *Client) upnp(ctx context.Context, gw Target, wanted []uint16) (igdRead, error) {
	location, err := c.search(ctx, gw)
	if err != nil || location == nil {
		return igdRead{speaks: netmodel.No}, err
	}
	services, err := c.services(ctx, location)
	if err != nil || len(services) == 0 {
		return igdRead{speaks: netmodel.No}, nil //nolint:nilerr // an IGD whose description won't load speaks nothing usable
	}
	out := igdRead{speaks: netmodel.Yes}
	seen := map[uint16]bool{}
	for _, s := range services {
		if !out.wan.IsValid() {
			out.wan = c.wan(ctx, s)
		}
		for _, f := range c.forwards(ctx, s, wanted) {
			if !seen[f.External] {
				seen[f.External] = true
				out.forwards = append(out.forwards, f)
			}
		}
	}
	return out, nil
}

// wan is the external address s answers, unset when it has none.
func (c *Client) wan(ctx context.Context, s service) netip.Addr {
	args, code := c.soap(ctx, s, "GetExternalIPAddress", nil)
	if code != 0 {
		return netip.Addr{}
	}
	a, err := netip.ParseAddr(args["NewExternalIPAddress"])
	if err != nil || a.IsUnspecified() {
		return netip.Addr{}
	}
	return a
}

// forwards is s's TCP forwards: its whole list, then each wanted port
// asked by name.
func (c *Client) forwards(ctx context.Context, s service, wanted []uint16) []netmodel.Forward {
	var out []netmodel.Forward
	// The list ends at the first error, 713 on a router that keeps to
	// the spec.
	for i := 0; ; i++ {
		args, code := c.soap(ctx, s, "GetGenericPortMappingEntry", [][2]string{{"NewPortMappingIndex", strconv.Itoa(i)}})
		if code != 0 {
			break
		}
		ext, err := strconv.ParseUint(args["NewExternalPort"], 10, 16)
		if err == nil && args["NewProtocol"] == "TCP" {
			out = append(out, forward(uint16(ext), args))
		}
	}
	for _, p := range wanted {
		args, code := c.soap(ctx, s, "GetSpecificPortMappingEntry", [][2]string{
			{"NewRemoteHost", ""}, {"NewExternalPort", strconv.Itoa(int(p))}, {"NewProtocol", "TCP"},
		})
		if code == 0 {
			out = append(out, forward(p, args))
		}
	}
	return out
}

func forward(external uint16, args map[string]string) netmodel.Forward {
	internal, _ := strconv.ParseUint(args["NewInternalPort"], 10, 16)
	client, _ := netip.ParseAddr(args["NewInternalClient"])
	return netmodel.Forward{External: external, Client: client, Internal: uint16(internal), Description: args["NewPortMappingDescription"]}
}

// search sends one M-SEARCH and answers the LOCATION of the first answer
// from the router whose ST is an IGD type, nil when none comes.
func (c *Client) search(ctx context.Context, gw Target) (*url.URL, error) {
	msg := []byte("M-SEARCH * HTTP/1.1\r\nHOST: 239.255.255.250:1900\r\nMAN: \"ssdp:discover\"\r\nMX: 1\r\nST: " + igdDevice + "\r\n\r\n")
	wait := c.Timings.Exchange
	var conn *net.UDPConn
	var err error
	if gw.Unicast {
		conn, err = c.Packets.DialUDP(ctx, gw.SSDP)
	} else {
		wait = c.Timings.SSDPWait
		conn, err = c.Packets.ListenUDP(ctx, netip.AddrPortFrom(gw.Local, 0))
	}
	if err != nil {
		return nil, fmt.Errorf("open a socket for SSDP: %w", err)
	}
	defer func() { _ = conn.Close() }()
	if gw.Unicast {
		_, err = conn.Write(msg)
	} else {
		_, err = conn.WriteToUDPAddrPort(msg, gw.SSDP)
	}
	if err != nil {
		return nil, nil
	}
	var location *url.URL
	readFrom(conn, gw.PCP.Addr(), time.Now().Add(wait), func(p []byte) bool {
		res, err := http.ReadResponse(bufio.NewReader(bytes.NewReader(p)), nil)
		if err != nil {
			return false
		}
		_ = res.Body.Close()
		u, err := url.Parse(res.Header.Get("Location"))
		if err != nil || !slices.Contains(igdTypes, res.Header.Get("St")) || (u.Scheme != "http" && u.Scheme != "https") {
			return false
		}
		location = u
		return true
	})
	return location, nil
}

type service struct {
	kind    string
	control *url.URL
}

type description struct {
	Base   string    `xml:"URLBase"`
	Device xmlDevice `xml:"device"`
}

type xmlDevice struct {
	Services []struct {
		Type    string `xml:"serviceType"`
		Control string `xml:"controlURL"`
	} `xml:"serviceList>service"`
	Devices []xmlDevice `xml:"deviceList>device"`
}

// services are the WAN connection services the description at location
// lists, at any depth.
func (c *Client) services(ctx context.Context, location *url.URL) ([]service, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, location.String(), http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("the router's description: %w", err)
	}
	body, err := c.do(req)
	if err != nil {
		return nil, err
	}
	var d description
	if err := xml.Unmarshal(body, &d); err != nil {
		return nil, fmt.Errorf("the router's description: %w", err)
	}
	base := location
	if b, err := url.Parse(d.Base); d.Base != "" && err == nil {
		base = b
	}
	var out []service
	var walk func(xmlDevice)
	walk = func(dev xmlDevice) {
		for _, s := range dev.Services {
			if s.Type != ipService && s.Type != pppService {
				continue
			}
			if u, err := base.Parse(s.Control); err == nil {
				out = append(out, service{kind: s.Type, control: u})
			}
		}
		for _, sub := range dev.Devices {
			walk(sub)
		}
	}
	walk(d.Device)
	return out, nil
}

func (c *Client) do(req *http.Request) ([]byte, error) {
	ctx, cancel := context.WithTimeout(req.Context(), c.Timings.Exchange)
	defer cancel()
	res, err := c.HTTP.Do(req.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("ask the router: %w", err)
	}
	defer func() { _ = res.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(res.Body, descriptionLimit))
	if err != nil {
		return nil, fmt.Errorf("read the router's answer: %w", err)
	}
	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusInternalServerError {
		return nil, fmt.Errorf("the router answered %s", res.Status)
	}
	return body, nil
}

type soapAnswer struct {
	Body struct {
		Inner struct {
			Args []struct {
				XMLName xml.Name
				Value   string `xml:",chardata"`
			} `xml:",any"`
		} `xml:",any"`
		Fault struct {
			Code int `xml:"detail>UPnPError>errorCode"`
		} `xml:"Fault"`
	} `xml:"Body"`
}

// soap calls action on s and answers its out arguments, or the UPnP error
// code: -1 for an answer that isn't SOAP.
func (c *Client) soap(ctx context.Context, s service, action string, args [][2]string) (out map[string]string, code int) {
	var b strings.Builder
	for _, a := range args {
		b.WriteString("<" + a[0] + ">")
		_ = xml.EscapeText(&b, []byte(a[1]))
		b.WriteString("</" + a[0] + ">")
	}
	body := `<?xml version="1.0"?><s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/"><s:Body><u:` +
		action + ` xmlns:u="` + s.kind + `">` + b.String() + `</u:` + action + `></s:Body></s:Envelope>`
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.control.String(), strings.NewReader(body))
	if err != nil {
		return nil, -1
	}
	req.Header.Set("Content-Type", `text/xml; charset="utf-8"`)
	req.Header.Set("SOAPAction", `"`+s.kind+"#"+action+`"`)
	raw, err := c.do(req)
	if err != nil {
		return nil, -1
	}
	var ans soapAnswer
	if err := xml.Unmarshal(raw, &ans); err != nil {
		return nil, -1
	}
	if ans.Body.Fault.Code != 0 {
		return nil, ans.Body.Fault.Code
	}
	out = map[string]string{}
	for _, a := range ans.Body.Inner.Args {
		out[a.XMLName.Local] = strings.TrimSpace(a.Value)
	}
	return out, 0
}
