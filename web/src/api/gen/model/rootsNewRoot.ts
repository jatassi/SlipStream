export interface RootsNewRoot {
  /** @pattern ^[a-z][a-z0-9]{1,15}$ */
  contentType: string;
  /**
   * An absolute path to an existing folder on the Controller
   * @minLength 1
   * @maxLength 4096
   */
  path: string;
}
