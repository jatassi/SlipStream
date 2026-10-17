export interface JoinInputBody {
  /**
   * @minLength 1
   * @maxLength 100
   */
  displayName: string;
  /** @pattern ^[0-9]{4}$ */
  pin: string;
}
