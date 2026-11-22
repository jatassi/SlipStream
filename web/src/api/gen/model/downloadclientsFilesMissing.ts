import type { DownloadclientsMissingTorrent } from "./downloadclientsMissingTorrent.ts";

export interface DownloadclientsFilesMissing {
  /**
   * The torrents whose files the new Node didn't find
   * @minimum 1
   */
  count: number;
  /** The first 20 of them, in the order SlipStream added them */
  first: DownloadclientsMissingTorrent[];
}
