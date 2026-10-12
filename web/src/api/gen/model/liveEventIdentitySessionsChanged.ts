import type { IdentitySessionsChanged } from "./identitySessionsChanged.ts";
import type { LiveEventIdentitySessionsChangedEvent } from "./liveEventIdentitySessionsChangedEvent.ts";

export interface LiveEventIdentitySessionsChanged {
  event: LiveEventIdentitySessionsChangedEvent;
  data: IdentitySessionsChanged;
}
