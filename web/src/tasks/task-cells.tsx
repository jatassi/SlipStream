import { Alert02Icon } from "@hugeicons/core-free-icons";
import { Link } from "@tanstack/react-router";

import type { TasksRun, TasksTask } from "../api/gen/model";
import { sinceWords } from "../format";
import { Data } from "../ui/data";
import { Icon } from "../ui/icon";
import { LiveBar } from "../ui/live-bar";
import { OutcomeChip } from "./outcome-chip";
import { nextWords, percentOf, progressWords, runningWords, waitingWords } from "./words";

export type Progress = { done: number; total: number };

export type Live = { now: Date; progress: ReadonlyMap<number, Progress> };

export const progressOf = (task: TasksTask, { progress }: Live): Progress | undefined =>
  task.state.kind === "running" ? progress.get(task.state.runId) : undefined;

export function TaskName({ task }: { task: TasksTask }) {
  return (
    <span className="flex min-w-0 flex-col gap-0.5">
      <Link
        to="/tasks/$taskId"
        params={{ taskId: task.id }}
        className="truncate rounded-item font-medium hover:underline"
      >
        {task.name}
      </Link>
      <span className="truncate text-meta text-ink-3">{task.about}</span>
    </span>
  );
}

export function LastRun({ run, now }: { run: TasksRun | undefined; now: Date }) {
  if (run === undefined) {
    return <span className="text-ink-3">Not run yet</span>;
  }
  const ended = run.endedAt === null ? [] : [sinceWords(run.endedAt, now)];
  return (
    <span className="flex min-w-0 flex-col items-start gap-1">
      <OutcomeChip outcome={run.result} />
      <span className="max-w-full truncate text-meta text-ink-3">
        {[run.summary, ...ended].filter((part) => part !== "").join(" · ")}
      </span>
    </span>
  );
}

function Running({
  task,
  since,
  progress,
  now,
}: {
  task: TasksTask;
  since: string;
  progress: Progress;
  now: Date;
}) {
  return (
    <span className="flex max-w-[240px] min-w-0 flex-col gap-1.5">
      <span className="flex items-baseline justify-between gap-3 text-meta text-ink-3">
        <span className="truncate">{runningWords(since, now)}</span>
        <Data className="text-ink-1">{progressWords(progress)}</Data>
      </span>
      <LiveBar label={`${task.name} progress`} value={percentOf(progress)} />
    </span>
  );
}

export function TaskState({
  task,
  progress,
  now,
}: {
  task: TasksTask;
  progress: Progress | undefined;
  now: Date;
}) {
  const { state } = task;
  if (state.kind === "running") {
    return <Running task={task} since={state.since} progress={progress ?? state} now={now} />;
  }
  if (state.kind === "waiting") {
    return (
      <span className="flex items-center gap-1.5 font-medium text-warn-ink">
        <Icon icon={Alert02Icon} className="size-4 shrink-0" />
        {waitingWords(state, now)}
      </span>
    );
  }
  return <span>{nextWords(task, state.next, now)}</span>;
}
