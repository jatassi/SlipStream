import type { QueryClient } from "@tanstack/react-query";

import { getListIndexersQueryKey } from "../api/gen/query/indexers/indexers";
import {
  getGetNodeQueryKey,
  getGetRoleBoardQueryKey,
  getGetTopologyQueryKey,
  getListNodesQueryKey,
} from "../api/gen/query/nodes/nodes";
import { getGetCountsQueryKey } from "../api/gen/query/shell/shell";
import { getGetNodeConfigQueryKey, getGetNodeSystemQueryKey } from "../api/gen/query/system/system";
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
    void queryClient.invalidateQueries({ queryKey: getGetNodeSystemQueryKey(data.nodeId) });
    void queryClient.invalidateQueries({ queryKey: getGetRoleBoardQueryKey() });
    void queryClient.invalidateQueries({ queryKey: getGetCountsQueryKey() });
    void queryClient.invalidateQueries({ queryKey: getListIndexersQueryKey() });
    void queryClient.invalidateQueries({ queryKey: getGetTopologyQueryKey() });
  },
  // Samples feed the Diagram alone, so no query caches them.
  "nodes.traffic": () => {},
  "node.settings-changed": (data, { queryClient }) => {
    void queryClient.invalidateQueries({ queryKey: getGetNodeConfigQueryKey(data.nodeId) });
    void queryClient.invalidateQueries({ queryKey: getGetNodeSystemQueryKey(data.nodeId) });
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
} satisfies Pick<
  LiveEffects,
  "nodes.changed" | "nodes.traffic" | "node.settings-changed" | "tasks.changed" | "tasks.progress"
>;
