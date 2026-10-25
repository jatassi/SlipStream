import type { DownloadclientsClientCard } from "./downloadclientsClientCard.ts";
import type { DownloadclientsThresholds } from "./downloadclientsThresholds.ts";

export interface DownloadclientsClientsList {
  /** Each Node assigned Downloading or draining it, in Node order */
  cards: DownloadclientsClientCard[];
  thresholds: DownloadclientsThresholds;
}
