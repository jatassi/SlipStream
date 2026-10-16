import type { QueryClient } from "@tanstack/react-query";

import { getGetNodeQueryKey, getListNodesQueryKey } from "../api/gen/query/nodes/nodes";
import { getListOpenTaskRunsQueryKey, type listOpenTaskRuns } from "../api/gen/query/tasks/tasks";
import type { LiveEffects } from "./types";

const everyTasksQuery = {
  predicate: ({ queryKey: [path] }: { queryKey: readonly unknown[] }) =>
    typeof path === "string" && /^\/api\/v1\/tasks(?:\/|$)/u.test(path),
};

const refreshing = new WeakMap<QueryClient, { again: boolean }>();

async function refreshTasks(queryClient: QueryClient): Promise<void> {
  const inFlight = refreshing.get(queryClient);
  if (inFlight !== undefined) {
    inFlight.again = true;
    return;
  }
  const slot = { again: false };
  refreshing.set(queryClient, slot);
  await queryClient.invalidateQueries(everyTasksQuery);
  refreshing.delete(queryClient);
  if (slot.again) {
    await refreshTasks(queryClient);
  }
}

export const systemEffects = {
  "nodes.changed": (data, { queryClient }) => {
    void queryClient.invalidateQueries({ queryKey: getListNodesQueryKey() });
    void queryClient.invalidateQueries({ queryKey: getGetNodeQueryKey(data.nodeId) });
  },
  "tasks.changed": (_data, { queryClient }) => {
    void refreshTasks(queryClient);
  },
  "tasks.progress": (data, { queryClient }) => {
    queryClient.setQueryData<Awaited<ReturnType<typeof listOpenTaskRuns>>>(
      getListOpenTaskRunsQueryKey(),
      (open) =>
        open && {
          ...open,
          runs: open.runs.map((run) =>
            run.runId === data.runId ? { ...run, done: data.done, total: data.total } : run,
          ),
        },
    );
  },
} satisfies Pick<LiveEffects, "nodes.changed" | "tasks.changed" | "tasks.progress">;
