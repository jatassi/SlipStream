import type { SystemPasskeys } from "./systemPasskeys.ts";
import type { SystemStatusClaim } from "./systemStatusClaim.ts";

export interface SystemStatus {
  claim: SystemStatusClaim;
  /** Whether the address this request reached SlipStream at can hold a passkey */
  passkeys: SystemPasskeys;
}
