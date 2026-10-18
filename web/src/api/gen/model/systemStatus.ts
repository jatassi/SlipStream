import type { SystemPasskeys } from "./systemPasskeys.ts";
import type { SystemRemoved } from "./systemRemoved.ts";
import type { SystemStatusClaim } from "./systemStatusClaim.ts";

export interface SystemStatus {
  claim: SystemStatusClaim;
  /** Whether the address this request reached SlipStream at can hold a passkey */
  passkeys: SystemPasskeys;
  /** Set on an Unclaimed Node its Controller removed, until it joins again */
  removed?: SystemRemoved;
}
