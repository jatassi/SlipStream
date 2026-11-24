import type {
  DownloadclientsClientCard as Card,
  HealthVolume,
  MovesMoveAsk,
  MovesMovePreview,
  NodesNamed,
  NodesOutcome,
  RootsNodeVolumes,
  RootsRoot,
} from "../api/gen/model";
import { healthOfRoot } from "../roots/storage-block";
import { type Side, firstOther, sideBody } from "./client-move-dialog";

/** The Node a change of Roles drains of Storage or Downloading while it still has their work. */
export type Draining = {
  readonly node: NodesNamed;
  readonly storage: boolean;
  readonly downloading: boolean;
};

export function drainingOf(outcome: NodesOutcome): Draining | undefined {
  let found: Draining | undefined;
  for (const effect of outcome.effects) {
    if (
      effect.kind !== "drains" ||
      effect.held.units === 0 ||
      (effect.role !== "storage" && effect.role !== "downloading")
    ) {
      continue;
    }
    found = {
      node: effect.node,
      storage: (found?.storage ?? false) || effect.role === "storage",
      downloading: (found?.downloading ?? false) || effect.role === "downloading",
    };
  }
  return found;
}

const freeOn = (root: RootsRoot, volumes: readonly RootsNodeVolumes[]) =>
  volumes
    .find(({ nodeId }) => nodeId === root.nodeId)
    ?.volumes.find(({ rootIds }) => rootIds.includes(root.id))?.free ?? 0;

/** The bytes free on root's volume above its Warning level, all of them while Health has no level for it. */
const aboveWarning = (
  root: RootsRoot,
  volumes: readonly RootsNodeVolumes[],
  levelled: readonly HealthVolume[],
) => freeOn(root, volumes) - (healthOfRoot(root, volumes, levelled)?.warnBytes ?? 0);

/** The ready roots of root's Content Type on other Nodes, most free space above the Warning level first. */
const destinationsOf = (root: RootsRoot, listed: Listed) =>
  listed.roots
    .filter(
      (to) =>
        to.contentType === root.contentType && to.nodeId !== root.nodeId && to.state === "ready",
    )
    .toSorted(
      (a, b) =>
        aboveWarning(b, listed.volumes, listed.levelled) -
        aboveWarning(a, listed.volumes, listed.levelled),
    );

export type Listed = {
  readonly roots: readonly RootsRoot[];
  readonly volumes: readonly RootsNodeVolumes[];
  readonly levelled: readonly HealthVolume[];
  readonly cards: readonly Card[];
};

/** The Node's own Download Clients: its built-in one, then each external one. */
const clientsOf = (card: Card | undefined): { side: Side; name: string }[] =>
  card === undefined
    ? []
    : [
        { side: { nodeId: card.nodeId }, name: "Built-in client" },
        ...card.externals.map((client) => ({
          side: { nodeId: card.nodeId, clientId: client.id },
          name: client.name,
        })),
      ];

export const clientKey = ({ clientId }: Side) => clientId ?? "built-in";

export type Picks = {
  readonly roots: Readonly<Record<string, string>>;
  readonly clients: Readonly<Record<string, Side>>;
  readonly deleteEmptied: boolean;
};

export type RootPair = {
  readonly root: RootsRoot;
  readonly options: readonly RootsRoot[];
  readonly to: string | undefined;
};

export type ClientPair = {
  readonly side: Side;
  readonly name: string;
  readonly to: Side | undefined;
};

/**
 * Each source root and client of the drained Node with where it goes: the
 * Admin's pick, else the root with the most free space above its Warning
 * level and the first client on another Node.
 */
export function pairsOf(draining: Draining, picks: Picks, listed: Listed) {
  const { node } = draining;
  const others = listed.cards.filter(({ nodeId }) => nodeId !== node.id);
  const roots: RootPair[] = draining.storage
    ? listed.roots
        .filter(({ nodeId }) => nodeId === node.id)
        .map((root) => {
          const options = destinationsOf(root, listed);
          return { root, options, to: picks.roots[root.id] ?? options[0]?.id };
        })
    : [];
  const clients: ClientPair[] = draining.downloading
    ? clientsOf(listed.cards.find(({ nodeId }) => nodeId === node.id)).map(({ side, name }) => ({
        side,
        name,
        to: picks.clients[clientKey(side)] ?? firstOther(others, side),
      }))
    : [];
  return { roots, clients, others };
}

/** The drain's Move, or undefined while nothing has somewhere to go. */
export function askOf(
  draining: Draining,
  roots: readonly RootPair[],
  clients: readonly ClientPair[],
  deleteEmptied: boolean,
): MovesMoveAsk | undefined {
  const ask = {
    origin: "drain",
    works: [],
    roots: roots.flatMap(({ root, to }) => (to === undefined ? [] : [{ rootId: root.id, to }])),
    clients: clients.flatMap(({ side, to }) =>
      to === undefined ? [] : [{ ...sideBody(side), to: sideBody(to) }],
    ),
    options: {
      deleteEmptiedRoots: draining.storage && deleteEmptied,
      includeLinkedTorrents: draining.downloading,
    },
  } satisfies MovesMoveAsk;
  return ask.roots.length > 0 || ask.clients.length > 0 ? ask : undefined;
}

export const whatWords = ({ storage, downloading }: Draining) => {
  if (storage && downloading) {
    return "library files and torrents";
  }
  return storage ? "library files" : "torrents";
};

export function downloadingWords(plan: MovesMovePreview): string | undefined {
  const n = plan.units.filter((unit) =>
    unit.reasons.some(({ code }) => code === "downloading"),
  ).length;
  if (n === 0) {
    return undefined;
  }
  return n === 1
    ? "1 torrent is still downloading. It moves once it finishes. To move it now, send its Grab to another client."
    : `${n} torrents are still downloading. They move once they finish. To move them now, send their Grabs to another client.`;
}
