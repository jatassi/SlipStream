import type { IdentityAccount } from "./identityAccount.ts";
import type { IdentitySignedInFirstRun } from "./identitySignedInFirstRun.ts";
import type { IdentitySignedInKeyPrompt } from "./identitySignedInKeyPrompt.ts";

export interface IdentitySignedIn {
  account: IdentityAccount;
  /** An Admin's first setup; absent for a Member */
  firstRun?: IdentitySignedInFirstRun;
  /** Whether the prompt to add a passkey may show after sign-in */
  keyPrompt: IdentitySignedInKeyPrompt;
  /** Show the prompt to add a passkey for this address, once in this session. False while first setup is open, because its passkey step asks instead. */
  passkeyPrompt: boolean;
  /** The session must set a new PIN before anything but reading the session and signing out */
  pinRequired: boolean;
}
