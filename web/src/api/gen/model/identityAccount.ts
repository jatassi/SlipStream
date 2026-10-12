export interface IdentityAccount {
  admin: boolean;
  /**
   * @minLength 1
   * @maxLength 100
   */
  displayName: string;
  /** @maxLength 254 */
  email: string;
  id: string;
}
