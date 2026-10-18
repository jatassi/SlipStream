import { getListIndexersQueryKey } from "../api/gen/query/indexers/indexers";
import {
  getGetNodeQueryKey,
  getGetRoleBoardQueryKey,
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

export const systemEffects = {
  "nodes.changed": (data, { refresh }) => {
    refresh({ queryKey: getListNodesQueryKey() });
    refresh({ queryKey: getGetNodeQueryKey(data.nodeId) });
    refresh({ queryKey: getGetNodeSystemQueryKey(data.nodeId) });
    refresh({ queryKey: getGetRoleBoardQueryKey() });
    refresh({ queryKey: getGetCountsQueryKey() });
    refresh({ queryKey: getListIndexersQueryKey() });
  },
  "node.settings-changed": (data, { refresh }) => {
    refresh({ queryKey: getGetNodeConfigQueryKey(data.nodeId) });
    refresh({ queryKey: getGetNodeSystemQueryKey(data.nodeId) });
  },
  "tasks.changed": (_data, { refresh }) => {
    refresh(everyTasksQuery);
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
