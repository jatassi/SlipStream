import type { QueryClient } from "@tanstack/react-query";

import type { GrabsSearchSettings } from "../api/gen/model";
import { getGetSearchSettingsQueryKey } from "../api/gen/query/grabs/grabs";
import { getGetTaskQueryKey, getListTasksQueryKey } from "../api/gen/query/tasks/tasks";

const backlogTask = "backlog-search";
const rssTask = "rss-sync";

// Backlog search's and RSS sync's schedules show on the Tasks page too,
// which reads them from the tasks and hears no event when they change here.
export const backlogCadenceKey = `tasks.${backlogTask}.cadence`;
export const rssCadenceKey = `tasks.${rssTask}.cadence`;

const taskOfCadence: Record<string, string> = {
  [backlogCadenceKey]: backlogTask,
  [rssCadenceKey]: rssTask,
};

/** A save answers with the Automatic search settings as written, so the section shows them at once. */
export async function adoptSearchSettings(
  queryClient: QueryClient,
  key: string,
  settings: GrabsSearchSettings,
) {
  queryClient.setQueryData(getGetSearchSettingsQueryKey(), settings);
  const task = taskOfCadence[key];
  if (task !== undefined) {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: getListTasksQueryKey() }),
      queryClient.invalidateQueries({ queryKey: getGetTaskQueryKey(task) }),
    ]);
  }
}
