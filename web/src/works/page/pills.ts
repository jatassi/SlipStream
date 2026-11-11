import type { FilesState, FilesSummary, WorksContentTypeItems } from "../../api/gen/model";
import { stateOrder, statusMarks } from "../../ui/state-marks";

export type PillTone = "met" | "fail" | "live" | "plain";

export type Pill = { readonly tone: PillTone; readonly words: string };

/** What the hero says of the Work's cells or Items: pills first, then the rest in tertiary text. */
export type Pills = {
  readonly pills: readonly Pill[];
  readonly rest: readonly string[];
};

/** The states that stand out as pills. The rest read as tertiary text. */
const pillTones: { readonly [S in FilesState]?: PillTone } = {
  failed: "fail",
  downloading: "live",
};

const order = stateOrder.filter((state) => state !== "met");

const wordsOf = (state: FilesState, count: number) =>
  state === "upgradable" && count !== 1
    ? `${count} upgrades available`
    : `${count} ${statusMarks[state].word.toLowerCase()}`;

/**
 * "3 of 9 met", "1 failed" and "2 downloading", then "2 missing · 1 upgrade available".
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
    const tone = pillTones[state];
    if (tone === undefined) {
      rest.push(wordsOf(state, count));
    } else {
      pills.push({ tone, words: wordsOf(state, count) });
    }
  }
  return { pills, rest };
}
