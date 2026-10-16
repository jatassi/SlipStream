import type { IdentityAccount } from "./identityAccount.ts";

export interface IdentitySession {
  account: IdentityAccount;
  /** The session must set a new PIN before anything but reading the session and signing out */
  pinRequired: boolean;
}
