export interface IdentitySignInBody {
  /** @maxLength 254 */
  email: string;
  /** @pattern ^[0-9]{4}$ */
  pin: string;
}
