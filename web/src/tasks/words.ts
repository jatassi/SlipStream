import type {
  TasksGroupGroup,
  TasksOpenRun,
  TasksOpenRuns,
  TasksRunResult,
  TasksUnitOutcome,
  TasksWaitingState,
} from "../api/gen/model";
import { type Noun, nounOf } from "../format";
import { waitingLine } from "../nodes/standing";
import type { Clock } from "../ui/clock";
import type { Signal } from "../ui/signal-chip";
import { whenWords } from "../ui/time-words";

export const groupTitles: { readonly [G in TasksGroupGroup]: string } = {
  controller: "Controller",
  acquire: "Acquire",
  store: "Store",
  process: "Process",
  connect: "Connect",
  reach: "Reach",
};

export type Outcome = NonNullable<TasksRunResult> | NonNullable<TasksUnitOutcome>;

export const outcomes: {
  readonly [O in Outcome]: { readonly word: string; readonly signal: Signal | undefined };
} = {
  succeeded: { word: "Succeeded", signal: "ok" },
  partial: { word: "Partial", signal: "warn" },
  failed: { word: "Failed", signal: "fail" },
  skipped: { word: "Skipped", signal: undefined },
  canceled: { word: "Canceled", signal: undefined },
};

export const waitingWords = (state: TasksWaitingState, clock: Clock): string =>
  waitingLine(state.node, state.standing, state.since, clock);

export const runningWords = (since: string, clock: Clock): string =>
  `Running ${whenWords(since, "since", clock)}`;

export const progressWords = (
  { done, total }: { done: number; total: number },
  noun: Noun,
): string => `${done} of ${total} ${nounOf(total, noun)}`;

export const percentOf = ({ done, total }: { done: number; total: number }): number =>
  total === 0 ? 0 : Math.floor((done / total) * 100);

/**
 * The header shows a run an Admin started at once, and a scheduled run once
 * it has been open the header delay, so short runs never blink there.
 */
export function visibleRuns(
  { headerDelaySeconds, runs }: TasksOpenRuns,
  now: Date,
): TasksOpenRun[] {
  return runs.filter(
    (run) =>
      run.trigger === "run-now" ||
      now.getTime() - Date.parse(run.startedAt) >= headerDelaySeconds * 1000,
  );
}
