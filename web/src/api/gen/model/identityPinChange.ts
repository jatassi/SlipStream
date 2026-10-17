export interface IdentityPinChange {
  /**
   * The current PIN, left out only by a session that must set a new PIN
   * @pattern ^[0-9]{4}$
   */
  current?: string;
  /** @pattern ^[0-9]{4}$ */
  new: string;
}
