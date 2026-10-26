export interface GrabsMetadataSearch {
  /** When the Grab fails unless a peer has sent the metadata */
  endsAt: string;
  /**
   * Peers connected at the Downloading Node's last response
   * @minimum 0
   */
  peers: number;
}
