import type { DownloadclientsDaemon } from "./downloadclientsDaemon.ts";

export interface DownloadclientsTestReport {
  /** The daemons Deluge's web UI lists; empty for every other kind */
  daemons: DownloadclientsDaemon[];
  /** The download folder as the Node sees it */
  nodeFolder: string;
  /** The product's name as it responded, such as Transmission */
  product: string;
  /** Such as 4.1.1 */
  version: string;
  /** What could go wrong with these settings, as sentences */
  warnings: string[];
}
