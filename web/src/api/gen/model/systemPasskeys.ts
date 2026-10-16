import type { SystemPasskeysReason } from "./systemPasskeysReason.ts";

export interface SystemPasskeys {
  available: boolean;
  /** Why it can't: an IP address, a name over plain HTTP other than localhost, or a host that is not a domain name */
  reason?: SystemPasskeysReason;
}
