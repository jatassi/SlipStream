export interface IdentitySettings {
  /** The origin links are built on, empty until set */
  address: string;
  /** Failed sign-in attempts one address may make in the window */
  addressFailures: number;
  addressWindowSeconds: number;
  emailLinkLifetimeSeconds: number;
  /** Wrong PINs in a row before the PIN turns off */
  pinOffAfter: number;
  /** Wrong PINs before a wait */
  pinWaitAfter: number;
  /** The wait after too many wrong PINs */
  pinWaitSeconds: number;
  /** How long a confirm with the PIN or a passkey lasts */
  reconfirmWindowSeconds: number;
  resetLinkLifetimeSeconds: number;
  /** How long a session lasts unused */
  sessionLifetimeSeconds: number;
  signupLinkLifetimeSeconds: number;
  /** The address this request reached SlipStream at, or this machine's name when that was an IP address */
  suggestedAddress: string;
}
