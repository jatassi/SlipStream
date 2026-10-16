import { getGetNodeQueryKey, getListNodesQueryKey } from "../api/gen/query/nodes/nodes";
import { getGetNodeConfigQueryKey, getGetNodeSystemQueryKey } from "../api/gen/query/system/system";
import { getListOpenTaskRunsQueryKey, type listOpenTaskRuns } from "../api/gen/query/tasks/tasks";
import type { LiveEffects } from "./types";

const everyTasksQuery = {
  predicate: ({ queryKey: [path] }: { queryKey: readonly unknown[] }) =>
    typeof path === "string" && /^\/api\/v1\/tasks(?:\/|$)/u.test(path),
};

export const systemEffects = {
  "nodes.changed": (data, { queryClient }) => {
    void queryClient.invalidateQueries({ queryKey: getListNodesQueryKey() });
    void queryClient.invalidateQueries({ queryKey: getGetNodeQueryKey(data.nodeId) });
    void queryClient.invalidateQueries({ queryKey: getGetNodeSystemQueryKey(data.nodeId) });
  },
  "node.settings-changed": (data, { queryClient }) => {
    void queryClient.invalidateQueries({ queryKey: getGetNodeConfigQueryKey(data.nodeId) });
    void queryClient.invalidateQueries({ queryKey: getGetNodeSystemQueryKey(data.nodeId) });
  },
  "tasks.changed": (_data, { queryClient }) => {
    void queryClient.invalidateQueries(everyTasksQuery);
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
  "nodes.changed" | "node.settings-changed" | "tasks.changed" | "tasks.progress"
>;
