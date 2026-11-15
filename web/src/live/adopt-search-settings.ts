import type { QueryClient } from "@tanstack/react-query";

import type { GrabsSearchSettings } from "../api/gen/model";
import { getGetSearchSettingsQueryKey } from "../api/gen/query/grabs/grabs";

/** A save answers with the Automatic search settings as written, so the section shows them at once. */
export function adoptSearchSettings(queryClient: QueryClient, settings: GrabsSearchSettings) {
  queryClient.setQueryData(getGetSearchSettingsQueryKey(), settings);
}
