export interface RehomeRootInputBody {
  /**
   * The missing file count the Admin confirmed from a rehome_missing response: those files are forgotten, so their cells read missing and wanted. A different count is refused with rehome_missing again
   * @minimum 0
   */
  forget?: number;
  /** The Storage Node whose disk holds it now; left out, its Node */
  nodeId?: string;
  /**
   * The folder that holds it now, in its Node's path style; left out, its folder
   * @maxLength 4096
   */
  path?: string;
}
