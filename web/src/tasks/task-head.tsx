import type { TasksTask } from "../api/gen/model";
import type { Clock } from "../ui/clock";
import { type StatusPart, StatusSentence } from "../ui/status-sentence";
import { whenWords } from "../ui/time-words";
import { type Progress, TaskState } from "./task-cells";
import { outcomes, runningWords, waitingWords } from "./words";

export const unitsAnchor = "units";
export const runsAnchor = "runs";

function lastPart(task: TasksTask, clock: Clock): StatusPart {
  const { result, endedAt } = task.lastRun ?? { result: null, endedAt: null };
  if (result === null || endedAt === null) {
    return { kind: "plain", text: "Not run yet" };
  }
  const { word, signal } = outcomes[result];
  const text = `${word} ${whenWords(endedAt, "at", clock)}`;
  if (signal === "fail") {
    return { kind: "failure", text, href: `#${runsAnchor}` };
  }
  return signal === "warn"
    ? { kind: "warning", text, href: `#${runsAnchor}` }
    : { kind: "plain", text };
}

function statePart(task: TasksTask, clock: Clock): StatusPart {
  const { state } = task;
  if (state.kind === "running") {
    return { kind: "live", text: runningWords(state.since, clock) };
  }
  if (state.kind === "waiting") {
    return { kind: "warning", text: waitingWords(state, clock), href: `#${unitsAnchor}` };
  }
  if (state.off) {
    return { kind: "plain", text: "Off" };
  }
  const next = whenWords(state.next, task.cadence.inWindow ? "next" : "in", clock);
  return { kind: "plain", text: `Next run ${next}` };
}

export function TaskHead({
  task,
  progress,
  clock,
}: {
  task: TasksTask;
  progress: Progress | undefined;
  clock: Clock;
}) {
  return (
    <div className="mb-6 flex flex-col gap-2">
      <StatusSentence parts={[lastPart(task, clock), statePart(task, clock)]} />
      <p className="max-w-[64ch] text-(length:--body-size) text-ink-3">{task.about}</p>
      {task.state.kind === "running" && (
        <div className="max-w-[420px]">
          <TaskState task={task} progress={progress} clock={clock} />
        </div>
      )}
    </div>
  );
}
