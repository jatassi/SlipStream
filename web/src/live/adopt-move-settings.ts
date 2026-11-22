import type { QueryClient } from "@tanstack/react-query";

import type { MovesMoveSettings } from "../api/gen/model";
import { getGetMoveSettingsQueryKey } from "../api/gen/query/moves/moves";

/** A save answers with the Moves settings as written, so each field shows it at once. */
export function adoptMoveSettings(queryClient: QueryClient, settings: MovesMoveSettings) {
  queryClient.setQueryData(getGetMoveSettingsQueryKey(), settings);
}
