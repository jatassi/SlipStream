import type { IdentityAddressChanged } from "./identityAddressChanged.ts";
import type { LiveEventIdentityAddressChangedEvent } from "./liveEventIdentityAddressChangedEvent.ts";

export interface LiveEventIdentityAddressChanged {
  event: LiveEventIdentityAddressChangedEvent;
  data: IdentityAddressChanged;
}
