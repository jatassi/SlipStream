export interface IdentitySettingsPatch {
  /** @maxLength 2048 */
  address?: string;
  addressFailures?: number;
  addressWindowSeconds?: number;
  emailLinkLifetimeSeconds?: number;
  pinOffAfter?: number;
  pinWaitAfter?: number;
  pinWaitSeconds?: number;
  reconfirmWindowSeconds?: number;
  resetLinkLifetimeSeconds?: number;
  sessionLifetimeSeconds?: number;
  signupLinkLifetimeSeconds?: number;
}
