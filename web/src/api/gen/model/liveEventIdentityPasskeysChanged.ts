import type { IdentityPasskeysChanged } from "./identityPasskeysChanged.ts";
import type { LiveEventIdentityPasskeysChangedEvent } from "./liveEventIdentityPasskeysChangedEvent.ts";

export interface LiveEventIdentityPasskeysChanged {
  event: LiveEventIdentityPasskeysChangedEvent;
  data: IdentityPasskeysChanged;
}
