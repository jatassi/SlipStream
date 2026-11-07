import type { IndexersIndexer, IndexersLimitUse, IndexersQuery, NodesNode } from "../api/gen/model";
import { clock, countWords } from "../format";
import type { AlertTone } from "../ui/count";

const kindWords: { readonly [K in Exclude<IndexersIndexer["kind"], "native">]: string } = {
  catalog: "Catalog site",
  torznab: "Torznab feed",
  rss: "RSS feed",
};

/** What kind of Indexer it is in words; a ported site is named by its site. */
export function kindOf(indexer: IndexersIndexer): string {
  if (indexer.port !== null) {
    return indexer.port.name;
  }
  if (indexer.kind === "native") {
    throw new Error("a native Indexer came without its port");
  }
  return kindWords[indexer.kind];
}

const syncedFormat = new Intl.DateTimeFormat("en-GB", {
  day: "numeric",
  month: "short",
  year: "numeric",
  timeZone: "UTC",
});

/** Where a ported site's support comes from: Prowlarr's code as of the commit it was synced to. */
export const portedWords = (name: string, synced: string) =>
  `SlipStream's ${name} support is ported from Prowlarr's code of ${syncedFormat.format(new Date(synced))}.`;

const seconds = { one: "second", many: "seconds" } as const;

/** The least time between two requests to a ported site. */
export const spacingWords = (name: string, spacing: number) =>
  `Requests to ${name} are at least ${countWords(spacing, seconds)} apart.`;

export const nodeName = (nodes: readonly NodesNode[], id: string): string =>
  nodes.find((node) => node.id === id)?.name ?? "a removed Node";

/** The signal an Indexer's sentence carries: a stop fails, a backoff warns. */
export function toneOf(indexer: IndexersIndexer): AlertTone | undefined {
  if (!indexer.on) {
    return undefined;
  }
  if (indexer.waits?.kind === "stopped") {
    return "fail";
  }
  return indexer.waits?.kind === "failing" || indexer.fix !== null ? "warn" : undefined;
}

const per = { hour: "per hour", day: "per day" } as const;

export function limitWords(limit: IndexersLimitUse): string {
  if (limit.count === 0) {
    return "No limit";
  }
  const used = `${limit.used} of ${limit.count} ${per[limit.per]}`;
  return limit.freeAt === null ? used : `${used}, full until ${clock(limit.freeAt)}`;
}

export const useWords: { readonly [U in IndexersQuery["use"]]: string } = {
  search: "Search",
  recent: "Recent items",
  fetch: ".torrent fetch",
  test: "Test",
};

const releases = { one: "Release", many: "Releases" } as const;

const settled: { readonly [O in Exclude<IndexersQuery["outcome"], "ok" | null>]: string } = {
  limited: "Limited",
  failing: "Failed",
  stopped: "Stopped",
  lost: "Lost because its Node never responded",
};

const okWords = ({ use, releases: found }: IndexersQuery): string => {
  if (use === "fetch") {
    return "Fetched";
  }
  if (use === "test") {
    return "Passed";
  }
  return found === 0 ? "No Releases" : `Found ${countWords(found, releases)}`;
};

export function outcomeWords(query: IndexersQuery): string {
  if (query.outcome === null) {
    return "Running";
  }
  return query.outcome === "ok" ? okWords(query) : settled[query.outcome];
}

const indexerNoun = { one: "Indexer", many: "Indexers" } as const;

/** What the list comes to, in a sentence: who needs the Admin first, then who searches ask. */
export function summaryOf(indexers: readonly IndexersIndexer[]): string {
  const needing = indexers.filter((indexer) => indexer.on && indexer.fix !== null);
  if (needing.length === 1) {
    return `${needing[0]?.name} needs you. Open it to see what to do.`;
  }
  if (needing.length > 1) {
    return `${countWords(needing.length, indexerNoun)} need you. Open each to see what to do.`;
  }
  const on = indexers.filter((indexer) => indexer.on);
  if (on.length === 0) {
    return "Every Indexer is off, so searches find nothing. Turn one on to search again.";
  }
  const asked = on.filter((indexer) => indexer.waits === null);
  if (asked.length < on.length) {
    return `Searches ask ${asked.length} of the ${on.length} Indexers that are on. The others are waiting, and each row says why.`;
  }
  return on.length === 1
    ? `Searches ask ${on[0]?.name}.`
    : `Searches ask all ${on.length} Indexers that are on.`;
}
