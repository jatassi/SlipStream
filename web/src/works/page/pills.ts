import type { FilesState, FilesSummary, WorksContentTypeItems } from "../../api/gen/model";

export type PillTone = "met" | "fail" | "live" | "plain";

export type Pill = { readonly tone: PillTone; readonly words: string };

/** What the hero says of the Work's cells or Items: pills first, then the rest in tertiary text. */
export type Pills = {
  readonly pills: readonly Pill[];
  readonly rest: readonly string[];
};

type Counted =
  | {
      readonly place: "pill";
      readonly tone: PillTone;
      readonly words: (count: number) => string;
    }
  | { readonly place: "rest"; readonly words: (count: number) => string };

const counted: { readonly [S in Exclude<FilesState, "met">]: Counted } = {
  failed: { place: "pill", tone: "fail", words: (count) => `${count} failed` },
  downloading: {
    place: "pill",
    tone: "live",
    words: (count) => `${count} in flight`,
  },
  upgradable: {
    place: "rest",
    words: (count) => `${count} ${count === 1 ? "upgrade" : "upgrades"} available`,
  },
  missing: { place: "rest", words: (count) => `${count} missing` },
  unreleased: { place: "rest", words: (count) => `${count} unreleased` },
};

const order = [
  "failed",
  "downloading",
  "upgradable",
  "missing",
  "unreleased",
] as const satisfies readonly (keyof typeof counted)[];

/**
 * "3 of 9 met", "1 failed" and "2 in flight", then "1 upgrade available · 2 missing".
 * One Item counts cells. Many Items count the Items that count, as the poster does.
 */
export function pillsOf(
  { cells, itemStatus, items }: FilesSummary,
  holds: WorksContentTypeItems,
): Pills {
  const counts = holds === "many" ? itemStatus : cells;
  const met = counts.met ?? 0;
  const total =
    holds === "many" ? items : order.reduce((sum, state) => sum + (counts[state] ?? 0), met);
  if (total === 0) {
    return { pills: [], rest: [] };
  }
  const pills: Pill[] = [
    { tone: met === total ? "met" : "plain", words: `${met} of ${total} met` },
  ];
  const rest: string[] = [];
  for (const state of order) {
    const count = counts[state];
    if (count === undefined) {
      continue;
    }
    const entry = counted[state];
    if (entry.place === "pill") {
      pills.push({ tone: entry.tone, words: entry.words(count) });
    } else {
      rest.push(entry.words(count));
    }
  }
  return { pills, rest };
}
