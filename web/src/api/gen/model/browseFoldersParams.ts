import type { BrowseFoldersPurpose } from "./browseFoldersPurpose.ts";

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
  /**
   * import: browse only the Node's library roots and download folders, which are then its places
   */
  purpose?: BrowseFoldersPurpose;
};
