export interface CheckRootInputBody {
  nodeId: string;
  /**
   * @minLength 1
   * @maxLength 4096
   */
  path: string;
  /** The root being re-homed, which the folder may overlap */
  rootId?: string;
}
