// Package reachability is Remote access on the Controller. words.go is the
// one place a check, fix or fact of the judge becomes a sentence, for the
// API, Health and the web alike.
package reachability

import (
	"fmt"
	"net/netip"
	"strconv"

	"github.com/jatassi/slipstream/internal/netmodel"
	"github.com/jatassi/slipstream/internal/reachability/judge"
	"github.com/jatassi/slipstream/internal/role"
)

// Words renders the judge's ids. Node names a Node by id.
type Words struct {
	Node func(role.NodeID) string
}

// param names a Params field a template reads.
type param uint8

const (
	pNode param = iota
	pTester
	pHost
	pPort
	pLocalPort
	pAddr
	pLocalAddr
	pWant
	pPrefix
	pDate
	pTool
	pClass
	pProgram
	pFamily
	pURL
	pRaw
	pReason
)

// sentence is one id's words. needs are the Params fields the judge always
// sets for the id, and may the ones a template reads when they are set.
type sentence struct {
	needs, may []param
	title      func(w Words, p judge.Params) string
	step       func(w Words, p judge.Params) string
	details    string
}

func needs(ps ...param) []param { return ps }

var fixes = map[judge.FixID]sentence{
	judge.FixForwardNotSeen: {needs: needs(pNode, pPort, pLocalPort), may: needs(pLocalAddr),
		title: fixed("SlipStream can't see your router's port forwards."),
		step: func(w Words, p judge.Params) string {
			return fmt.Sprintf("Forward TCP port %d to %s, port %d, then add a Node outside your home to confirm it.", p.Port, w.where(p), p.LocalPort)
		},
		details: "If your provider shares your address (CGNAT), forwarding doesn't help."},
	judge.FixManualForward: {needs: needs(pNode, pPort, pLocalPort), may: needs(pLocalAddr),
		title: func(_ Words, p judge.Params) string { return fmt.Sprintf("Port %d is closed at your router.", p.Port) },
		step: func(w Words, p judge.Params) string {
			return fmt.Sprintf("Forward TCP port %d to %s, port %d.", p.Port, w.where(p), p.LocalPort)
		}},
	judge.FixNoGateway: {needs: needs(pNode, pPort),
		title: func(w Words, p judge.Params) string {
			return fmt.Sprintf("Players outside can't connect to %s on port %d.", w.Node(p.Node), p.Port)
		},
		step: func(w Words, p judge.Params) string {
			return fmt.Sprintf("Allow TCP port %d to %s in your router or your cloud provider's firewall.", p.Port, w.Node(p.Node))
		}},
	judge.FixMappable: {needs: needs(pNode, pPort),
		title: func(_ Words, p judge.Params) string {
			return fmt.Sprintf("Your router can open port %d for SlipStream, and SlipStream hasn't opened it.", p.Port)
		},
		step: func(w Words, p judge.Params) string {
			return fmt.Sprintf("Turn on Open router ports automatically for %s.", w.Node(p.Node))
		}},
	judge.FixMappablePlex: {needs: needs(pNode),
		title: fixed("Your router can open Plex's port, and nothing has opened it."),
		step: func(w Words, p judge.Params) string {
			return fmt.Sprintf("Turn on Remote access in Plex, or turn on Open router ports automatically for %s.", w.Node(p.Node))
		}},
	judge.FixCGNAT: {needs: needs(pNode), may: needs(pProgram),
		title: func(w Words, p judge.Params) string {
			return fmt.Sprintf("Your provider shares one public IPv4 address between many homes (CGNAT), so players outside can't connect to %s directly.", w.Node(p.Node))
		},
		step: func(_ Words, p judge.Params) string {
			return fmt.Sprintf("Ask your provider for a public IPv4 address, or put %s behind your own proxy on a VPS.", subject(p))
		}},
	judge.FixDoubleNAT: {needs: needs(pNode, pPort, pAddr),
		title: func(w Words, p judge.Params) string { return w.Node(p.Node) + " is behind two routers." },
		step: func(_ Words, p judge.Params) string {
			return fmt.Sprintf("On the outer one, forward TCP port %d to %s, your router's internet side, or set the outer one to bridge mode.", p.Port, p.Addr)
		},
		details: "If your provider runs the outer router, ask them for a public IPv4 address."},
	judge.FixVPN: {needs: needs(pNode, pPort, pLocalPort), may: needs(pLocalAddr),
		title: func(w Words, p judge.Params) string {
			return w.Node(p.Node) + " sends its internet traffic through a VPN, so SlipStream can't check your router."
		},
		step: func(w Words, p judge.Params) string {
			return fmt.Sprintf("Forward TCP port %d to %s, port %d, and exclude that port from the VPN on %s.", p.Port, w.where(p), p.LocalPort, w.Node(p.Node))
		}},
	judge.FixFirewall: {needs: needs(pNode, pPort, pTool, pFamily), may: needs(pProgram),
		title: func(w Words, p judge.Params) string {
			return fmt.Sprintf("%s on %s blocks port %d.", toolName(p.Tool), w.Node(p.Node), p.Port)
		},
		step: func(w Words, p judge.Params) string {
			switch p.Tool {
			case netmodel.WindowsDefender:
				return fmt.Sprintf("On %s, open PowerShell as administrator and run:", w.Node(p.Node))
			case netmodel.SocketFilter:
				return fmt.Sprintf("On %s, open Terminal and run:", w.Node(p.Node))
			case netmodel.UFW, netmodel.Firewalld:
				return fmt.Sprintf("On %s, run:", w.Node(p.Node))
			case netmodel.ToolUnknown:
			}
			return fmt.Sprintf("Allow TCP port %d in the firewall on %s.", p.Port, w.Node(p.Node))
		}},
	judge.FixFirewallUnknown: {needs: needs(pNode, pPort), may: needs(pTool, pFamily, pProgram),
		title: func(w Words, p judge.Params) string {
			return fmt.Sprintf("SlipStream can't read the firewall on %s.", w.Node(p.Node))
		},
		step: func(_ Words, p judge.Params) string {
			return fmt.Sprintf("Allow TCP port %d in it, then add a Node outside your home to confirm it.", p.Port)
		}},
	judge.FixFirewallDocker: {needs: needs(pNode, pPort), may: needs(pTool, pFamily, pProgram),
		title: func(_ Words, p judge.Params) string {
			return fmt.Sprintf("Docker publishes port %d past ufw, so SlipStream can't read whether it's allowed.", p.Port)
		},
		step: func(w Words, p judge.Params) string {
			return fmt.Sprintf("Allow TCP port %d to %s, then add a Node outside your home to confirm it.", p.Port, w.Node(p.Node))
		}},
	judge.FixBind: {needs: needs(pNode),
		title: func(w Words, p judge.Params) string {
			return fmt.Sprintf("SlipStream on %s accepts connections from %[1]s only.", w.Node(p.Node))
		},
		step: func(w Words, p judge.Params) string {
			return fmt.Sprintf("Set Listen on to All networks in Nodes › %s › Web and API, then restart %[1]s.", w.Node(p.Node))
		}},
	judge.FixBindProgram: {needs: needs(pNode, pPort), may: needs(pProgram),
		title: func(w Words, p judge.Params) string {
			return fmt.Sprintf("%s on %s accepts connections from %[2]s only.", program(p), w.Node(p.Node))
		},
		step: fixed("Set it to listen on all networks, then restart it.")},
	judge.FixBindFamily: {needs: needs(pNode, pPort, pFamily), may: needs(pProgram),
		title: func(w Words, p judge.Params) string {
			return fmt.Sprintf("%s on %s doesn't accept %s connections on port %d.", program(p), w.Node(p.Node), p.Family, p.Port)
		},
		step: func(_ Words, p judge.Params) string {
			return fmt.Sprintf("Set it to listen on %s too, then restart it.", p.Family)
		}},
	judge.FixBindUnknown: {needs: needs(pNode, pPort, pFamily), may: needs(pProgram),
		title: func(w Words, p judge.Params) string {
			return fmt.Sprintf("SlipStream can't tell whether %s on %s accepts %s connections on port %d.", lower(program(p)), w.Node(p.Node), p.Family, p.Port)
		},
		step: fixed("Check that it listens on all networks, then add a Node outside your home to confirm it.")},
	judge.FixPlainHTTP: {
		title: fixed("The UI and portal use plain HTTP, so sign-ins cross the internet unencrypted."),
		step:  fixed("Put SlipStream behind an HTTPS proxy and set Address to its https:// name.")},
	judge.FixPrivateAddress: {needs: needs(pURL),
		title: func(_ Words, p judge.Params) string {
			return fmt.Sprintf("Outsiders can't use your Address, %s.", p.URL)
		},
		step: fixed("Set Address to the https:// name outsiders use.")},
	judge.FixStaleA: {needs: needs(pHost, pAddr, pWant),
		title: func(_ Words, p judge.Params) string {
			return fmt.Sprintf("The A record for %s is %s, your home's old address.", p.Host, p.Addr)
		},
		step: func(_ Words, p judge.Params) string { return fmt.Sprintf("Change it to %s.", p.Want) }},
	judge.FixExtraA: {needs: needs(pHost, pAddr),
		title: func(_ Words, p judge.Params) string {
			return fmt.Sprintf("The A record for %s also lists %s, which isn't your home's address.", p.Host, p.Addr)
		},
		step: func(_ Words, p judge.Params) string { return fmt.Sprintf("Remove %s from it.", p.Addr) }},
	judge.FixPrivateA: {needs: needs(pHost, pAddr),
		title: func(_ Words, p judge.Params) string {
			return fmt.Sprintf("The A record for %s is %s, which works only inside your home.", p.Host, p.Addr)
		},
		step: fixed("Change it to your home's public IPv4 address.")},
	judge.FixDNSFailed: {needs: needs(pHost), may: needs(pRaw),
		title: func(_ Words, p judge.Params) string {
			return fmt.Sprintf("Lookups of the A record for %s fail.", p.Host)
		},
		step: fixed("Add the A record at your DNS provider, or fix it there.")},
	judge.FixNoA: {needs: needs(pHost), may: needs(pProgram),
		title: func(_ Words, p judge.Params) string {
			return p.Host + " has no A record, so players without IPv6 can't connect."
		},
		step: func(_ Words, p judge.Params) string {
			return fmt.Sprintf("Add an A record, or put %s behind your own proxy on a VPS.", subject(p))
		}},
	judge.FixAAAAOffOrigin: {needs: needs(pHost, pAddr, pNode),
		title: func(w Words, p judge.Params) string {
			return fmt.Sprintf("The AAAA record for %s is %s, which %s doesn't have.", p.Host, p.Addr, w.Node(p.Node))
		},
		step: func(w Words, p judge.Params) string {
			return fmt.Sprintf("Change it to an IPv6 address of %s, or remove it.", w.Node(p.Node))
		}},
	judge.FixPrefixChange: {needs: needs(pHost, pAddr, pNode), may: needs(pPrefix, pWant),
		title: func(w Words, p judge.Params) string {
			return fmt.Sprintf("The AAAA record for %s is %s, which %s no longer has.", p.Host, p.Addr, w.Node(p.Node))
		},
		step: func(w Words, p judge.Params) string {
			switch {
			case p.Want.IsValid():
				return fmt.Sprintf("Change it to %s, or remove it.", p.Want)
			case p.Prefix.IsValid():
				return fmt.Sprintf("Change it to an address in %s's new prefix, %s, or remove it.", w.Node(p.Node), p.Prefix)
			}
			return "Remove it."
		}},
	judge.FixAAAAFailed: {needs: needs(pHost), may: needs(pRaw),
		title: func(_ Words, p judge.Params) string {
			return fmt.Sprintf("Lookups of the AAAA record for %s fail.", p.Host)
		},
		step: fixed("Fix the record at your DNS provider, or remove it.")},
	judge.FixPrivateAAAA: {needs: needs(pHost, pAddr),
		title: func(_ Words, p judge.Params) string {
			return fmt.Sprintf("The AAAA record for %s is %s, which works only inside your home.", p.Host, p.Addr)
		},
		step: fixed("Change it to a global IPv6 address, or remove it.")},
	judge.FixBrokenAAAA: {needs: needs(pHost, pAddr, pPort, pNode),
		title: func(_ Words, p judge.Params) string {
			return fmt.Sprintf("Outsiders can't connect to %s over IPv6 at %s.", p.Host, p.Addr)
		},
		step: func(w Words, p judge.Params) string {
			return fmt.Sprintf("Open port %d for IPv6 to %s, or remove the AAAA record.", p.Port, w.Node(p.Node))
		}},
	judge.FixUntestedIPv6: {needs: needs(pNode, pPort),
		title: func(w Words, p judge.Params) string {
			return fmt.Sprintf("Nothing outside your home has tested %s over IPv6.", w.Node(p.Node))
		},
		step: func(w Words, p judge.Params) string {
			return fmt.Sprintf("Open port %d for IPv6 to %s in your router.", p.Port, w.Node(p.Node))
		}},
	judge.FixProxyPort: {needs: needs(pPort, pAddr),
		title: func(_ Words, p judge.Params) string {
			return fmt.Sprintf("Port %d on your proxy at %s is closed.", p.Port, p.Addr)
		},
		step: fixed("Open it in the proxy's firewall or your provider's firewall.")},
	judge.FixProxyForward: {needs: needs(pAddr, pNode, pPort),
		title: func(_ Words, p judge.Params) string {
			return fmt.Sprintf("Players can't connect through your proxy at %s.", p.Addr)
		},
		step: func(w Words, p judge.Params) string {
			return fmt.Sprintf("Check that your proxy at %s forwards to %s, port %d.", p.Addr, w.Node(p.Node), p.Port)
		}},
	judge.FixCustomURL: {needs: needs(pURL),
		title: func(_ Words, p judge.Params) string {
			return fmt.Sprintf("Plex doesn't list %s as a custom server access URL, so Plex apps outside your home can't find it.", p.URL)
		},
		step: fixed("In Plex, open Settings › Network, select Show Advanced, and add it to Custom server access URLs.")},
	judge.FixCertificate: {needs: needs(pHost, pRaw), may: needs(pDate),
		title: func(_ Words, p judge.Params) string {
			switch p.Raw {
			case judge.Expired.String():
				return fmt.Sprintf("The certificate for %s expired on %s.", p.Host, p.Date.Format("Jan 2, 2006"))
			case judge.WrongName.String():
				return fmt.Sprintf("The certificate at %s is for another name.", p.Host)
			case judge.Untrusted.String():
				return fmt.Sprintf("The certificate for %s isn't from a trusted authority.", p.Host)
			}
			return p.Host + " doesn't complete a secure connection."
		},
		step: func(_ Words, p judge.Params) string {
			if p.Raw == judge.Handshake.String() {
				return "Check the HTTPS setup on your proxy."
			}
			return "Renew it on your proxy."
		}},
	judge.FixIdentity: {needs: needs(pHost, pPort), may: needs(pProgram),
		title: func(_ Words, p judge.Params) string {
			return fmt.Sprintf("Something other than %s responds at %s.", subject(p), hostPort(p.Host, p.Port))
		},
		step: func(_ Words, p judge.Params) string {
			return fmt.Sprintf("Check what your router or proxy forwards port %d to.", p.Port)
		}},
	judge.FixSignInServer: {needs: needs(pHost), may: needs(pProgram),
		title: func(_ Words, p judge.Params) string {
			return subject(p) + " apps can't get past your proxy's sign-in page."
		},
		step: func(_ Words, p judge.Params) string {
			return fmt.Sprintf("Let %s's address through without it.", subject(p))
		}},
	judge.FixSignInUI: {needs: needs(pHost),
		title: fixed("Your proxy shows a sign-in page before SlipStream."),
		step:  fixed("SlipStream can't test past it.")},
	judge.FixEchoUnconfirmed: {needs: needs(pTester),
		title: func(w Words, p judge.Params) string {
			return fmt.Sprintf("SlipStream responded to %s's test, but couldn't confirm it came from outside your home.", w.Node(p.Tester))
		},
		step: fixed("Add a Node on another network to test it.")},
	judge.FixNotTested: {
		title: fixed("Nothing outside your home has tested this route."),
		step:  fixed("Add a Node on another network, such as a VPS that isn't your proxy, to test it.")},
}

var facts = map[judge.FactID]sentence{
	judge.FactSplitHorizon: {needs: needs(pHost, pAddr), title: func(_ Words, p judge.Params) string {
		return fmt.Sprintf("On your home network, %s resolves to %s. Outsiders may get a different address.", p.Host, p.Addr)
	}},
	judge.FactTestedThroughProxy: {needs: needs(pTester, pAddr), title: func(w Words, p judge.Params) string {
		return fmt.Sprintf("Tested from %s through your proxy at %s.", w.Node(p.Tester), p.Addr)
	}},
	judge.FactProxyLeg: {needs: needs(pTester, pNode), title: func(w Words, p judge.Params) string {
		return fmt.Sprintf("%s is your proxy, so its test only shows that it can connect to %s.", w.Node(p.Tester), w.Node(p.Node))
	}},
	judge.FactSignInPage: {needs: needs(pHost), title: func(_ Words, p judge.Params) string {
		return p.Host + " shows a sign-in page first."
	}},
	judge.FactIPv6ViewersDirect: {needs: needs(pNode), title: func(w Words, p judge.Params) string {
		return fmt.Sprintf("IPv6 viewers can connect to %s directly.", w.Node(p.Node))
	}},
	judge.FactIPv6SkipsProxy: {needs: needs(pNode, pAddr), title: func(w Words, p judge.Params) string {
		return fmt.Sprintf("IPv6 viewers connect to %s directly and skip your proxy at %s.", w.Node(p.Node), p.Addr)
	}},
	judge.FactAddressChurn: {needs: needs(pNode), title: func(w Words, p judge.Params) string {
		return w.Node(p.Node) + "'s public address keeps changing."
	}},
	judge.FactTesterAway: {needs: needs(pTester), title: func(w Words, p judge.Params) string {
		return w.Node(p.Tester) + " is offline, so it couldn't test this route."
	}},
	judge.FactSameNetwork: {needs: needs(pTester, pNode), title: func(w Words, p judge.Params) string {
		return fmt.Sprintf("%s is on the same network as %s, so its test wouldn't show what players outside see.", w.Node(p.Tester), w.Node(p.Node))
	}},
	judge.FactAtAddress: {needs: needs(pTester, pAddr), title: func(w Words, p judge.Params) string {
		return fmt.Sprintf("%s has the address being tested, %s, so it can't test this route.", w.Node(p.Tester), p.Addr)
	}},
	judge.FactServesIt: {needs: needs(pTester), title: func(w Words, p judge.Params) string {
		return w.Node(p.Tester) + " is part of this route, so it can't test it from outside."
	}},
	judge.FactTesterDNS: {needs: needs(pTester, pHost, pAddr), title: func(w Words, p judge.Params) string {
		return fmt.Sprintf("%s gets %s for %s, so it can't test this route.", w.Node(p.Tester), p.Addr, p.Host)
	}},
	judge.FactClaimContradicted: {needs: needs(pTester, pWant), may: needs(pAddr, pPrefix), title: func(w Words, p judge.Params) string {
		claim := p.Addr.String()
		if p.Prefix.IsValid() {
			claim = p.Prefix.String()
		}
		return fmt.Sprintf("%s reports its public address as %s, but its connection comes from %s, so it can't test this route.", w.Node(p.Tester), claim, p.Want)
	}},
	judge.FactLinkUnconfirmed: {needs: needs(pTester, pReason), title: func(w Words, p judge.Params) string {
		if p.Reason == judge.ReasonOtherFamily {
			return fmt.Sprintf("SlipStream couldn't confirm %s's public address, because its connection uses the other IP version.", w.Node(p.Tester))
		}
		return fmt.Sprintf("SlipStream couldn't confirm %s's public address, because its connection to the Controller is over a private network.", w.Node(p.Tester))
	}},
	judge.FactPlexRelayQuality: {title: fixed("Plex apps outside your home connect through Plex's own relay, which limits video quality.")},
	judge.FactPlexMapping: {may: needs(pRaw), title: func(_ Words, p judge.Params) string {
		if p.Raw == "" {
			return "Plex reports no remote access state."
		}
		return fmt.Sprintf("Plex reports its remote access as %s.", p.Raw)
	}},
	judge.FactPlexTested: {needs: needs(pTester, pAddr, pPort), title: func(w Words, p judge.Params) string {
		return fmt.Sprintf("%s couldn't connect to %s, but Plex's own check from outside passed.", w.Node(p.Tester), netip.AddrPortFrom(p.Addr, p.Port))
	}},
	judge.FactOtherProgram: {needs: needs(pNode, pPort), may: needs(pProgram), title: func(w Words, p judge.Params) string {
		return fmt.Sprintf("%s on %s listens on port %d, so these checks read its socket.", program(p), w.Node(p.Node), p.Port)
	}},
	judge.FactOwnBind: {needs: needs(pNode, pPort), may: needs(pProgram), title: func(w Words, p judge.Params) string {
		return fmt.Sprintf("On %s, %s listens on port %d, and the service behind it listens on this machine only.", w.Node(p.Node), lower(program(p)), p.Port)
	}},
	judge.FactEchoThroughProgram: {needs: needs(pNode, pPort), may: needs(pProgram), title: func(w Words, p judge.Params) string {
		return fmt.Sprintf("The test arrived through %s on %s, so SlipStream can't see which address it came from.", lower(program(p)), w.Node(p.Node))
	}},
	judge.FactEchoOwnStream: {needs: needs(pTester), title: func(w Words, p judge.Params) string {
		return fmt.Sprintf("The test came back through %s's own connection to the Controller, so it doesn't show what players outside see.", w.Node(p.Tester))
	}},
	judge.FactEchoTesterAddress: {needs: needs(pTester, pAddr), title: func(w Words, p judge.Params) string {
		return fmt.Sprintf("The test came from %s, an address on %s itself.", p.Addr, w.Node(p.Tester))
	}},
	judge.FactEchoSource: {needs: needs(pTester, pAddr), title: func(w Words, p judge.Params) string {
		return fmt.Sprintf("The test came from %s, which isn't %s's public address.", p.Addr, w.Node(p.Tester))
	}},
	judge.FactClass: {needs: needs(pNode), may: needs(pClass), title: func(w Words, p judge.Params) string {
		return w.ClassLine(p.Node, p.Class)
	}},
	judge.FactClassFromReaching: {needs: needs(pNode), may: needs(pProgram), title: func(w Words, p judge.Params) string {
		return fmt.Sprintf("%s isn't set to run on a Node, so these facts come from %s, which connects to it.", subject(p), w.Node(p.Node))
	}},
	judge.FactMappingPermanent: {needs: needs(pNode), title: func(w Words, p judge.Params) string {
		return fmt.Sprintf("The router gave %s a permanent port mapping. It stays on the router after %[1]s is removed.", w.Node(p.Node))
	}},
	judge.FactSTUNVariesPort: {needs: needs(pNode), title: func(w Words, p judge.Params) string {
		return fmt.Sprintf("STUN servers see %s on different ports, so its router changes the port for each destination.", w.Node(p.Node))
	}},
	judge.FactSTUNVariesAddress: {needs: needs(pNode), title: func(w Words, p judge.Params) string {
		return fmt.Sprintf("STUN servers see different public addresses for %s.", w.Node(p.Node))
	}},
	judge.FactFirewalldPartial: {needs: needs(pNode, pPort), title: func(w Words, p judge.Params) string {
		return fmt.Sprintf("firewalld on %s allows port %d, and SlipStream can't read the other firewall rules.", w.Node(p.Node), p.Port)
	}},
	judge.FactConnectBack: {needs: needs(pTester, pAddr, pPort), title: func(w Words, p judge.Params) string {
		return fmt.Sprintf("%s couldn't connect to %s either.", w.Node(p.Tester), netip.AddrPortFrom(p.Addr, p.Port))
	}},
}

var meanings = map[judge.CheckID]string{
	judge.CheckDNSA:           "The A record points to this route.",
	judge.CheckDNSAAAA:        "The AAAA record points to this route.",
	judge.CheckOriginBind:     "The program on the port accepts connections from other devices.",
	judge.CheckOriginFirewall: "The firewall allows the port.",
	judge.CheckOriginClass:    "Players outside can connect through the router.",
	judge.CheckOutsideConnect: "A Node outside your home can connect.",
	judge.CheckConnectBack:    "The testing Node can connect to your home's public address.",
	judge.CheckTLSCertificate: "The certificate is for this name and hasn't expired.",
	judge.CheckIdentity:       "The service at the address is the one you chose.",
	judge.CheckHTTPPlain:      "The UI and portal use HTTPS.",
	judge.CheckPlexCustomURL:  "Plex lists the public address.",
	judge.CheckPlexMapping:    "Plex's own check from outside passes.",
}

func fixed(s string) func(Words, judge.Params) string {
	return func(Words, judge.Params) string { return s }
}

// Title is what happened, a sentence that stands alone: Health's text.
func (w Words) Title(f judge.Fix) string { return fixes[f.ID].title(w, f.Params) }

// Step is what to do.
func (w Words) Step(f judge.Fix) string { return fixes[f.ID].step(w, f.Params) }

// Details is the caveat under Details, empty when there is none.
func (w Words) Details(f judge.Fix) string { return fixes[f.ID].details }

// FactText is a fact as a sentence.
func (w Words) FactText(f judge.Fact) string { return facts[f.ID].title(w, f.Params) }

// CheckMeaning is what a check passing means.
func CheckMeaning(id judge.CheckID) string { return meanings[id] }

// Command is the firewall command for a fix that carries one, empty
// otherwise. It names a family only where the tool's rules are per family,
// so never on Windows.
func Command(f judge.Fix) string {
	if !f.Command {
		return ""
	}
	p := f.Params
	port := strconv.Itoa(int(p.Port))
	switch p.Tool {
	case netmodel.WindowsDefender:
		return `New-NetFirewallRule -DisplayName "SlipStream port ` + port + `" -Direction Inbound -Protocol TCP -LocalPort ` + port + ` -Action Allow`
	case netmodel.UFW:
		from := "0.0.0.0/0"
		if p.Family != nil && *p.Family == netmodel.IPv6 {
			from = "::/0"
		}
		return "sudo ufw allow from " + from + " to any port " + port + " proto tcp"
	case netmodel.Firewalld:
		return "sudo firewall-cmd --permanent --add-port=" + port + "/tcp && sudo firewall-cmd --reload"
	case netmodel.SocketFilter:
		return `sudo /usr/libexec/ApplicationFirewall/socketfilterfw --unblockapp "` + p.Program + `"`
	case netmodel.ToolUnknown:
	}
	return ""
}

// ClassLine is a Node's class as one line, in no signal color.
func (w Words) ClassLine(n role.NodeID, c netmodel.Class4) string {
	format := map[netmodel.Class4]string{
		netmodel.Open:          "%s has a public address of its own.",
		netmodel.ThroughVPN:    "%s sends its internet traffic through a VPN.",
		netmodel.CGNAT:         "%s shares a public address with other homes (CGNAT).",
		netmodel.DoubleNAT:     "%s is behind two routers.",
		netmodel.Mappable:      "%s is behind a router that can open ports for SlipStream.",
		netmodel.BehindRouter:  "%s is behind a router that doesn't open ports for SlipStream.",
		netmodel.Class4Unknown: "SlipStream can't tell how %s connects to the internet.",
	}[c]
	return fmt.Sprintf(format, w.Node(n))
}

// Tone is a status part's color.
type Tone string

const (
	ToneOK      Tone = "ok"
	ToneNeutral Tone = "neutral"
	ToneWarn    Tone = "warn"
	ToneFail    Tone = "fail"
)

// StatusPart is one part of the status sentence. Exposure is set on a part
// about one Exposure, so the web links it to its card.
type StatusPart struct {
	Tone     Tone
	Text     string
	Exposure *judge.ExposureID
}

// StatusParts is the status sentence, joined with "·" on screen. subject
// names an Exposure: "SlipStream UI and portal" or the server's name.
func StatusParts(s judge.Status, subject func(judge.ExposureID) string) []StatusPart {
	routes := func(n int) string {
		if n == 1 {
			return "1 route"
		}
		return strconv.Itoa(n) + " routes"
	}
	if len(s.Work)+len(s.Untested)+len(s.Fix)+len(s.Relay) == 0 && len(s.LookRight) > 0 {
		return []StatusPart{
			{Tone: ToneNeutral, Text: routes(len(s.LookRight)) + verb(len(s.LookRight), " looks", " look") + " right from here"},
			{Tone: ToneNeutral, Text: "Not tested from outside"},
		}
	}
	var out []StatusPart
	if n := len(s.Work); n > 0 {
		out = append(out, StatusPart{Tone: ToneOK, Text: routes(n) + verb(n, " works", " work")})
	}
	if n := len(s.LookRight) + len(s.Untested); n > 0 {
		out = append(out, StatusPart{Tone: ToneNeutral, Text: strconv.Itoa(n) + " not tested from outside"})
	}
	for _, g := range []struct {
		ids  []judge.ExposureID
		tone Tone
		what string
	}{{s.Fix, ToneWarn, " a fix"}, {s.Relay, ToneFail, " a relay"}} {
		switch n := len(g.ids); {
		case n == 1:
			out = append(out, StatusPart{Tone: g.tone, Text: subject(g.ids[0]) + " needs" + g.what, Exposure: &g.ids[0]})
		case n > 1:
			out = append(out, StatusPart{Tone: g.tone, Text: routes(n) + " need" + g.what})
		}
	}
	return out
}

func verb(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

// where is a forward's target: the origin's address inside the home, or
// its name when no address was read.
func (w Words) where(p judge.Params) string {
	if p.LocalAddr.IsValid() {
		return p.LocalAddr.String()
	}
	return w.Node(p.Node)
}

// subject is the service a fix names: the Media Server's name, or
// SlipStream itself.
func subject(p judge.Params) string {
	if p.Program == "" {
		return "SlipStream"
	}
	return p.Program
}

// program is the program on a port: its name, or a description when the
// reader couldn't see its owner.
func program(p judge.Params) string {
	if p.Program == "" {
		return fmt.Sprintf("The program on port %d", p.Port)
	}
	return p.Program
}

// lower is program's description inside a sentence.
func lower(s string) string {
	if len(s) > 4 && s[:4] == "The " {
		return "the " + s[4:]
	}
	return s
}

func hostPort(host string, port uint16) string {
	return host + ":" + strconv.Itoa(int(port))
}

func toolName(t netmodel.Tool) string {
	return [...]string{"The firewall", "Windows Defender Firewall", "The macOS firewall", "firewalld", "ufw"}[t]
}

// NotWatched is the class line of a Node nothing on which is checked.
func (w Words) NotWatched(n role.NodeID) string {
	return fmt.Sprintf("Not checked, because nothing on %s is set up for use away from home.", w.Node(n))
}

// NotReadYet is the class line of a Node no check has read yet.
func (w Words) NotReadYet(n role.NodeID) string {
	return w.Node(n) + "'s network hasn't been checked yet."
}

// Skipped is a route whose origin is away.
func (w Words) Skipped(origin role.NodeID, since string) string {
	return fmt.Sprintf("Not checked since %s because %s is offline.", since, w.Node(origin))
}

// TestedBy is who tested a route from outside.
func (w Words) TestedBy(j judge.Judged, tester role.NodeID) string {
	for _, fv := range j.Families {
		if fv.Tested && fv.ByPlex {
			return "Tested by Plex."
		}
	}
	if tester.IsZero() {
		return "Not tested from outside."
	}
	return fmt.Sprintf("Tested from %s.", w.Node(tester))
}

// Families is a route's offered families as its tag.
func Families(offered []netmodel.Family) string {
	switch {
	case len(offered) == 2:
		return "IPv4 and IPv6"
	case len(offered) == 1 && offered[0] == netmodel.IPv6:
		return "IPv6 only"
	}
	return "IPv4"
}

// ActionLabel is a fix's button.
func ActionLabel(k judge.ActionKind) string {
	return map[judge.ActionKind]string{judge.PairNode: "Add a Node", judge.MapPorts: "Turn on Open router ports automatically"}[k]
}
