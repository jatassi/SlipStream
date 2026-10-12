import { useRouteContext } from "@tanstack/react-router";
import { useSyncExternalStore } from "react";

import type { LiveState } from "./types";

const labels: Record<LiveState, string> = {
  idle: "off",
  connecting: "connecting",
  open: "on",
  reconnecting: "reconnecting",
  paused: "paused",
};

export function useLiveState(): LiveState {
  const live = useRouteContext({ from: "__root__", select: (context) => context.live });
  return useSyncExternalStore(live.subscribe, live.getState);
}

export function LiveStatus() {
  const state = useLiveState();
  return <output className="text-sm text-ink-3">Live updates: {labels[state]}</output>;
}
