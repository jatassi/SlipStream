export interface IdentitySettingsPatch {
  /**
   * null resets it to its default
   * @nullable
   */
  addressFailures?: number | null;
  /**
   * Seconds; null resets it to its default
   * @nullable
   */
  addressWindow?: number | null;
  /**
   * Seconds; null resets it to its default
   * @nullable
   */
  emailLinkLifetime?: number | null;
  /**
   * null resets it to its default
   * @nullable
   */
  pinOffAfter?: number | null;
  /**
   * Seconds; null resets it to its default
   * @nullable
   */
  pinWait?: number | null;
  /**
   * null resets it to its default
   * @nullable
   */
  pinWaitAfter?: number | null;
  /**
   * Seconds; null resets it to its default
   * @nullable
   */
  reconfirmWindow?: number | null;
  /**
   * Seconds; null resets it to its default
   * @nullable
   */
  resetLinkLifetime?: number | null;
  /**
   * Seconds; null resets it to its default
   * @nullable
   */
  sessionLifetime?: number | null;
  /**
   * Seconds; null resets it to its default
   * @nullable
   */
  signupLinkLifetime?: number | null;
}
