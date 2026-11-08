import type { DownloadclientsFault } from "./downloadclientsFault.ts";
import type { DownloadclientsSeedingView } from "./downloadclientsSeedingView.ts";
import type { DownloadclientsTorrentRowHold } from "./downloadclientsTorrentRowHold.ts";
import type { DownloadclientsTorrentRowPending } from "./downloadclientsTorrentRowPending.ts";
import type { DownloadclientsTorrentRowQuality } from "./downloadclientsTorrentRowQuality.ts";
import type { DownloadclientsTorrentRowSeverity } from "./downloadclientsTorrentRowSeverity.ts";
import type { DownloadclientsTorrentRowState } from "./downloadclientsTorrentRowState.ts";
import type { DownloadclientsTorrentRowTabsItem } from "./downloadclientsTorrentRowTabsItem.ts";
import type { DownloadclientsTrackerCell } from "./downloadclientsTrackerCell.ts";
import type { DownloadclientsTrackerRef } from "./downloadclientsTrackerRef.ts";

export interface DownloadclientsTorrentRow {
  addedAt: string;
  /** Absent when its metainfo names no announce URL */
  announce?: DownloadclientsTrackerCell;
  /** The Content Type it was first grabbed for */
  contentType: string;
  /**
   * Bytes done as its Node last reported them; torrents.live carries the live figure
   * @minimum 0
   */
  done: number;
  /** Its newest Grab's Edition */
  edition?: string;
  /** The engine's error message for the failure */
  error?: string;
  /** The fault found in the release itself, once there is one */
  fault?: DownloadclientsFault;
  /** Why its Node holds it back, as its Node last reported */
  hold?: DownloadclientsTorrentRowHold;
  id: string;
  /** When its Node last reported it, set while the Node is out of step */
  lastKnown?: string;
  /** The release's name in its metainfo, shown for a Torrent whose Work was deleted */
  name: string;
  nodeId: string;
  /** An Admin's pause or resume its Node hasn't applied yet */
  pending?: DownloadclientsTorrentRowPending;
  /** The quality its newest Grab's release title names */
  quality?: DownloadclientsTorrentRowQuality;
  /**
   * Its place, from 1, among its Node's downloads waiting for a slot
   * @minimum 0
   */
  queuePosition?: number;
  /**
   * Uploaded over the larger of downloaded and size
   * @minimum 0
   */
  ratio: number;
  /** Its seeding against its Tracker's rule and its seeding goal */
  seeding: DownloadclientsSeedingView;
  /**
   * All-time seconds seeded
   * @minimum 0
   */
  seedingSeconds: number;
  severity: DownloadclientsTorrentRowSeverity;
  /**
   * Bytes wanted
   * @minimum 0
   */
  size: number;
  state: DownloadclientsTorrentRowState;
  /** The Torrents view's tabs it shows under */
  tabs: DownloadclientsTorrentRowTabsItem[];
  /** The Tracker its first Grab's Release reached, null when none did */
  tracker: DownloadclientsTrackerRef | null;
  /** When a window hold starts it, or a check resumes it */
  until?: string;
  /**
   * All-time bytes uploaded
   * @minimum 0
   */
  uploaded: number;
  /** Its newest Grab's Work, absent once no Grab is left */
  workId?: string;
}
