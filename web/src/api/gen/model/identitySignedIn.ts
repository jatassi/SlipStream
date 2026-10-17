import type { IdentityAccount } from "./identityAccount.ts";
import type { IdentitySignedInKeyPrompt } from "./identitySignedInKeyPrompt.ts";

export interface IdentitySignedIn {
  account: IdentityAccount;
  /** Whether the prompt to add a passkey may show after sign-in */
  keyPrompt: IdentitySignedInKeyPrompt;
  /** Show the prompt to add a passkey for this address, once in this session */
  passkeyPrompt: boolean;
  /** The session must set a new PIN before anything but reading the session and signing out */
  pinRequired: boolean;
}
