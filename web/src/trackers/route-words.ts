import type {
  DownloadclientsClientCard,
  DownloadclientsClientsList,
  NodesNode,
  TrackersChoice,
  TrackersRoute,
  TrackersTarget,
  TrackersTrackerItemPrivacy,
} from "../api/gen/model";

/** The names a route's clients go by: every Node, and every external client with its Node. */
export type ClientNames = {
  readonly node: (id: string) => string;
  readonly external: (
    id: string,
  ) => { readonly name: string; readonly nodeId: string; readonly on: boolean } | undefined;
};

export function clientNames(
  nodes: readonly NodesNode[],
  { cards, orphans }: Pick<DownloadclientsClientsList, "cards" | "orphans">,
): ClientNames {
  const externals = [...cards.flatMap((card) => card.externals), ...orphans];
  return {
    node: (id) => nodes.find((node) => node.id === id)?.name ?? id,
    external: (id) => externals.find((client) => client.id === id),
  };
}

/** "the built-in client on seedbox" or "qBittorrent on den". */
export function targetWords(names: ClientNames, { nodeId, clientId }: TrackersTarget): string {
  const node = names.node(nodeId);
  const client = clientId === undefined ? undefined : names.external(clientId);
  return client === undefined ? `the built-in client on ${node}` : `${client.name} on ${node}`;
}

/** How the built-in client a route names announces itself, while its Node reports it. */
export function announcesWords(
  cards: readonly DownloadclientsClientCard[],
  names: ClientNames,
  to: TrackersTarget | null,
): string | undefined {
  if (to === null || to.clientId !== undefined) {
    return undefined;
  }
  const client = cards.find((card) => card.nodeId === to.nodeId)?.client;
  return (
    client &&
    `The built-in client on ${names.node(to.nodeId)} announces as libtorrent ${client.libtorrent}.`
  );
}

const byRoot = "download where the Work's library root is";

function skippedWords(names: ClientNames, stored: TrackersChoice, then: string): string {
  const client = stored.clientId === undefined ? undefined : names.external(stored.clientId);
  if (client?.on === false) {
    return `${client.name} is off, so these Grabs ${then}.`;
  }
  const node = names.node(client?.nodeId ?? stored.builtInOn ?? "");
  return `${node} no longer holds Downloading, so these Grabs ${then}.`;
}

/** Where a home Node's Grabs go now, and whether the Admin picked it. */
export function routeWords(names: ClientNames, route: TrackersRoute): string {
  const { to, stored, skipped } = route;
  if (to === null) {
    return stored !== null && skipped
      ? skippedWords(names, stored, byRoot)
      : `${names.node(route.homeNodeId)} doesn't hold Downloading, so these Grabs ${byRoot}.`;
  }
  const where = targetWords(names, to);
  if (stored === null) {
    return `These Grabs go to ${where}, the default.`;
  }
  return skipped
    ? skippedWords(names, stored, `go to ${where}, the default`)
    : `These Grabs go to ${where}, which you picked.`;
}

/** Why a Tracker that isn't private has no routes. */
export function unroutedWords(
  name: string,
  privacy: Exclude<TrackersTrackerItemPrivacy, "private">,
): string {
  const words = {
    public: `${name} is public. Its Grabs ${byRoot}.`,
    "semi-private": `${name} is semi-private. Its Grabs ${byRoot}, as a public Tracker's do.`,
    unknown: `SlipStream reads whether ${name} is private from its first torrent. Until then its Grabs ${byRoot}.`,
  } as const;
  return words[privacy];
}
