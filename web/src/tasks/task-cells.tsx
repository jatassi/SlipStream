import { Alert02Icon } from "@hugeicons/core-free-icons";
import { Link } from "@tanstack/react-router";

import type { TasksRun, TasksTask } from "../api/gen/model";
import type { Clock } from "../ui/clock";
import { Data } from "../ui/data";
import { Icon } from "../ui/icon";
import { ProgressCell, percentOf } from "../ui/progress";
import { PlainTime } from "../ui/relative-time";
import { OutcomeChip } from "./outcome-chip";
import { progressWords, runningWords, waitingWords } from "./words";

export type Progress = { done: number; total: number };

export type Live = { clock: Clock; progress: ReadonlyMap<number, Progress> };

export const progressOf = (task: TasksTask, { progress }: Live): Progress | undefined =>
  task.state.kind === "running" ? progress.get(task.state.runId) : undefined;

export function TaskName({ task }: { task: TasksTask }) {
  return (
    <span className="flex min-w-0 flex-col gap-0.5">
      <Link
        to="/tasks/$taskId"
        params={{ taskId: task.id }}
        className="rounded-item font-medium hover:underline"
      >
        {task.name}
      </Link>
      <span className="text-meta text-pretty text-ink-3">{task.about}</span>
    </span>
  );
}

export function LastRun({ run, clock }: { run: TasksRun | undefined; clock: Clock }) {
  if (run === undefined) {
    return <span className="text-ink-3">Not run yet</span>;
  }
  return (
    <span className="flex min-w-0 flex-col items-start gap-1">
      <OutcomeChip outcome={run.result} />
      <span className="max-w-full text-meta text-pretty text-ink-3">
        {run.summary}
        {run.summary !== "" && run.endedAt !== null && " · "}
        {run.endedAt !== null && <PlainTime at={run.endedAt} form="clock" zone={clock.zone} />}
      </span>
    </span>
  );
}

function Running({
  task,
  since,
  progress,
  clock,
}: {
  task: TasksTask;
  since: string;
  progress: Progress;
  clock: Clock;
}) {
  return (
    <ProgressCell
      label={`${task.name} progress`}
      value={percentOf(progress.done, progress.total, 0)}
      motion="live"
      amount={<Data>{progressWords(progress, task.noun)}</Data>}
      note={runningWords(since, clock)}
    />
  );
}

export function TaskState({
  task,
  progress,
  clock,
}: {
  task: TasksTask;
  progress: Progress | undefined;
  clock: Clock;
}) {
  const { state } = task;
  if (state.kind === "running") {
    return <Running task={task} since={state.since} progress={progress ?? state} clock={clock} />;
  }
  if (state.kind === "waiting") {
    return (
      <span className="flex items-center gap-1.5 font-medium text-warn-ink">
        <Icon icon={Alert02Icon} className="size-4 shrink-0" />
        {waitingWords(state, clock)}
      </span>
    );
  }
  return state.off ? (
    <span>Off</span>
  ) : (
    <PlainTime
      at={state.next}
      form={task.cadence.inWindow ? "due" : "countdown"}
      zone={clock.zone}
    />
  );
}
