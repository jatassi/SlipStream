import {
  getGetHealthSettingsQueryKey,
  getGetSystemHealthQueryKey,
  getListHealthVolumesQueryKey,
} from "../api/gen/query/health/health";
import { getListIndexersQueryKey } from "../api/gen/query/indexers/indexers";
import {
  getGetNodeQueryKey,
  getGetRoleBoardQueryKey,
  getGetTopologyQueryKey,
  getListNodesQueryKey,
} from "../api/gen/query/nodes/nodes";
import { getListRootsQueryKey, getListVolumesQueryKey } from "../api/gen/query/roots/roots";
import { getGetCountsQueryKey } from "../api/gen/query/shell/shell";
import { getGetNodeConfigQueryKey, getGetNodeSystemQueryKey } from "../api/gen/query/system/system";
import { getListOpenTaskRunsQueryKey, type listOpenTaskRuns } from "../api/gen/query/tasks/tasks";
import { everyJoinCodeQuery } from "../nodes/join-codes";
import { takeTraffic } from "./traffic";
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
    refresh({ queryKey: getGetTopologyQueryKey() });
    refresh({ queryKey: getListRootsQueryKey() });
    refresh({ queryKey: getListVolumesQueryKey() });
    refresh(everyJoinCodeQuery);
  },
  // Samples feed the Diagram alone, so they go to its ring, not a query.
  "nodes.traffic": (data) => {
    takeTraffic(data);
  },
  "node.settings-changed": (data, { refresh }) => {
    refresh({ queryKey: getGetNodeConfigQueryKey(data.nodeId) });
    refresh({ queryKey: getGetNodeSystemQueryKey(data.nodeId) });
  },
  "tasks.changed": (_data, { refresh }) => {
    refresh(everyTasksQuery);
  },
  "health.changed": (data, { refresh }) => {
    refresh({ queryKey: getGetSystemHealthQueryKey() });
    refresh({ queryKey: getGetCountsQueryKey() });
    refresh({ queryKey: getListHealthVolumesQueryKey() });
    if (data.categories.includes("storage")) {
      refresh({ queryKey: getGetHealthSettingsQueryKey() });
    }
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
  | "nodes.changed"
  | "nodes.traffic"
  | "node.settings-changed"
  | "tasks.changed"
  | "health.changed"
  | "tasks.progress"
>;
