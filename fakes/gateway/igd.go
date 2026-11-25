package gateway

import (
	"encoding/xml"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/internal/netsite"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/node"
)

// IGD is the router's UPnP Internet Gateway Device: its description and
// the WANIPConnection and WANPPPConnection services, which list the same
// forwards, as some routers do. SSDP's answer names its front for the
// asking Node. It records into Fake's capture, so the world serves it
// after Fake.
var IGD = fake.Spec{
	Name:      igdName,
	Plays:     netsite.Plays,
	New:       newIGD,
	Endpoints: func(*node.EndpointsBlock, fake.Address) {},
	Online:    fake.NoOnlineRun(online),
}

const (
	descriptionPath = "/rootDesc.xml"
	ipService       = "urn:schemas-upnp-org:service:WANIPConnection:1"
	pppService      = "urn:schemas-upnp-org:service:WANPPPConnection:1"
)

type igd struct {
	site     *netsite.Model
	requests *fake.Capture
}

func newIGD(s scenario.Scenario, env fake.Env) (fake.Server, error) {
	requests, ok := env.Shared.Get(sharedRequests)
	if !ok {
		return nil, errors.New("gateway-igd answers fake-gateway's routers, and the world built no fake-gateway before it")
	}
	site, err := netsite.From(env, s)
	if err != nil {
		return nil, err
	}
	capture, _ := requests.(*fake.Capture)
	g := &igd{site: site, requests: capture}
	return fake.HTTP(env, fake.Routes{
		"description":    {Pattern: "GET " + descriptionPath, Handler: g.description},
		"ip-connection":  {Pattern: "POST /ctl/IPConn", Handler: g.control(ipService)},
		"ppp-connection": {Pattern: "POST /ctl/PPPConn", Handler: g.control(pppService)},
	}), nil
}

// speaks is whether the asking Node's router speaks UPnP; a router that
// doesn't has no IGD to answer.
func (g *igd) speaks(r *http.Request) bool {
	gw, ok := g.site.Gateway(fake.OriginOf(r.Context()))
	return ok && gw.Speaks("upnp")
}

func (g *igd) description(w http.ResponseWriter, r *http.Request) {
	if !g.speaks(r) {
		http.NotFound(w, r)
		return
	}
	g.requests.Record(r.Context(), Request{Protocol: "upnp", Op: "description"})
	w.Header().Set("Content-Type", `text/xml; charset="utf-8"`)
	_, _ = w.Write([]byte(xml.Header + `<root xmlns="urn:schemas-upnp-org:device-1-0"><specVersion><major>1</major><minor>0</minor></specVersion>
<device><deviceType>urn:schemas-upnp-org:device:InternetGatewayDevice:1</deviceType><friendlyName>fake-gateway</friendlyName><UDN>uuid:fake-gateway</UDN>
<deviceList><device><deviceType>urn:schemas-upnp-org:device:WANDevice:1</deviceType><UDN>uuid:fake-gateway-wan</UDN>
<deviceList><device><deviceType>urn:schemas-upnp-org:device:WANConnectionDevice:1</deviceType><UDN>uuid:fake-gateway-conn</UDN>
<serviceList>
<service><serviceType>` + ipService + `</serviceType><serviceId>urn:upnp-org:serviceId:WANIPConn1</serviceId><controlURL>/ctl/IPConn</controlURL><eventSubURL>/evt/IPConn</eventSubURL><SCPDURL>/WANIPCn.xml</SCPDURL></service>
<service><serviceType>` + pppService + `</serviceType><serviceId>urn:upnp-org:serviceId:WANPPPConn1</serviceId><controlURL>/ctl/PPPConn</controlURL><eventSubURL>/evt/PPPConn</eventSubURL><SCPDURL>/WANPPPCn.xml</SCPDURL></service>
</serviceList></device></deviceList></device></deviceList></device></root>`))
}

// UPnP's error codes for WANIPConnection.
const (
	errInvalidArgs   = 402
	errNotAuthorized = 606
	errArrayIndex    = 713
	errNoSuchEntry   = 714
	errConflict      = 718
	errPermanentOnly = 725
)

type envelope struct {
	Body struct {
		Action struct {
			XMLName xml.Name
			Args    []struct {
				XMLName xml.Name
				Value   string `xml:",chardata"`
			} `xml:",any"`
		} `xml:",any"`
	} `xml:"Body"`
}

// control answers one SOAP action of service for the asking Node's site.
func (g *igd) control(service string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !g.speaks(r) {
			http.NotFound(w, r)
			return
		}
		var env envelope
		if err := xml.NewDecoder(r.Body).Decode(&env); err != nil {
			http.Error(w, "not a SOAP envelope", http.StatusBadRequest)
			return
		}
		action := env.Body.Action.XMLName.Local
		args := map[string]string{}
		for _, a := range env.Body.Action.Args {
			args[a.XMLName.Local] = strings.TrimSpace(a.Value)
		}
		write := action == "AddPortMapping" || action == "AddAnyPortMapping" || action == "DeletePortMapping"
		g.requests.Record(r.Context(), Request{Protocol: "upnp", Op: action, Write: write})
		out, code := g.act(fake.OriginOf(r.Context()), action, args)
		if code != 0 {
			soapFault(w, code)
			return
		}
		soapAnswer(w, service, action, out)
	}
}

// act plays action with args on origin's router, answering its out
// arguments or a UPnP error code.
func (g *igd) act(origin, action string, args map[string]string) (out [][2]string, code int) {
	gw, _ := g.site.Gateway(origin)
	forwards := g.site.Forwards(origin)
	switch action {
	case "GetExternalIPAddress":
		wan := ""
		if gw.WAN.IsValid() {
			wan = gw.WAN.String()
		}
		return [][2]string{{"NewExternalIPAddress", wan}}, 0
	case "GetGenericPortMappingEntry":
		i, err := strconv.Atoi(args["NewPortMappingIndex"])
		if err != nil || i < 0 || i >= len(forwards) {
			return nil, errArrayIndex
		}
		return g.entry(forwards[i], true), 0
	case "GetSpecificPortMappingEntry":
		for _, f := range forwards {
			if strconv.Itoa(int(f.External)) == args["NewExternalPort"] && f.Transport == args["NewProtocol"] {
				return g.entry(f, false), 0
			}
		}
		return nil, errNoSuchEntry
	case "AddPortMapping", "AddAnyPortMapping":
		return g.add(origin, gw, action, args)
	case "DeletePortMapping":
		external, err := port(args["NewExternalPort"])
		if err != nil {
			return nil, errInvalidArgs
		}
		if !g.site.Release(origin, func(m netsite.Mapping) bool {
			return m.External == external && m.Transport == args["NewProtocol"]
		}) {
			return nil, errNoSuchEntry
		}
		return nil, 0
	}
	return nil, errInvalidArgs
}

func (g *igd) add(origin string, gw netsite.Gateway, action string, args map[string]string) (out [][2]string, code int) {
	external, err1 := port(args["NewExternalPort"])
	internal, err2 := port(args["NewInternalPort"])
	lease, err3 := strconv.ParseUint(args["NewLeaseDuration"], 10, 32)
	if err := errors.Join(err1, err2, err3); err != nil {
		return nil, errInvalidArgs
	}
	switch {
	case gw.Refuse == "606":
		return nil, errNotAuthorized
	case gw.Refuse == "725" && lease != 0:
		return nil, errPermanentOnly
	case gw.Refuse == "refuse", gw.Refuse == "other-port" && action == "AddPortMapping":
		return nil, errConflict
	}
	if gw.Refuse == "other-port" {
		external++
	}
	g.site.Grant(origin, netsite.Mapping{
		Protocol: "upnp", Transport: args["NewProtocol"], External: external, Node: origin, Internal: internal,
		Lifetime: time.Duration(lease) * time.Second, Description: args["NewPortMappingDescription"],
	})
	if action == "AddAnyPortMapping" {
		return [][2]string{{"NewReservedPort", strconv.Itoa(int(external))}}, 0
	}
	return nil, 0
}

func port(arg string) (uint16, error) {
	p, err := strconv.ParseUint(arg, 10, 16)
	return uint16(p), err
}

func (g *igd) entry(f netsite.Mapping, generic bool) [][2]string {
	var out [][2]string
	if generic {
		out = append(out, [2]string{"NewRemoteHost", ""}, [2]string{"NewExternalPort", strconv.Itoa(int(f.External))}, [2]string{"NewProtocol", f.Transport})
	}
	return append(out,
		[2]string{"NewInternalPort", strconv.Itoa(int(f.Internal))},
		[2]string{"NewInternalClient", g.site.LAN(f.Node).String()},
		[2]string{"NewEnabled", "1"},
		[2]string{"NewPortMappingDescription", f.Description},
		[2]string{"NewLeaseDuration", strconv.Itoa(int(f.Lifetime / time.Second))},
	)
}

func soapAnswer(w http.ResponseWriter, service, action string, out [][2]string) {
	var b strings.Builder
	for _, a := range out {
		b.WriteString("<" + a[0] + ">")
		_ = xml.EscapeText(&b, []byte(a[1]))
		b.WriteString("</" + a[0] + ">")
	}
	w.Header().Set("Content-Type", `text/xml; charset="utf-8"`)
	_, _ = fmt.Fprintf(w, `%s<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/"><s:Body><u:%sResponse xmlns:u="%s">%s</u:%sResponse></s:Body></s:Envelope>`,
		xml.Header, action, service, b.String(), action)
}

func soapFault(w http.ResponseWriter, code int) {
	w.Header().Set("Content-Type", `text/xml; charset="utf-8"`)
	w.WriteHeader(http.StatusInternalServerError)
	_, _ = fmt.Fprintf(w, `%s<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/"><s:Body><s:Fault><faultcode>s:Client</faultcode><faultstring>UPnPError</faultstring><detail><UPnPError xmlns="urn:schemas-upnp-org:control-1-0"><errorCode>%d</errorCode><errorDescription>fake-gateway</errorDescription></UPnPError></detail></s:Fault></s:Body></s:Envelope>`,
		xml.Header, code)
}
