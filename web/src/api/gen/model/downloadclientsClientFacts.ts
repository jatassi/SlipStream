import type { DownloadclientsClientFactsReach } from "./downloadclientsClientFactsReach.ts";

export interface DownloadclientsClientFacts {
  /** The address and port it listens on, absent while it has no listener */
  bound?: string;
  /** libtorrent's version */
  libtorrent: string;
  /** The listen address and port it is set to */
  listen: string;
  /** Why it has no listener */
  listenError?: string;
  /** The peer ID prefix it announces, which a tracker's allow list names */
  peerIdPrefix: string;
  /** reachable once a peer from the internet connected in to where it listens now */
  reach: DownloadclientsClientFactsReach;
  /** When the first such peer connected in */
  reachedAt?: string;
  /** The user agent it announces */
  userAgent: string;
}
