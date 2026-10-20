import type { DownloadclientsTabCountTab } from "./downloadclientsTabCountTab.ts";

export interface DownloadclientsTabCount {
  tab: DownloadclientsTabCountTab;
  /** @minimum 0 */
  torrents: number;
}
