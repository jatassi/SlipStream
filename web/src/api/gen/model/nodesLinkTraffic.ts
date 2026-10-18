export interface NodesLinkTraffic {
  /** The Controller */
  a: string;
  /** Bytes a second from a to b, TLS records included */
  abBps: number;
  /** The Worker */
  b: string;
  /** Bytes a second from b to a, TLS records included */
  baBps: number;
  /** The link's round trip; 0 until its first ping is answered */
  rttMs: number;
}
