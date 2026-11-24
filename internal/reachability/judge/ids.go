package judge

import "slices"

// CheckID, FixID and FactID are closed sets. Each has words in
// reachability/words.go, and a registry test fails on an id without them.
// The judge emits ids and Params, never sentences.
type (
	CheckID string
	FixID   string
	FactID  string
)

const (
	CheckDNSA           CheckID = "dns.a"
	CheckDNSAAAA        CheckID = "dns.aaaa"
	CheckOriginBind     CheckID = "origin.bind"
	CheckOriginFirewall CheckID = "origin.firewall"
	CheckOriginClass    CheckID = "origin.class"
	CheckOutsideConnect CheckID = "outside.connect"
	CheckConnectBack    CheckID = "outside.connect-back"
	CheckTLSCertificate CheckID = "tls.certificate"
	CheckIdentity       CheckID = "identity"
	CheckHTTPPlain      CheckID = "http.plain"
	CheckPlexCustomURL  CheckID = "plex.custom-url"
	CheckPlexMapping    CheckID = "plex.mapping"
)

var CheckIDs = []CheckID{
	CheckDNSA, CheckDNSAAAA, CheckOriginBind, CheckOriginFirewall, CheckOriginClass, CheckOutsideConnect,
	CheckConnectBack, CheckTLSCertificate, CheckIdentity, CheckHTTPPlain, CheckPlexCustomURL, CheckPlexMapping,
}

const (
	FixForwardNotSeen  FixID = "forward-not-seen"
	FixManualForward   FixID = "manual-forward"
	FixNoGateway       FixID = "no-gateway"
	FixMappable        FixID = "mappable"
	FixMappablePlex    FixID = "mappable-plex"
	FixCGNAT           FixID = "cgnat"
	FixDoubleNAT       FixID = "double-nat"
	FixVPN             FixID = "vpn"
	FixFirewall        FixID = "firewall"
	FixFirewallUnknown FixID = "firewall-unknown"
	FixFirewallDocker  FixID = "firewall-docker"
	FixBind            FixID = "bind"
	FixBindProgram     FixID = "bind-program"
	FixBindFamily      FixID = "bind-family"
	FixBindUnknown     FixID = "bind-unknown"
	FixPlainHTTP       FixID = "plain-http"
	FixPrivateAddress  FixID = "private-address"
	FixStaleA          FixID = "stale-a"
	FixExtraA          FixID = "extra-a"
	FixPrivateA        FixID = "private-a"
	FixDNSFailed       FixID = "dns-failed"
	FixNoA             FixID = "no-a"
	FixAAAAOffOrigin   FixID = "aaaa-off-origin"
	FixPrefixChange    FixID = "prefix-change"
	FixAAAAFailed      FixID = "aaaa-failed"
	FixPrivateAAAA     FixID = "private-aaaa"
	FixBrokenAAAA      FixID = "broken-aaaa"
	FixUntestedIPv6    FixID = "untested-ipv6"
	FixProxyPort       FixID = "proxy-port"
	FixProxyForward    FixID = "proxy-forward"
	FixCustomURL       FixID = "custom-url"
	FixCertificate     FixID = "certificate"
	FixIdentity        FixID = "identity"
	FixSignInServer    FixID = "sign-in-server"
	FixSignInUI        FixID = "sign-in-ui"
	FixEchoUnconfirmed FixID = "echo-unconfirmed"
	FixNotTested       FixID = "not-tested"
)

var FixIDs = []FixID{
	FixForwardNotSeen, FixManualForward, FixNoGateway, FixMappable, FixMappablePlex, FixCGNAT, FixDoubleNAT,
	FixVPN, FixFirewall, FixFirewallUnknown, FixFirewallDocker, FixBind, FixBindProgram, FixBindFamily,
	FixBindUnknown, FixPlainHTTP, FixPrivateAddress, FixStaleA, FixExtraA, FixPrivateA, FixDNSFailed, FixNoA,
	FixAAAAOffOrigin, FixPrefixChange, FixAAAAFailed, FixPrivateAAAA, FixBrokenAAAA, FixUntestedIPv6,
	FixProxyPort, FixProxyForward, FixCustomURL, FixCertificate, FixIdentity, FixSignInServer,
	FixSignInUI, FixEchoUnconfirmed, FixNotTested,
}

const (
	FactSplitHorizon       FactID = "split-horizon"
	FactTestedThroughProxy FactID = "tested-through-proxy"
	FactProxyLeg           FactID = "proxy-leg"
	FactSignInPage         FactID = "sign-in-page"
	FactIPv6ViewersDirect  FactID = "ipv6-viewers-direct"
	FactIPv6SkipsProxy     FactID = "ipv6-skips-proxy"
	FactAddressChurn       FactID = "address-churn"
	FactTesterAway         FactID = "tester-away"
	FactSameNetwork        FactID = "same-network"
	FactAtAddress          FactID = "at-address"
	FactServesIt           FactID = "serves-it"
	FactTesterDNS          FactID = "tester-dns"
	FactClaimContradicted  FactID = "claim-contradicted"
	FactLinkUnconfirmed    FactID = "link-unconfirmed"
	FactPlexRelayQuality   FactID = "plex-relay-quality"
	FactPlexMapping        FactID = "plex-mapping"
	FactPlexTested         FactID = "plex-tested"
	FactOtherProgram       FactID = "other-program"
	FactOwnBind            FactID = "own-bind"
	FactEchoThroughProgram FactID = "echo-through-program"
	FactEchoOwnStream      FactID = "echo-own-stream"
	FactEchoTesterAddress  FactID = "echo-tester-address"
	FactEchoSource         FactID = "echo-source"
	FactClass              FactID = "class"
	FactClassFromReaching  FactID = "class-from-reaching"
	FactMappingPermanent   FactID = "mapping-permanent"
	FactSTUNVariesPort     FactID = "stun-varies-port"
	FactSTUNVariesAddress  FactID = "stun-varies-address"
	FactFirewalldPartial   FactID = "firewalld-partial"
	FactConnectBack        FactID = "connect-back"
)

var FactIDs = []FactID{
	FactSplitHorizon, FactTestedThroughProxy, FactProxyLeg, FactSignInPage, FactIPv6ViewersDirect,
	FactIPv6SkipsProxy, FactAddressChurn, FactTesterAway, FactSameNetwork, FactAtAddress, FactServesIt,
	FactTesterDNS, FactClaimContradicted, FactLinkUnconfirmed, FactPlexRelayQuality, FactPlexMapping,
	FactPlexTested, FactOtherProgram, FactOwnBind, FactEchoThroughProgram, FactEchoOwnStream,
	FactEchoTesterAddress, FactEchoSource, FactClass, FactClassFromReaching, FactMappingPermanent,
	FactSTUNVariesPort, FactSTUNVariesAddress, FactFirewalldPartial, FactConnectBack,
}

// Link-unconfirmed reasons, Params.Reason.
const (
	ReasonPrivate     = "private"
	ReasonOtherFamily = "other-family"
)

func (id CheckID) Valid() bool { return slices.Contains(CheckIDs, id) }
func (id FixID) Valid() bool   { return slices.Contains(FixIDs, id) }
func (id FactID) Valid() bool  { return slices.Contains(FactIDs, id) }
