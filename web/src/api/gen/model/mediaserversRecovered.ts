import type { MediaserversKind } from "./mediaserversKind.ts";
import type { MediaserversRecoveredReason } from "./mediaserversRecoveredReason.ts";

export interface MediaserversRecovered {
  kind: MediaserversKind;
  reason: MediaserversRecoveredReason;
  serverId: string;
}
