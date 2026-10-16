import type {
  TasksGroupGroup,
  TasksOpenRun,
  TasksOpenRuns,
  TasksRunResult,
  TasksTask,
  TasksUnitOutcome,
  TasksWaitingState,
} from "../api/gen/model";
import { atWords, inWords, sinceWords } from "../format";
import type { Signal } from "../ui/signal-chip";

export const groupTitles: { readonly [G in TasksGroupGroup]: string } = {
  controller: "Controller",
  acquire: "Acquire",
  store: "Store",
  process: "Process",
  connect: "Connect",
  reach: "Reach",
};

export type Outcome = NonNullable<TasksRunResult> | NonNullable<TasksUnitOutcome>;

/** Skipped and Cancelled raise nothing, so they carry no signal colour. */
export const outcomes: {
  readonly [O in Outcome]: { readonly word: string; readonly signal: Signal | undefined };
} = {
  succeeded: { word: "Succeeded", signal: "ok" },
  partial: { word: "Partial", signal: "warn" },
  failed: { word: "Failed", signal: "fail" },
  skipped: { word: "Skipped", signal: undefined },
  canceled: { word: "Cancelled", signal: undefined },
};

const links: { readonly [L in TasksWaitingState["link"]]: string } = {
  down: "offline",
  mismatched: "on another version",
};

export const waitingWords = (state: TasksWaitingState, now: Date): string =>
  `Waiting for ${state.node} · ${links[state.link]} since ${sinceWords(state.since, now)}`;

export const nextWords = (task: TasksTask, next: string, now: Date): string =>
  task.cadence.inWindow ? atWords(next, now) : inWords(next, now);

export const runningWords = (since: string, now: Date): string =>
  `Running since ${sinceWords(since, now)}`;

export const progressWords = ({ done, total }: { done: number; total: number }): string =>
  `${done} of ${total}`;

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
