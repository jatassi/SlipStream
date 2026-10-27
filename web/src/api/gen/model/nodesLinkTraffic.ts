export interface NodesLinkTraffic {
  /** The Controller */
  a: string;
  /** Bytes a second from a to b over the link and their Transfer lanes, TLS records included */
  abBps: number;
  /** The Worker */
  b: string;
  /** Bytes a second from b to a over the link and their Transfer lanes, TLS records included */
  baBps: number;
  /** The link's round trip; 0 until its first ping is answered */
  rttMs: number;
}
