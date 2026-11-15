import type { QueryClient } from "@tanstack/react-query";

import type { GrabsSearchSettings } from "../api/gen/model";
import { getGetSearchSettingsQueryKey } from "../api/gen/query/grabs/grabs";
import { getGetTaskQueryKey, getListTasksQueryKey } from "../api/gen/query/tasks/tasks";

const backlogTask = "backlog-search";

// Backlog search's schedule shows on the Tasks page too, which reads it
// from the task and hears no event when it changes here.
export const backlogCadenceKey = `tasks.${backlogTask}.cadence`;

/** A save answers with the Automatic search settings as written, so the section shows them at once. */
export async function adoptSearchSettings(
  queryClient: QueryClient,
  key: string,
  settings: GrabsSearchSettings,
) {
  queryClient.setQueryData(getGetSearchSettingsQueryKey(), settings);
  if (key === backlogCadenceKey) {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: getListTasksQueryKey() }),
      queryClient.invalidateQueries({ queryKey: getGetTaskQueryKey(backlogTask) }),
    ]);
  }
}
