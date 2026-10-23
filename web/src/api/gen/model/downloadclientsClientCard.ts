import type { DownloadclientsClientFacts } from "./downloadclientsClientFacts.ts";
import type { DownloadclientsDiskGuard } from "./downloadclientsDiskGuard.ts";
import type { DownloadclientsLimits } from "./downloadclientsLimits.ts";
import type { DownloadclientsTotals } from "./downloadclientsTotals.ts";

export interface DownloadclientsClientCard {
  /** The built-in client as it reports itself, absent until it does and while the Node is out of step */
  client?: DownloadclientsClientFacts;
  /** Set while the disk guard holds the Node's new downloads: its downloads volume is below its error level */
  diskGuard?: DownloadclientsDiskGuard;
  /** The Node is in step; while it isn't the card has no client */
  inStep: boolean;
  limits: DownloadclientsLimits;
  nodeId: string;
  nodeName: string;
  /**
   * The Node's live Torrents
   * @minimum 0
   */
  torrents: number;
  /** Added up over the Node's live Torrents */
  totals: DownloadclientsTotals;
}
