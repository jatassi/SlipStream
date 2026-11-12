export interface ImportFolderInputBody {
  /**
   * Absolute on the Node, inside one of its library roots or download folders
   * @minLength 1
   */
  folder: string;
  nodeId: string;
}
