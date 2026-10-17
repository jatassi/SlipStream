import type { IdentityLinksChanged } from "./identityLinksChanged.ts";
import type { LiveEventIdentityLinksChangedEvent } from "./liveEventIdentityLinksChangedEvent.ts";

export interface LiveEventIdentityLinksChanged {
  event: LiveEventIdentityLinksChangedEvent;
  data: IdentityLinksChanged;
}
