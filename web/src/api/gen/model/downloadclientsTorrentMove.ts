import type { DownloadclientsTorrentMovePhase } from "./downloadclientsTorrentMovePhase.ts";
import type { DownloadclientsTorrentMoveWaiting } from "./downloadclientsTorrentMoveWaiting.ts";

export interface DownloadclientsTorrentMove {
  batchId: string;
  phase: DownloadclientsTorrentMovePhase;
  toNodeId: string;
  toNodeName: string;
  unitId: string;
  /** What it waits for; absent while it waits for nothing */
  waiting?: DownloadclientsTorrentMoveWaiting;
  /** What it does now, such as Copying to den */
  words: string;
}
