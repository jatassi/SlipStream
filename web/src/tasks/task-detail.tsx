import {
  useGetTaskSuspense,
  useListOpenTaskRunsSuspense,
  useListTasksSuspense,
} from "../api/gen/query/tasks/tasks";
import { Page } from "../shell/page";
import { TaskAction } from "./task-actions";
import { TaskHead } from "./task-head";
import { TaskRuns } from "./task-runs";
import { TaskTuning } from "./task-tuning";
import { TaskUnits } from "./task-units";
import { useClock } from "./use-now";

const wordsTick = 15_000;

export function TaskDetail({ taskId }: { taskId: string }) {
  const { task, units, runs, tuning } = useGetTaskSuspense(taskId).data;
  const open = useListOpenTaskRunsSuspense().data.runs;
  const clock = useClock(wordsTick, useListTasksSuspense().data.settings.window.zone);
  const runId = task.state.kind === "running" ? task.state.runId : undefined;
  return (
    <Page action={<TaskAction task={task} size="default" />}>
      <TaskHead task={task} progress={open.find((run) => run.runId === runId)} clock={clock} />
      <TaskTuning task={task} tuning={tuning} />
      <TaskUnits units={units} clock={clock} />
      <TaskRuns runs={runs} clock={clock} />
    </Page>
  );
}
