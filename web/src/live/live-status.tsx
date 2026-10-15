import { useRouteContext } from "@tanstack/react-router";
import { useSyncExternalStore } from "react";

import type { LiveState } from "./types";

export function useLiveState(): LiveState {
  const live = useRouteContext({ from: "__root__", select: (context) => context.live });
  return useSyncExternalStore(live.subscribe, live.getState);
}
