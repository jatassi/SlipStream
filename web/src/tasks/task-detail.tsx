import { useGetTaskSuspense, useListOpenTaskRunsSuspense } from "../api/gen/query/tasks/tasks";
import { Page } from "../shell/page";
import { TaskAction } from "./task-actions";
import { TaskHead } from "./task-head";
import { TaskRuns } from "./task-runs";
import { TaskTuning } from "./task-tuning";
import { TaskUnits } from "./task-units";
import { useNow } from "./use-now";

const wordsTick = 15_000;

export function TaskDetail({ taskId }: { taskId: string }) {
  const { task, units, runs } = useGetTaskSuspense(taskId).data;
  const open = useListOpenTaskRunsSuspense().data.runs;
  const now = useNow(wordsTick);
  const runId = task.state.kind === "running" ? task.state.runId : undefined;
  return (
    <Page action={<TaskAction task={task} size="default" />}>
      <TaskHead task={task} progress={open.find((run) => run.runId === runId)} now={now} />
      <TaskTuning task={task} />
      <TaskUnits units={units} now={now} />
      <TaskRuns runs={runs} now={now} />
    </Page>
  );
}
