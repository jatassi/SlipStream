import type { DownloadclientsClientCard } from "./downloadclientsClientCard.ts";
import type { DownloadclientsExternalClient } from "./downloadclientsExternalClient.ts";
import type { DownloadclientsKind } from "./downloadclientsKind.ts";
import type { DownloadclientsThresholds } from "./downloadclientsThresholds.ts";

export interface DownloadclientsClientsList {
  /** Each Node assigned Downloading or draining it, in Node order */
  cards: DownloadclientsClientCard[];
  /** The kinds of external client, in the order the form offers them */
  kinds: DownloadclientsKind[];
  /** External clients on Nodes that no longer serve Downloading, which can only be deleted */
  orphans: DownloadclientsExternalClient[];
  thresholds: DownloadclientsThresholds;
}
