import type { DownloadclientsStateCountState } from "./downloadclientsStateCountState.ts";

export interface DownloadclientsStateCount {
  state: DownloadclientsStateCountState;
  /** @minimum 0 */
  torrents: number;
}
