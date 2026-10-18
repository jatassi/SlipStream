export interface RootsNewRoot {
  /** @pattern ^[a-z][a-z0-9]{1,15}$ */
  contentType: string;
  /** Make it its Content Type's default; a Content Type's first root is its default whatever this says */
  makeDefault?: boolean;
  /**
   * An absolute path to an existing folder on the Controller
   * @minLength 1
   * @maxLength 4096
   */
  path: string;
}
