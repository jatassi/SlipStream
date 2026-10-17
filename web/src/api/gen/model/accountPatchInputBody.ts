export interface AccountPatchInputBody {
  admin?: boolean;
  /** Turning it off ends every session it holds */
  disabled?: boolean;
  /**
   * @minLength 1
   * @maxLength 100
   */
  displayName?: string;
  /**
   * Changed at once; the old address is told
   * @maxLength 254
   */
  email?: string;
}
