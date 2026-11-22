import type { QueryClient } from "@tanstack/react-query";

import type { HomeSettings } from "../api/gen/model";
import { getGetHomeQueryKey, getGetHomeSettingsQueryKey } from "../api/gen/query/home/home";

/** A save answers with Rows on Home as written, so the field shows it at once and Home reads its lists again. */
export async function adoptHomeSettings(queryClient: QueryClient, settings: HomeSettings) {
  queryClient.setQueryData(getGetHomeSettingsQueryKey(), settings);
  await queryClient.invalidateQueries({ queryKey: getGetHomeQueryKey() });
}
