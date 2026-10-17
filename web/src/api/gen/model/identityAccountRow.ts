import type { IdentityPinState } from "./identityPinState.ts";

export interface IdentityAccountRow {
  admin: boolean;
  /** Turned off: it can't sign in and holds no session */
  disabled: boolean;
  /**
   * @minLength 1
   * @maxLength 100
   */
  displayName: string;
  /** @maxLength 254 */
  email: string;
  id: string;
  /** Absent until it first signs in */
  lastSignInAt?: string;
  passkeyCount: number;
  pin: IdentityPinState;
}
