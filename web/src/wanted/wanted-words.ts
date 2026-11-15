import type {
  GrabsItemRange,
  GrabsLimitUntil,
  GrabsQueueStatus,
  GrabsWantedRow,
  GrabsWantedRowReason,
  ListWantedSort,
} from "../api/gen/model";
import { type Clock, type Noun, atWords, countWords, pastAtWords, sinceWords } from "../format";
import type { StatusPart } from "../ui/status-sentence";

export type Reason = GrabsWantedRowReason;

export type Sort = ListWantedSort;

/** Every reason in the order a row picks its first. */
export const reasons: readonly Reason[] = ["failed", "missing", "upgrade", "gone"];

/** What Wanted lists until the Admin filters: every reason but File gone. */
export const defaultReasons: readonly Reason[] = ["failed", "missing", "upgrade"];

export const reasonLabels: { readonly [R in Reason]: string } = {
  failed: "Failed",
  missing: "Missing",
  upgrade: "Upgrade available",
  gone: "File gone",
};

const otherNouns: { readonly [R in Reason]: Noun } = {
  failed: { one: "failed", many: "failed" },
  missing: { one: "missing", many: "missing" },
  upgrade: { one: "upgrade", many: "upgrades" },
  gone: { one: "file gone", many: "files gone" },
};

export const sorts: readonly Sort[] = ["next", "last", "title", "released"];

export const sortLabels: { readonly [S in Sort]: string } = {
  next: "Next search",
  last: "Last search",
  title: "Title",
  released: "Release date",
};

export const works: Noun = { one: "Work", many: "Works" };

const counted = new Intl.NumberFormat("en");

/** A count of Works with its thousands grouped, such as "1,204 Works". */
export const worksWords = (count: number): string =>
  `${counted.format(count)} ${count === 1 ? works.one : works.many}`;

const possessive = (name: string): string => (name.endsWith("s") ? `${name}'` : `${name}'s`);

function limitPart(limit: GrabsLimitUntil, clock: Clock): StatusPart {
  const href = `/settings/indexers/${limit.indexerId}`;
  return limit.starved
    ? {
        kind: "warning",
        text: `RSS sync uses all of ${possessive(limit.indexerName)} search limit`,
        href,
      }
    : {
        kind: "warning",
        text: `Waiting for ${possessive(limit.indexerName)} search limit until ${sinceWords(limit.until, clock)}`,
        href,
      };
}

/** What the queue is doing, as the Status sentence's parts. */
export function statusParts(status: GrabsQueueStatus, clock: Clock): StatusPart[] {
  const busy: StatusPart[] = [
    ...(status.searching > 0
      ? [{ kind: "live" as const, text: `${counted.format(status.searching)} searching` }]
      : []),
    ...(status.queued > 0
      ? [{ kind: "plain" as const, text: `${counted.format(status.queued)} queued` }]
      : []),
  ];
  return [
    ...(busy.length === 0 ? [{ kind: "plain" as const, text: "Nothing queued" }] : busy),
    {
      kind: "plain",
      text:
        status.nextBacklogAt === null
          ? "Backlog search is off"
          : `Next Backlog search ${atWords(status.nextBacklogAt, clock)}`,
    },
    ...status.limits.map((limit) => limitPart(limit, clock)),
  ];
}

/** The row's reason, then the count of Items under each other reason: "Missing. 3 upgrades." */
export function reasonWords(row: GrabsWantedRow): string {
  const others = reasons.flatMap((reason) => {
    const count = row.others[reason] ?? 0;
    return reason === row.reason || count === 0
      ? []
      : [`${countWords(count, otherNouns[reason])}.`];
  });
  return [`${reasonLabels[row.reason]}.`, ...others].join(" ");
}

/** The Items a row lists, as runs: "S02E01 to S02E08, S03E02". */
export const rangesWords = (ranges: readonly GrabsItemRange[]): string =>
  ranges.map(({ first, last }) => (first === last ? first : `${first} to ${last}`)).join(", ");

const resultWords: {
  readonly [R in NonNullable<GrabsWantedRow["lastResult"]>]: (when: string) => string;
} = {
  grabbed: (when) => `Grabbed ${when}`,
  waiting: () => "Waiting for a better Release",
  nothing_found: (when) => `Found nothing ${when}`,
  nothing_acceptable: (when) => `Found nothing that fits a Target ${when}`,
  no_indexer: (when) => `No Indexer could search ${when}`,
  no_root: (when) => `Couldn't search ${when} because the Work has no library root`,
};

/** How the last search went and when: "Found nothing on Mon at 09:12". */
export function lastWords(row: GrabsWantedRow, clock: Clock): string {
  if (row.lastResult === null || row.lastSearchedAt === null) {
    return "Not searched yet";
  }
  return resultWords[row.lastResult](pastAtWords(row.lastSearchedAt, clock));
}

/** When SlipStream searches the row next: "Tomorrow at 14:00", "Queued", "Only when you search". */
export function nextWords(row: GrabsWantedRow, clock: Clock): string {
  if (row.state === "searching") {
    return "Searching";
  }
  if (row.state === "queued") {
    return "Queued";
  }
  return row.nextSearchAt === null ? "Only when you search" : atWords(row.nextSearchAt, clock);
}

/** A phone row's second line: "Missing. S02E01 to S02E08. Next search Tomorrow at 14:00." */
export function phoneLine(row: GrabsWantedRow, clock: Clock): string {
  const items = rangesWords(row.ranges);
  const next =
    row.state === "idle" && row.nextSearchAt !== null
      ? `Next search ${nextWords(row, clock)}.`
      : `${nextWords(row, clock)}.`;
  return [`${reasonLabels[row.reason]}.`, ...(items === "" ? [] : [`${items}.`]), next].join(" ");
}
