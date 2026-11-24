import { HttpResponse, http } from "msw";

import type {
  ReachabilityCard,
  ReachabilityNodeNetwork,
  ReachabilityRemoteAccess,
  ReachabilityRoute,
  ReachabilitySuggestionView,
  SettingField,
} from "../api/gen/model";
import { homeId } from "./nodes";

const plexId = "0199f3c0-3333-7000-8000-000000000001";
const uiId = "0199f3c0-3333-7000-8000-000000000002";
export const livingRoomPlexId = "0199f3c0-2222-7000-8000-000000000002";

export const nothingSetUp: ReachabilityRemoteAccess = {
  nothingWanted: true,
  status: { counts: { work: 0, lookRight: 0, untested: 0, fix: 0, relay: 0 }, parts: [] },
  card: null,
  routes: [],
  lastRun: null,
  suggestions: [],
  nodes: [],
};

export const plexSuggestion: ReachabilitySuggestionView = {
  kind: "media-server",
  serverId: livingRoomPlexId,
  subject: "Living room Plex",
  public: "https://plex.example.org",
  addressPrivate: false,
};

export const uiSuggestion: ReachabilitySuggestionView = {
  kind: "ui-portal",
  subject: "SlipStream UI and portal",
  public: "https://slipstream.example.org",
  addressPrivate: false,
};

const forward = {
  id: "manual-forward",
  title: "Port 32400 is closed at your router.",
  step: "Forward TCP port 32400 to 192.168.1.20, port 32400.",
  action: null,
} as const;

/** Living room Plex Direct on home, its forward missing: Fixable on a failing check. */
export const plexRoute: ReachabilityRoute = {
  exposure: {
    id: plexId,
    kind: "media-server",
    subject: "Living room Plex",
    public: "https://plex.example.org",
  },
  checked: true,
  route: "direct",
  families: "IPv4",
  verdict: "fixable",
  tested: true,
  folded: false,
  card: "warn",
  fix: forward,
  skipped: null,
  testedBy: "Tested from seedbox.",
  judgedAt: "2026-11-24T21:38:00Z",
  facts: [],
  hops: [
    { kind: "player", worst: "passes", checks: [], facts: [] },
    {
      kind: "dns",
      worst: "passes",
      facts: [],
      checks: [
        {
          id: "dns.a",
          result: "passes",
          fact: false,
          meaning: "The A record points at your home.",
          fix: null,
          by: "home",
        },
      ],
    },
    {
      kind: "origin",
      worst: "fails",
      facts: [],
      checks: [
        {
          id: "outside.connect",
          family: "ipv4",
          result: "fails",
          points: "fixable",
          fact: false,
          meaning: "A Node outside your home tried to connect to the public address.",
          fix: forward,
          observed: "Connection refused",
          by: "seedbox",
        },
      ],
    },
    { kind: "media-server", worst: "not-reached", checks: [], facts: [] },
  ],
};

/** The UI and portal through the Admin's own proxy, working and tested: one folded line. */
export const uiRoute: ReachabilityRoute = {
  ...plexRoute,
  exposure: {
    id: uiId,
    kind: "ui-portal",
    subject: "SlipStream UI and portal",
    public: "https://slipstream.example.org",
  },
  route: "own-proxy",
  verdict: "reachable",
  folded: true,
  card: "none",
  fix: null,
  hops: [],
};

export const cards = {
  fail: {
    kind: "fail",
    exposureId: plexId,
    subject: "Living room Plex",
    verdict: "needs-relay",
    fix: {
      id: "cgnat",
      title:
        "Your provider shares one public IPv4 address between many homes (CGNAT), so players outside can't connect to home directly.",
      step: "Ask your provider for a public IPv4 address, or put Living room Plex behind your own proxy on a VPS.",
      action: null,
    },
  },
  warn: {
    kind: "warn",
    exposureId: plexId,
    subject: "Living room Plex",
    verdict: "fixable",
    fix: {
      id: "mappable",
      title: "Your router can open port 32400 for SlipStream, and SlipStream hasn't opened it.",
      step: "Turn on Open router ports automatically for home.",
      action: {
        kind: "map-ports",
        nodeId: homeId,
        label: "Turn on Open router ports automatically",
      },
    },
  },
  untested: {
    kind: "untested",
    exposureId: plexId,
    subject: "Living room Plex",
    verdict: "fixable",
    fix: {
      id: "not-tested",
      title: "Nothing outside your home has tested this route.",
      step: "Add a Node on another network, such as a VPS that isn't your proxy, to test it.",
      action: { kind: "pair-node", label: "Add a Node" },
    },
  },
} as const satisfies Record<string, ReachabilityCard>;

/** Two routes set up, Plex's card the one given. */
export function wanted(card: ReachabilityCard | null = cards.warn): ReachabilityRemoteAccess {
  return {
    nothingWanted: false,
    status: {
      counts: { work: 1, lookRight: 0, untested: 0, fix: 1, relay: 0 },
      parts: [
        { tone: "ok", text: "1 route works" },
        { tone: "warn", text: "Living room Plex needs a fix", exposureId: plexId },
      ],
    },
    card,
    routes: [uiRoute, plexRoute],
    lastRun: { at: "2026-11-24T21:38:00Z", testers: [{ nodeId: homeId, name: "seedbox" }] },
    suggestions: [],
    nodes: [
      {
        nodeId: homeId,
        name: "home",
        line: "home is behind a router that doesn't open ports for SlipStream.",
      },
    ],
  };
}

/**
 * The Remote access API: the page as given, and each write it receives. The
 * page holds the routes the test set; a write doesn't change it.
 */
export function remoteAccessServer(page: ReachabilityRemoteAccess = nothingSetUp) {
  const sent: { readonly what: string; readonly body?: unknown }[] = [];
  const record =
    (what: string) =>
    async ({ request }: { request: Request }) => {
      const text = await request.text();
      sent.push(text === "" ? { what } : { what, body: JSON.parse(text) });
    };
  return {
    sent,
    handlers: [
      http.get("*/api/v1/remote-access", () => HttpResponse.json(page)),
      http.post("*/api/v1/remote-access/exposures", async (info) => {
        await record("add")(info);
        return HttpResponse.json(
          { id: plexId, kind: "media-server", subject: "Living room Plex", public: "" },
          { status: 201 },
        );
      }),
      http.post("*/api/v1/remote-access/run", async (info) => {
        await record("run")(info);
        return HttpResponse.json({ runId: 1 });
      }),
      http.patch("*/api/v1/remote-access/nodes/:id/settings", async (info) => {
        await record(`node ${String(info.params["id"])}`)(info);
        return HttpResponse.json({ settings: [] });
      }),
    ],
  };
}

const about = { source: "database", applies: "live", pendingRestart: false } as const;

export const nodeSettings: SettingField[] = [
  {
    ...about,
    kind: "switch",
    key: "reachability.mapPorts",
    label: "Open router ports automatically",
    value: false,
    running: false,
    default: false,
    isDefault: true,
  },
];

export const homeNetwork: ReachabilityNodeNetwork = {
  nodeId: homeId,
  name: "home",
  line: "home is behind a router that can open ports for SlipStream.",
  now: true,
  observedAt: "2026-11-24T21:38:00Z",
  interfaces: [{ name: "eth0", addresses: ["192.168.1.20"], tunnel: false }],
  listeners: [{ port: 8484, ipv4: "yes", ipv6: "no", loopback: false, program: "slipstream" }],
  firewall: [{ port: 8484, family: "ipv4", policy: "allows" }],
  gateway: { address: "192.168.1.1", wan: "203.0.113.9" },
  mappings: [],
};

/** A Node's Network section and its settings, each settings write recorded. */
export function networkServer(network: ReachabilityNodeNetwork = homeNetwork) {
  const sent: unknown[] = [];
  return {
    sent,
    handlers: [
      http.get("*/api/v1/remote-access/nodes/:id", () => HttpResponse.json(network)),
      http.get("*/api/v1/remote-access/nodes/:id/settings", () =>
        HttpResponse.json({ settings: nodeSettings }),
      ),
      http.patch("*/api/v1/remote-access/nodes/:id/settings", async ({ request }) => {
        sent.push(await request.json());
        return HttpResponse.json({ settings: nodeSettings });
      }),
    ],
  };
}
