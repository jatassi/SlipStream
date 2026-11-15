import type { SystemPasskeys } from "./systemPasskeys.ts";
import type { SystemRemoved } from "./systemRemoved.ts";
import type { SystemStatusClaim } from "./systemStatusClaim.ts";

export interface SystemStatus {
  claim: SystemStatusClaim;
  /** Whether passkeys work at the address this request used */
  passkeys: SystemPasskeys;
  /** Set on an Unclaimed Node its Controller removed, until it joins again */
  removed?: SystemRemoved;
}
