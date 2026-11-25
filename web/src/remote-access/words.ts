import type {
  ReachabilityCheckID,
  ReachabilityCheckViewResult,
  ReachabilityHopViewKind,
  ReachabilityRouteRoute,
  ReachabilityRouteVerdict,
} from "../api/gen/model";
import type { Defines, Term } from "../ui/defined";

export const hopNames: { readonly [K in ReachabilityHopViewKind]: string } = {
  player: "Player",
  dns: "DNS",
  proxy: "Proxy",
  relay: "Relay",
  tls: "TLS",
  "node-link": "Node link",
  origin: "Origin",
  "media-server": "Media Server",
};

export const verdicts: Defines<ReachabilityRouteVerdict, Term> = {
  reachable: { word: "Reachable", definition: "Players outside your home can connect to it." },
  fixable: {
    word: "Fixable",
    definition: "One change on your side lets players outside your home connect to it.",
  },
  "needs-relay": {
    word: "Needs a relay",
    definition:
      "Your home network can't accept connections from outside, so players need a Node outside your home to forward them.",
  },
};

export const notTested: Term = {
  word: "Not tested",
  definition:
    "No Node outside your home has connected to it yet, so SlipStream can't confirm what players see.",
};

export const routeKinds: Defines<Exclude<ReachabilityRouteRoute, "unplaced">, Term> = {
  direct: { word: "Direct", definition: "Players connect straight to your home." },
  "own-proxy": {
    word: "Through your own proxy",
    definition: "Players connect to a server you run, which forwards them to your home.",
  },
  "through-relay": {
    word: "Through the Relay Node",
    definition: "Players connect to your Relay Node, which forwards them to your home.",
  },
};

export const results: { readonly [R in ReachabilityCheckViewResult]: string } = {
  passes: "Passes",
  fails: "Fails",
  "not-tested": "Not tested from outside",
  "not-used": "Not used",
  "not-reached": "Not reached",
};

export const familyNames = { ipv4: "IPv4", ipv6: "IPv6" } as const;

/** What a check's value under Details holds. */
export const observedLabels: { readonly [C in ReachabilityCheckID]: string } = {
  "dns.a": "A record",
  "dns.aaaa": "AAAA record",
  "origin.bind": "Program on the port",
  "origin.firewall": "Firewall rule",
  "origin.class": "Network",
  "outside.connect": "Address tried",
  "outside.connect-back": "Address tried",
  "tls.certificate": "Names on the certificate",
  identity: "Response",
  "http.plain": "Address",
  "plex.custom-url": "Custom URLs in Plex",
  "plex.mapping": "Public address in Plex",
};
