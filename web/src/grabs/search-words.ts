import type {
  GrabsBehind,
  GrabsGrab,
  GrabsGroupState,
  GrabsGroupStateLastResult,
  GrabsGroupStateState,
  GrabsSearch,
  GrabsSearchQueued,
  GrabsSearchTrigger,
  GrabsStopped,
  GrabsWorkSearching,
} from "../api/gen/model";
import {
  type Clock,
  type Noun,
  countWords,
  isToday,
  laterAtWords,
  laterWords,
  pastAtWords,
} from "../format";

const releases: Noun = { one: "release", many: "releases" };
const works: Noun = { one: "Work", many: "Works" };
const grabWord: Noun = { one: "Grab", many: "Grabs" };
const days: Noun = { one: "day", many: "days" };
const dayMs = 86_400_000;
const list = new Intl.ListFormat("en", { type: "conjunction" });

export type Trigger = NonNullable<GrabsSearchTrigger>;

/** What started a run, as the Searches list names it. */
export const triggerWords: Readonly<Record<Trigger, string>> = {
  search: "Search",
  search_all: "Search all",
  add: "Added",
  monitor: "Monitoring turned on",
  profile: "Profile changed",
  editions: "Editions changed",
  retarget: "Target changed",
  release_date: "Release date",
  refresh: "Refresh",
  backlog: "Backlog search",
  failed_grab: "After a failed Grab",
  rss: "RSS sync",
};

type Outcome = NonNullable<GrabsSearch["outcome"]>;
type Counted = Pick<GrabsSearch, "grabs" | "candidates">;

/** How a run finished, as the Searches list says it. */
export const outcomeWords: Readonly<Record<Outcome, (run: Counted) => string>> = {
  grabbed: ({ grabs }) => `Grabbed ${countWords(grabs, releases)}`,
  waiting: () => "Waiting for a better release",
  nothing_found: () => "Found nothing",
  nothing_acceptable: ({ candidates }) =>
    `Found ${countWords(candidates, releases)}, but none fits a Target`,
  no_indexer: () => "No Indexer could search",
  no_root: () => "No library root",
  listed: () => "Listed releases",
};

/** What found a Grab's Release, for Grab Details' "Found by". */
export const foundByWords: Readonly<
  Record<GrabsGrab["foundBy"], (grab: Pick<GrabsGrab, "rssIndexerName">) => string>
> = {
  search: () => "Search",
  backlog: () => "Backlog search",
  rss: ({ rssIndexerName }) =>
    rssIndexerName === null ? "RSS sync" : `RSS sync from ${rssIndexerName}`,
  failed_grab: () => "Search again after a failed Grab",
  manual: () => "Manual search",
};

/** A group's state word for its level's header; an idle group shows none. */
export const groupWords: Readonly<Record<GrabsGroupStateState, string | undefined>> = {
  idle: undefined,
  queued: "Queued",
  searching: "Searching",
};

/**
 * What holds a group, as its popover says it: a queued group's wait to
 * start, or a searching group's paced line waiting at an Indexer.
 * Undefined when nothing does.
 */
export function behindWords(behind: GrabsBehind | null, clock: Clock): string | undefined {
  if (behind === null) {
    return undefined;
  }
  if (behind.indexerName !== null && behind.starved) {
    return `This search waits for ${behind.indexerName}. RSS sync uses all of ${behind.indexerName}'s search limit.`;
  }
  if (behind.indexerName !== null && behind.until !== null) {
    return `Waiting for ${behind.indexerName}'s search limit until ${laterWords(behind.until, clock).words}.`;
  }
  if (behind.ahead === 0) {
    return undefined;
  }
  const from = behind.aheadFrom === null ? "" : ` from ${triggerWords[behind.aheadFrom]}`;
  return `Queued behind ${countWords(behind.ahead, works)}${from}.`;
}

/** "Waiting for a better release until 22:40" */
export const waitingWords = (until: string, clock: Clock): string =>
  `Waiting for a better release until ${laterWords(until, clock).words}`;

/** A waiting cell's fact: how many Releases it holds and when it grabs the best. */
export function waitWords(candidates: number, until: string, clock: Clock): string {
  return `${countWords(candidates, releases)} found so far. SlipStream grabs the best one ${laterAtWords(until, clock)} unless a better one arrives.`;
}

const nextWords = (at: string | null, clock: Clock): string =>
  at === null ? "" : ` Next search ${laterAtWords(at, clock)}.`;

export type LastResult = NonNullable<GrabsGroupStateLastResult>;

type FoundNothing = Extract<LastResult, "nothing_found" | "nothing_acceptable">;

/** What a group's last search found when it grabbed nothing. */
const foundWords: Readonly<Record<FoundNothing, (group: GrabsGroupState, clock: Clock) => string>> =
  {
    nothing_found: ({ lastSearchedAt }, clock) =>
      lastSearchedAt === null
        ? "Found nothing."
        : `Found nothing ${pastAtWords(lastSearchedAt, clock)}.`,
    nothing_acceptable: ({ lastFound }) =>
      `Found ${countWords(lastFound, releases)}, but none fits a Target.`,
  };

/** A group's last search, as the hero says it. */
export const lastWords: Readonly<
  Record<LastResult, (group: GrabsGroupState, clock: Clock) => string>
> = {
  grabbed: ({ lastGrabs }) => `Grabbed ${countWords(lastGrabs, releases)}`,
  waiting: () => "Waiting for a better release",
  nothing_found: (group, clock) =>
    `${foundWords.nothing_found(group, clock)}${nextWords(group.nextSearchAt, clock)}`,
  nothing_acceptable: (group, clock) =>
    `${foundWords.nothing_acceptable(group, clock)}${nextWords(group.nextSearchAt, clock)}`,
  no_indexer: () => "No Indexer could search. Check Indexers in Health.",
  no_root: () => "No library root was reachable. Check Library roots in Settings.",
};

/** The hero's one line about what SlipStream is doing for the Work. */
export type Lead =
  | { readonly kind: "searching"; readonly words: string; readonly behind: string | undefined }
  | { readonly kind: "queued"; readonly words: string; readonly behind: string | undefined }
  | { readonly kind: "waiting"; readonly words: string }
  | { readonly kind: "result"; readonly words: string; readonly result: LastResult };

function latest(groups: readonly GrabsGroupState[]): GrabsGroupState | undefined {
  return groups
    .filter((group) => group.lastResult !== null && group.lastSearchedAt !== null)
    .toSorted((a, b) => (b.lastSearchedAt ?? "").localeCompare(a.lastSearchedAt ?? ""))
    .at(0);
}

/**
 * The hero's line: the group the server leads with while one searches or
 * is queued, else the Work's wait, else its latest result. Nothing before
 * the first search. `many` is a Work of many Items, whose groups are named.
 */
export function leadWords(
  state: GrabsWorkSearching,
  clock: Clock,
  many: boolean,
): Lead | undefined {
  const { lead, groups } = state;
  const led = groups.find(({ key }) => key === lead.groupKey);
  if (led?.state === "searching") {
    const of = lead.of > 1 ? ` of ${lead.of}` : "";
    return {
      kind: "searching",
      words: many ? `Searching ${led.label}${of}` : "Searching",
      behind: behindWords(led.behind, clock),
    };
  }
  if (led?.state === "queued") {
    return { kind: "queued", words: "Queued", behind: behindWords(led.behind, clock) };
  }
  if (lead.wait !== null) {
    return { kind: "waiting", words: waitingWords(lead.wait.until, clock) };
  }
  const last = latest(groups);
  const result = last?.lastResult ?? null;
  if (last === undefined || result === null) {
    return undefined;
  }
  return { kind: "result", words: lastWords[result](last, clock), result };
}

/**
 * The toast after a press of Search. `searched` names what was pressed
 * ("12 episodes", "Dune: Part Two"). `now` is the Work's state read after
 * the press: a group held by a limit or a queue ahead of it is queued, one
 * nothing holds starts at once.
 */
export function queuedToast(
  answer: GrabsSearchQueued,
  now: GrabsWorkSearching,
  searched: string,
): string {
  const queued = answer.groups.filter(({ state }) => state === "queued");
  if (queued.length === 0) {
    return `Already searching ${list.format(answer.groups.map(({ label }) => label))}`;
  }
  const held = queued.every(({ key }) => {
    const group = now.groups.find((candidate) => candidate.key === key);
    return group?.state === "queued" && group.behind !== null;
  });
  return held ? `Queued ${searched} for search.` : `Searching ${searched}`;
}

/** A stopped cell's card: how many Grabs failed over how long, and when Backlog search looks again. */
export function stoppedWords(stop: GrabsStopped, clock: Clock): string {
  const first = stop.failures.at(0);
  const last = stop.failures.at(-1);
  const span =
    first === undefined || last === undefined ? 0 : Date.parse(last.at) - Date.parse(first.at);
  const over =
    first !== undefined && isToday(first.at, clock)
      ? "today"
      : `in ${countWords(Math.max(1, Math.ceil(span / dayMs)), days)}`;
  const failed = `${countWords(stop.failures.length, grabWord)} failed for this file ${over}.`;
  return stop.nextSearchAt === null
    ? `${failed} Backlog search is off, so SlipStream searches for it only when you press Search again.`
    : `${failed} Backlog search looks for it again ${laterAtWords(stop.nextSearchAt, clock)}.`;
}

/** A Not found row's detail: what its group's last search found, as the hero says it, and when the next one runs. */
export function notFoundWords(group: GrabsGroupState, clock: Clock): string {
  const found = foundWords[
    group.lastResult === "nothing_acceptable" ? "nothing_acceptable" : "nothing_found"
  ](group, clock);
  return group.nextSearchAt === null
    ? found
    : `${found} SlipStream searches again ${laterAtWords(group.nextSearchAt, clock)}.`;
}
