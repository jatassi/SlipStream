export interface NodesNodeDial {
  /**
   * Send [] to stop dialing the Node, which then dials the Controller
   * @maxItems 0
   */
  addresses?: string[];
  /** Dial the Node at this Node address from now on */
  nodeAddress?: string;
}
