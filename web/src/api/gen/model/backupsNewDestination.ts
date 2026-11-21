export interface BackupsNewDestination {
  /**
   * An absolute folder in the Node's path style
   * @minLength 1
   * @maxLength 4096
   */
  folder: string;
  nodeId: string;
}
