import type { QueryClient } from "@tanstack/react-query";

import type {
  HealthReport,
  SystemNodeConfig,
  TasksOverview,
  TasksSchedule,
} from "../api/gen/model";
import { getGetSystemHealthQueryKey } from "../api/gen/query/health/health";
import { getGetNodeConfigQueryKey } from "../api/gen/query/system/system";
import { getListTasksQueryKey } from "../api/gen/query/tasks/tasks";

// A save answers with the settings as written, so the page shows them at once
// rather than after the live event's refetch.

export function adoptNodeConfig(
  queryClient: QueryClient,
  nodeId: string,
  config: SystemNodeConfig,
) {
  queryClient.setQueryData(getGetNodeConfigQueryKey(nodeId), config);
}

export function adoptTaskSettings(queryClient: QueryClient, settings: TasksSchedule) {
  queryClient.setQueryData<TasksOverview>(
    getListTasksQueryKey(),
    (overview) => overview && { ...overview, settings },
  );
}

export function adoptHealthReport(queryClient: QueryClient, report: HealthReport) {
  queryClient.setQueryData(getGetSystemHealthQueryKey(), report);
}
