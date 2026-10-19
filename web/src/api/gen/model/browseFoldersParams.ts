export type BrowseFoldersParams = {
  /**
   * An absolute folder in the Node's path style; empty for its places
   * @maxLength 4096
   */
  path?: string;
  /**
   * List hidden and system folders too
   */
  hidden?: boolean;
};
