import type { FilesState, FilesStateCounts } from "../api/gen/model";
import { notMonitored } from "./state-mark";
import { type StateMark, stateOrder, statusMarks } from "./state-marks";

/** One Item's place in the strip: its state, or quiet when nobody monitors it or it has nothing to download. */
export type Slot = FilesState | "quiet";

/** A run of Items in one slot, from the Item at `from`. */
export type Segment = { readonly slot: Slot; readonly span: number; readonly from: number };

/** What each segment stands for: one episode, a run of episodes in order, or every episode of a state. */
export type Strip = {
  readonly by: "episode" | "run" | "count";
  readonly segments: readonly Segment[];
};

export type StripLook = "header" | "edge" | "poster";

const looks: {
  readonly [L in StripLook]: {
    readonly fit: number;
    readonly box: string;
    readonly segment: string;
  };
} = {
  header: { fit: 45, box: "h-2 w-45 shrink-0 rounded-[2px] bg-fill", segment: "rounded-[2px]" },
  edge: { fit: 86, box: "h-0.75 w-full rounded-[1px] bg-fill", segment: "rounded-[1px]" },
  poster: { fit: 26, box: "h-1 w-full rounded-[1px] bg-well", segment: "rounded-[1px]" },
};

const gaps: { readonly [B in Strip["by"]]: string } = {
  episode: "gap-px",
  run: "gap-0.5",
  count: "gap-0.5",
};

const progressOrder: readonly Slot[] = [
  "met",
  "upgradable",
  "downloading",
  "missing",
  "failed",
  "unreleased",
  "quiet",
];

const slotLooks: { readonly [S in Slot]: Pick<StateMark, "word" | "bar"> } = {
  ...statusMarks,
  quiet: { word: notMonitored.word, bar: "" },
};

function runsOf(segments: readonly Segment[]): Segment[] {
  const runs: Segment[] = [];
  for (const segment of segments) {
    const last = runs.at(-1);
    if (last?.slot === segment.slot) {
      runs[runs.length - 1] = { ...last, span: last.span + segment.span };
    } else {
      runs.push(segment);
    }
  }
  return runs;
}

function countsOf(segments: readonly Segment[]): Segment[] {
  const totals = new Map<Slot, number>();
  for (const { slot, span } of segments) {
    totals.set(slot, (totals.get(slot) ?? 0) + span);
  }
  let from = 0;
  return progressOrder.flatMap((slot) => {
    const span = totals.get(slot) ?? 0;
    from += span;
    return span === 0 ? [] : [{ slot, span, from: from - span }];
  });
}

/**
 * The strip for Items in order, given as runs: one segment per Item while
 * they fit the look, else one per run of equal slots, else one per slot in
 * progress order, so the strip always reads left to right.
 */
export function segmentsOf(runs: readonly Segment[], look: StripLook): Strip {
  const { fit } = looks[look];
  const items = runs.reduce((sum, { span }) => sum + span, 0);
  if (items <= fit) {
    return {
      by: "episode",
      segments: runs.flatMap(({ slot, span, from }) =>
        Array.from({ length: span }, (_, index) => ({ slot, span: 1, from: from + index })),
      ),
    };
  }
  const merged = runsOf(runs);
  return merged.length <= fit
    ? { by: "run", segments: merged }
    : { by: "count", segments: countsOf(merged) };
}

/** Items in their order as runs of one, the input segmentsOf takes. */
export const inOrder = (slots: readonly Slot[]): Segment[] =>
  slots.map((slot, from) => ({ slot, span: 1, from }));

/** A rollup's counts as runs in progress order, so Met grows from the left as a progress bar does. */
export const byCounts = (counts: FilesStateCounts): Segment[] =>
  countsOf(stateOrder.map((state) => ({ slot: state, span: counts[state] ?? 0, from: 0 })));

/** "1 Failed, 3 Missing, 6 Met", the strip's accessible name, in the state order. */
export function stripWords({ segments }: Strip): string {
  const totals = countsOf(segments);
  const ordered = [...stateOrder, "quiet" as const].flatMap((slot) => {
    const span = totals.find((segment) => segment.slot === slot)?.span;
    return span === undefined ? [] : [`${span} ${slotLooks[slot].word}`];
  });
  return ordered.length === 0 ? "Nothing to download" : ordered.join(", ");
}

/**
 * A group's Items as one strip, each segment coloured by its state. A strip
 * the words beside it already name is hidden from screen readers.
 */
export function EpisodeStrip({
  strip,
  look,
  named,
}: {
  strip: Strip;
  look: StripLook;
  named: boolean;
}) {
  const { box, segment } = looks[look];
  const name = named
    ? ({ role: "img", "aria-label": stripWords(strip) } as const)
    : ({ "aria-hidden": true } as const);
  return (
    <span {...name} className={`flex overflow-hidden ${gaps[strip.by]} ${box}`}>
      {strip.segments.map(({ slot, span, from }) => (
        <span
          key={from}
          style={{ flexGrow: span }}
          className={`relative basis-0 ${segment} ${slotLooks[slot].bar}`}
        />
      ))}
    </span>
  );
}
