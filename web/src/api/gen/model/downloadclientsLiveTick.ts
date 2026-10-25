import type { DownloadclientsAggregate } from "./downloadclientsAggregate.ts";
import type { DownloadclientsLiveRow } from "./downloadclientsLiveRow.ts";

export interface DownloadclientsLiveTick {
  at: string;
  downloads: DownloadclientsAggregate;
  /** The Torrents whose rates, progress or peers moved */
  rows: DownloadclientsLiveRow[];
}
