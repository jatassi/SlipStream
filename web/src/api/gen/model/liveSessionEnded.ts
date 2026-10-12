import type { LiveSessionEndedReason } from "./liveSessionEndedReason.ts";

export interface LiveSessionEnded {
  reason: LiveSessionEndedReason;
}
