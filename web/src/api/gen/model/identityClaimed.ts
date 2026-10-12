import type { IdentityAccount } from "./identityAccount.ts";

export interface IdentityClaimed {
  account: IdentityAccount;
  /** @pattern ^ssk_[A-Za-z0-9_-]{43}$ */
  apiKey: string;
}
