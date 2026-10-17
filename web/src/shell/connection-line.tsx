import { useLiveState } from "../live/live-status";
import type { LiveState } from "../live/types";
import { useRestartState } from "./restart/restart-state";

// Idle and paused are the app's own doing and connecting is the first moment
// of a load, so only a lost stream is worth a line, and not while the Admin
// restarts the Node on purpose.
const lines: { readonly [S in LiveState]: string } = {
  idle: "",
  connecting: "",
  open: "",
  reconnecting: "Reconnecting to SlipStream…",
  paused: "",
};

export function ConnectionLine() {
  const line = lines[useLiveState()];
  const restarting = useRestartState().self !== null;
  return (
    <output className="text-meta whitespace-nowrap text-ink-3 empty:hidden">
      {restarting ? "" : line}
    </output>
  );
}
