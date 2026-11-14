import type { MediaserversKind } from "./mediaserversKind.ts";
import type { MediaserversWentUnhealthyReason } from "./mediaserversWentUnhealthyReason.ts";

export interface MediaserversWentUnhealthy {
  kind: MediaserversKind;
  reason: MediaserversWentUnhealthyReason;
  serverId: string;
}
