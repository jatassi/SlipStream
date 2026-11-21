export interface ReconcilingSaveRoot {
  /**
   * The missing file count the Admin confirmed, for Save anyway: those files are forgotten
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
  rootId: string;
}
