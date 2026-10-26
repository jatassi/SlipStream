import type { IdentityAccount } from "./identityAccount.ts";
import type { IdentitySessionFirstRun } from "./identitySessionFirstRun.ts";
import type { IdentitySessionKeyPrompt } from "./identitySessionKeyPrompt.ts";

export interface IdentitySession {
  account: IdentityAccount;
  /** An Admin's first setup; absent for a Member */
  firstRun?: IdentitySessionFirstRun;
  /** Whether the prompt to add a passkey may show after sign-in */
  keyPrompt: IdentitySessionKeyPrompt;
  /** The session must set a new PIN before anything but reading the session and signing out */
  pinRequired: boolean;
}
