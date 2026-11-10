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
import { everyJoinCodeQuery } from "../nodes/join-codes";
import { takeProgress } from "./task-progress";
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
  "roots.changed": (_data, { refresh }) => {
    refresh({ queryKey: getListRootsQueryKey() });
    refresh({ queryKey: getListVolumesQueryKey() });
    refresh({ queryKey: getGetTopologyQueryKey() });
  },
  "volumes.changed": (_data, { refresh }) => {
    refresh({ queryKey: getListVolumesQueryKey() });
    refresh({ queryKey: getListHealthVolumesQueryKey() });
    refresh({ queryKey: getGetTopologyQueryKey() });
  },
  "node.settings-changed": (data, { refresh }) => {
    refresh({ queryKey: getGetNodeConfigQueryKey(data.nodeId) });
    refresh({ queryKey: getGetNodeSystemQueryKey(data.nodeId) });
  },
  "tasks.changed": (data, { refresh }) => {
    refresh(everyTasksQuery);
    // A Library scan unit that fails or is canceled ends without
    // roots.changed, and its root's line still shows its last step.
    if (data.taskId === "library-scan") {
      refresh({ queryKey: getListRootsQueryKey() });
    }
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
    takeProgress(queryClient, data);
  },
} satisfies Pick<
  LiveEffects,
  | "nodes.changed"
  | "nodes.traffic"
  | "roots.changed"
  | "volumes.changed"
  | "node.settings-changed"
  | "tasks.changed"
  | "health.changed"
  | "tasks.progress"
>;
