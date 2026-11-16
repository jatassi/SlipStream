export interface NodesNodeDial {
  /**
   * Send [] to stop connecting to the Node, which then connects to the Controller
   * @maxItems 0
   */
  addresses?: string[];
  /** Connect to the Node at this Node address from now on */
  nodeAddress?: string;
}
