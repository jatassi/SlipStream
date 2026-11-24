import type { LiveEventMovesChangedEvent } from "./liveEventMovesChangedEvent.ts";
import type { MovesChanged } from "./movesChanged.ts";

export interface LiveEventMovesChanged {
  event: LiveEventMovesChangedEvent;
  data: MovesChanged;
}
