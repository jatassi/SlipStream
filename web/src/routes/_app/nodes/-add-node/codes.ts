import { useQueryClient } from "@tanstack/react-query";

import { refresh } from "../../../../live/refresh";
import { everyJoinCodeQuery } from "../../../../nodes/join-codes";

/** Refetches the open codes and each code's state after this page changed one. */
export function useRefreshCodes(): () => Promise<void> {
  const queryClient = useQueryClient();
  return () => refresh(queryClient, everyJoinCodeQuery);
}
