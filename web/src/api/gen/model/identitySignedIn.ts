import type { IdentityAccount } from "./identityAccount.ts";

export interface IdentitySignedIn {
  account: IdentityAccount;
  /** Show the prompt to add a passkey for this address, once in this session */
  passkeyPrompt: boolean;
  /** The session must set a new PIN before anything but reading the session and signing out */
  pinRequired: boolean;
}
