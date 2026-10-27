import type { IndexersIndexer, IndexersLimitUse, IndexersQuery, NodesNode } from "../api/gen/model";
import { clock, countWords } from "../format";
import type { AlertTone } from "../ui/count";

export const kindWords: { readonly [K in IndexersIndexer["kind"]]: string } = {
  catalog: "Catalog site",
  torznab: "Torznab feed",
  rss: "RSS feed",
};

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
