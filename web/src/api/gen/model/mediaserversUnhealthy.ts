import type { MediaserversUnhealthyReason } from "./mediaserversUnhealthyReason.ts";

export interface MediaserversUnhealthy {
  reason: MediaserversUnhealthyReason;
  since: string;
}
