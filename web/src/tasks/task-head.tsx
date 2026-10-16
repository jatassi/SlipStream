import type { TasksTask } from "../api/gen/model";
import { type Clock, sinceWords } from "../format";
import { type StatusPart, StatusSentence } from "../ui/status-sentence";
import { type Progress, TaskState } from "./task-cells";
import { nextWords, outcomes, runningWords, waitingWords } from "./words";

export const unitsAnchor = "units";
export const runsAnchor = "runs";

function lastPart(task: TasksTask, clock: Clock): StatusPart {
  const { result, endedAt } = task.lastRun ?? { result: null, endedAt: null };
  if (result === null || endedAt === null) {
    return { kind: "plain", text: "Not run yet" };
  }
  const { word, signal } = outcomes[result];
  const text = `${word} at ${sinceWords(endedAt, clock)}`;
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
  const next = nextWords(task, state.next, clock);
  return { kind: "plain", text: `Next run ${next.charAt(0).toLowerCase()}${next.slice(1)}` };
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
