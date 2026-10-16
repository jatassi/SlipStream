import { useCancelTaskRun, useRunTask } from "../api/gen/query/tasks/tasks";
import { isApiError } from "../api/errors";
import type { TasksTask } from "../api/gen/model";
import { Button, type ButtonSize } from "../ui/button";
import { toast } from "../ui/toast";

const failed = (error: Error) => {
  toast(isApiError(error) ? (error.detail ?? error.title) : error.message, { kind: "fail" });
};

export function CancelRun({
  runId,
  name,
  size = "small",
}: {
  runId: number;
  name: string;
  size?: ButtonSize;
}) {
  const cancel = useCancelTaskRun({ mutation: { onError: failed } });
  return (
    <Button
      size={size}
      busy={cancel.isPending}
      aria-label={`Cancel ${name}`}
      onClick={() => {
        cancel.mutate({ runId });
      }}
    >
      Cancel
    </Button>
  );
}

function RunNow({ task, size }: { task: TasksTask; size: ButtonSize }) {
  const run = useRunTask({ mutation: { onError: failed } });
  return (
    <Button
      size={size}
      busy={run.isPending}
      disabled={!task.runNow.available}
      aria-label={`Run now: ${task.name}`}
      onClick={() => {
        run.mutate({ taskId: task.id });
      }}
    >
      Run now
    </Button>
  );
}

export function TaskAction({ task, size = "small" }: { task: TasksTask; size?: ButtonSize }) {
  return task.state.kind === "running" ? (
    <CancelRun runId={task.state.runId} name={task.name} size={size} />
  ) : (
    <RunNow task={task} size={size} />
  );
}
