import type {
  NodesRoleBoard,
  NodesRoleView,
  NodesRoleViewState,
  NodesStorageResource,
  NodesTopology,
  NodesTopologyLinkState,
  NodesTopologyNode,
  NodesTorrentView,
} from "../../../../api/gen/model";
import type { Clock } from "../../../../ui/clock";
import { whenWords } from "../../../../ui/time-words";
import { dialOf, reachedBy } from "../../../../nodes/dial";
import {
  type Cluster,
  type RoleName,
  type Standing,
  labelOf,
  roleLine,
} from "../../../../nodes/roles";
import { type NodeState, nodeStates } from "../../../../nodes/state";
import { type Finding, findingsOf } from "../-findings";

export type RoleChip = {
  readonly role: RoleName;
  readonly label: string;
  readonly state: NodesRoleViewState;
  /** What the Role is doing, when that says more than the Node's own state. */
  readonly note: string;
  readonly words: string;
};

/** What giving a Node a Role also means, for a Role that means more than its work. */
export const roleTrust: Partial<Record<RoleName, string>> = {
  converting:
    "A Node with this Role reads the library files it converts and writes the files made from them.",
  notifying:
    "A Node with this Role receives your notification connections' secrets to send messages.",
};

/** One Node as the Roster lists it and the Diagram draws it. */
export type NodeRow = {
  readonly id: string;
  readonly name: string;
  readonly controller: boolean;
  readonly state: NodeState;
  /** Its state in words, then how it is reached. */
  readonly facts: readonly string[];
  /** When it was last seen, while it is away. */
  readonly seenAt: string | undefined;
  readonly roles: readonly RoleChip[];
  readonly findings: readonly Finding[];
  /** It holds work for a Role, which can be written off. */
  readonly holds: boolean;
  /** Its state as its tile reads it: "Online", or "Last seen 3 hours ago" while away. */
  readonly plaque: string;
  /** The torrents its Downloading keeps, when it holds Downloading. */
  readonly torrents: readonly NodesTorrentView[];
  /** Its library roots and their volumes while it stands in Storage, as last reported while away. */
  readonly storage: NodesStorageResource | undefined;
  /** Its listener for browsers is up, as far as its last probe found. */
  readonly listening: boolean;
  /** A Media Server it reaches responds, as its Media Server link last reported. */
  readonly television: boolean;
  /** How many Conversions it runs, as the latest conversions.live sample said. */
  readonly converting: number;
};

/** How much of a torrent is down, as a whole percent. */
export const shareDone = ({ done, wanted }: NodesTorrentView): string =>
  `${String(wanted > 0 ? Math.floor((done / wanted) * 100) : 0)}%`;

/** The Node link between the Controller and one Worker. */
export type LinkRow = {
  readonly worker: string;
  readonly state: NodesTopologyLinkState;
  /** Which end dialed the live session; absent while the link is down. */
  readonly dialer?: "controller" | "worker";
  readonly rttMs?: number;
};

export type Landing = {
  readonly sentence: string;
  readonly controller: string;
  /** The Controller, then the Workers in pairing order. */
  readonly nodes: readonly NodeRow[];
  readonly links: readonly LinkRow[];
};

const standingOf = (node: NodesTopologyNode): Standing => ({
  standing: node.state === "online" ? "in-step" : node.state,
  since: node.since,
});

// The Node's own state says it is away, so a Role's chip only adds what the
// Role itself is doing.
const quiet = (view: NodesRoleView): boolean =>
  view.state === "away" ||
  view.state === "lapsed" ||
  (view.state === "holding" && view.held === undefined);

/** A Node's Roles as chips: each label, and what the Role is doing when that says more than the Node's own state. */
export const roleChips = (
  cluster: Cluster,
  board: NodesRoleBoard,
  roles: readonly NodesRoleView[],
  standing: Standing,
  clock: Clock,
): RoleChip[] =>
  roles.map((view) => {
    const label = labelOf(board, view.role);
    const note = quiet(view) ? "" : roleLine(cluster, view, standing, clock);
    return {
      role: view.role,
      label,
      state: view.state,
      note,
      words: note === "" ? label : `${label} · ${note}`,
    };
  });

// A Worker whose link is down says only that it is away.
function reachOf(topology: NodesTopology, controller: string, node: NodesTopologyNode): string[] {
  if (node.controller) {
    return ["Controller"];
  }
  const by = dialOf(topology, node.id);
  if (by === undefined) {
    return [];
  }
  const rtt = topology.links.find(({ b }) => b === node.id)?.rttMs;
  const reached = reachedBy(controller, { name: node.name, dial: { by } });
  return [rtt === undefined ? reached : `${reached} · ${rtt} ms round trip`];
}

function rowOf(
  topology: NodesTopology,
  board: NodesRoleBoard,
  controller: string,
  state: NodeState,
  node: NodesTopologyNode,
  converting: number,
  clock: Clock,
): NodeRow {
  const { lastSeenAt } = node;
  return {
    id: node.id,
    name: node.name,
    controller: node.controller,
    state,
    facts: [nodeStates[state].word, ...reachOf(topology, controller, node)],
    seenAt: lastSeenAt,
    roles: roleChips(topology, board, node.roles, standingOf(node), clock),
    findings: findingsOf(node.probes),
    holds: node.roles.some(({ held }) => held !== undefined),
    plaque:
      lastSeenAt === undefined
        ? nodeStates[state].word
        : `Last seen ${whenWords(lastSeenAt, "ago", clock)}`,
    torrents: node.resources.downloading?.torrents ?? [],
    storage: node.resources.storage,
    listening: !node.probes.some(({ probe, fit }) => probe === "listener-up" && fit === "fails"),
    television: node.television,
    converting,
  };
}

/** The one reading of Topology that the Roster lists and the Diagram draws. */
export function landingOf(
  topology: NodesTopology,
  board: NodesRoleBoard,
  restarting: ReadonlySet<string>,
  running: ReadonlyMap<string, number>,
  clock: Clock,
): Landing {
  const controller = topology.nodes.find((node) => node.controller);
  if (controller === undefined) {
    throw new Error("Topology has no Controller");
  }
  const ordered = [controller, ...topology.nodes.filter((node) => !node.controller)];
  return {
    sentence: topology.sentence,
    controller: controller.name,
    nodes: ordered.map((node) =>
      rowOf(
        topology,
        board,
        controller.name,
        restarting.has(node.id) ? "restarting" : node.state,
        node,
        running.get(node.id) ?? 0,
        clock,
      ),
    ),
    links: topology.links.map((link) => ({
      worker: link.b,
      state: link.state,
      ...(link.dialer === undefined
        ? {}
        : { dialer: link.dialer === link.a ? "controller" : "worker" }),
      ...(link.rttMs === undefined ? {} : { rttMs: link.rttMs }),
    })),
  };
}
