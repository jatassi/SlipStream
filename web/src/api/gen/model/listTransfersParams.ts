export type ListTransfersParams = {
  /**
   * Only Transfers this Node sends or receives
   */
  nodeId?: string;
  /**
   * Only Transfers with a file from this torrent, by its v1 info hash in hex
   * @pattern ^[0-9a-f]{40}$
   */
  infoHash?: string;
};
