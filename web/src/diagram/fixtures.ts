import type {
  NodesRoleBoard,
  NodesRoleGroup,
  NodesRoleView,
  NodesTopology,
  NodesTopologyNode,
  NodesTorrentView,
  NodesTraffic,
} from "../api/gen/model";
import type { RoleName } from "../nodes/roles";

// Installs of 1 to 15 Nodes for the harness, the anatomy audit and the
// Diagram's tests: one reading of Topology, the role board and two traffic
// samples a second apart.

const AT = "2026-10-12T21:40:00Z";
const AWAY_SINCE = "2026-10-12T21:31:00Z";

const groups: NodesRoleGroup[] = [
  {
    group: "acquire",
    label: "Acquire",
    roles: [
      { role: "searching", label: "Searching", pinned: false },
      { role: "downloading", label: "Downloading", probe: "engine-starts", pinned: false },
    ],
  },
  {
    group: "store",
    label: "Store",
    roles: [{ role: "storage", label: "Storage", probe: "data-writable", pinned: false }],
  },
  {
    group: "process",
    label: "Process",
    roles: [
      { role: "converting", label: "Converting", pinned: false },
      { role: "subtitles", label: "Subtitles", pinned: false },
    ],
  },
  {
    group: "connect",
    label: "Connect",
    roles: [
      { role: "metadata", label: "Metadata", pinned: false },
      { role: "media-server-link", label: "Media Server link", pinned: false },
      { role: "notifying", label: "Notifying", pinned: false },
    ],
  },
  {
    group: "reach",
    label: "Reach",
    roles: [
      { role: "relaying", label: "Relaying", pinned: false },
      { role: "serving", label: "Serving", probe: "listener-up", pinned: true },
    ],
  },
];

const tableOrder: readonly RoleName[] = groups.flatMap(({ roles }) =>
  roles.map(({ role }) => role),
);

type Seat = {
  readonly name: string;
  readonly roles: readonly RoleName[];
  readonly away?: true;
  readonly draining?: RoleName;
};

const SEATS: readonly Seat[] = [
  {
    name: "home",
    roles: [
      "searching",
      "downloading",
      "storage",
      "converting",
      "metadata",
      "notifying",
      "serving",
    ],
  },
  { name: "seedbox", roles: ["downloading", "subtitles", "media-server-link", "relaying"] },
  { name: "vps", roles: ["relaying", "serving"] },
  { name: "attic", roles: ["storage", "subtitles"], away: true },
  { name: "garage", roles: ["downloading", "converting"], draining: "downloading" },
  { name: "office", roles: ["media-server-link", "metadata"] },
  { name: "cabin", roles: ["storage"] },
  { name: "studio", roles: ["converting", "subtitles", "notifying"] },
  { name: "lab", roles: ["searching"] },
  { name: "den", roles: ["downloading", "storage"] },
  { name: "loft", roles: ["relaying"] },
  { name: "shed", roles: ["subtitles"] },
  { name: "barn", roles: ["storage", "converting"] },
  { name: "mill", roles: ["metadata"] },
  { name: "pier", roles: ["searching", "notifying"] },
];

const idOf = (index: number): string =>
  `00000000-0000-4000-8000-${String(index + 1).padStart(12, "0")}`;

const torrentsOf = (index: number): NodesTorrentView[] =>
  [
    { title: "The Long Walk (2025)", wanted: 7.4e9, done: 0.62 },
    { title: "Severance S02E07", wanted: 2.1e9, done: 0.18 },
  ].map(({ title, wanted, done }, at) => ({
    grabId: `grab-${String(index)}-${String(at)}`,
    title,
    wanted,
    done: Math.round(wanted * done),
    peers: 14 - at * 5,
  }));

function viewOf(seat: Seat, role: RoleName): NodesRoleView {
  if (seat.away === true) {
    return { role, state: "away" };
  }
  if (seat.draining === role) {
    return { role, state: "draining", held: { units: 2, noun: "torrents", bytes: 9.5e9 } };
  }
  return { role, state: "holding" };
}

function nodeOf(seat: Seat, index: number): NodesTopologyNode {
  const roles = tableOrder.filter((role) => seat.roles.includes(role));
  const downloading = roles.includes("downloading") && seat.away !== true;
  return {
    id: idOf(index),
    name: seat.name,
    controller: index === 0,
    self: index === 0,
    state: seat.away === true ? "offline" : "online",
    since: seat.away === true ? AWAY_SINCE : "2026-10-12T09:00:00Z",
    ...(seat.away === true ? { lastSeenAt: AWAY_SINCE } : {}),
    probes: [],
    roles: roles.map((role) => viewOf(seat, role)),
    resources: downloading ? { downloading: { torrents: torrentsOf(index) } } : {},
  };
}

export type Fixture = {
  readonly topology: NodesTopology;
  readonly board: NodesRoleBoard;
  /** Two traffic samples a second apart. */
  readonly traffic: readonly [NodesTraffic, NodesTraffic];
};

function sampleOf(nodes: readonly NodesTopologyNode[], at: string, step: number): NodesTraffic {
  const [controller, ...workers] = nodes;
  return {
    at,
    links: workers
      .filter((node) => node.state === "online")
      .map((node, index) => ({
        a: controller?.id ?? "",
        b: node.id,
        abBps: 40_000 + index * 9_000,
        baBps: 1_800_000 + index * 600_000 + step * 120_000,
        rttMs: 18 + index * 11,
      })),
    nodes: nodes
      .filter((node) => node.resources.downloading !== undefined)
      .map((node) => ({
        id: node.id,
        downBps: 6_200_000 + step * 300_000,
        upBps: 900_000,
        torrents: (node.resources.downloading?.torrents ?? []).map((torrent) => ({
          grabId: torrent.grabId,
          peers: torrent.peers,
          done: Math.min(torrent.wanted, torrent.done + step * 6_200_000),
        })),
      })),
  };
}

function topologyOf(
  controller: NodesTopologyNode,
  workers: readonly NodesTopologyNode[],
): NodesTopology {
  const away = workers.filter((node) => node.state !== "online");
  return {
    at: AT,
    sentence:
      away.length === 0
        ? "Every Node is online."
        : `${away.map(({ name }) => name).join(" and ")} is offline.`,
    nodes: [controller, ...workers],
    links: workers.map((node, index) =>
      node.state === "online"
        ? {
            a: controller.id,
            b: node.id,
            state: "up",
            dialer: index % 3 === 2 ? controller.id : node.id,
            rttMs: 18 + index * 11,
          }
        : { a: controller.id, b: node.id, state: "down" },
    ),
  };
}

const boardOf = (nodes: readonly NodesTopologyNode[]): NodesRoleBoard => ({
  basis: "basis-1",
  groups,
  gestures: [],
  nodes: nodes.map((node) => ({
    id: node.id,
    name: node.name,
    controller: node.controller,
    away: node.state !== "online",
    assigned: node.roles.map(({ role }) => role),
    roles: node.roles,
    probes: [],
    suggested: [],
  })),
});

/** A Controller alone, which holds every Role. */
const LONE: Seat = { name: "home", roles: tableOrder };

/** An install of `count` Nodes: the Controller, then Workers in pairing order. */
export function fixtureOf(count: number): Fixture {
  const seats = count === 1 ? [LONE] : SEATS.slice(0, count);
  const nodes = seats.map((seat, index) => nodeOf(seat, index));
  const [controller, ...workers] = nodes;
  if (controller === undefined) {
    throw new Error("a fixture has at least its Controller");
  }
  return {
    topology: topologyOf(controller, workers),
    board: boardOf(nodes),
    traffic: [sampleOf(nodes, "2026-10-12T21:39:59Z", 0), sampleOf(nodes, AT, 1)],
  };
}

/** The reading of the clock every fixture is drawn at. */
export const fixtureClock = { now: new Date(AT), zone: "UTC" } as const;

export const FIXTURE_SIZES = [1, 2, 5, 8, 12] as const;
