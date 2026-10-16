import { createFileRoute } from "@tanstack/react-router";

import {
  getGetTaskSuspenseQueryOptions,
  getListOpenTaskRunsSuspenseQueryOptions,
  getListTasksSuspenseQueryOptions,
} from "../../../api/gen/query/tasks/tasks";
import { ensureQuery } from "../../../api/query-client";
import { loaderTitle } from "../../../shell/titles";
import { TaskDetail } from "../../../tasks/task-detail";
import { Skeleton } from "../../../ui/skeleton";

export const Route = createFileRoute("/_app/tasks/$taskId")({
  staticData: { title: loaderTitle },
  loader: async ({ context, params: { taskId } }) => {
    const [{ task }] = await Promise.all([
      ensureQuery(context.queryClient, getGetTaskSuspenseQueryOptions(taskId)),
      ensureQuery(context.queryClient, getListTasksSuspenseQueryOptions()),
      ensureQuery(context.queryClient, getListOpenTaskRunsSuspenseQueryOptions()),
    ]);
    return { title: task.name };
  },
  pendingComponent: () => <Skeleton label="Loading the task" />,
  component: TaskPage,
});

function TaskPage() {
  return <TaskDetail taskId={Route.useParams().taskId} />;
}
