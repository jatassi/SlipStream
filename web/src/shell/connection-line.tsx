import { useLiveState } from "../live/live-status";
import type { LiveState } from "../live/types";

// Idle and paused are the app's own doing and connecting is the first moment
// of a load, so only a lost stream is worth a line.
const lines: { readonly [S in LiveState]: string } = {
  idle: "",
  connecting: "",
  open: "",
  reconnecting: "Reconnecting to SlipStream…",
  paused: "",
};

export function ConnectionLine() {
  return (
    <output className="text-meta whitespace-nowrap text-ink-3 empty:hidden">
      {lines[useLiveState()]}
    </output>
  );
}
