import type { IdentityAccountsChanged } from "./identityAccountsChanged.ts";
import type { LiveEventIdentityAccountsChangedEvent } from "./liveEventIdentityAccountsChangedEvent.ts";

export interface LiveEventIdentityAccountsChanged {
  event: LiveEventIdentityAccountsChangedEvent;
  data: IdentityAccountsChanged;
}
