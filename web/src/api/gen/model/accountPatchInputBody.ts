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
   * Changed at once. SlipStream emails the old address about the change
   * @maxLength 254
   */
  email?: string;
}
