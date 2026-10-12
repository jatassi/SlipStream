export interface IdentityClaimBody {
  /**
   * @minLength 1
   * @maxLength 100
   */
  displayName: string;
  /** @maxLength 254 */
  email: string;
  /** @pattern ^[0-9]{4}$ */
  pin: string;
}
