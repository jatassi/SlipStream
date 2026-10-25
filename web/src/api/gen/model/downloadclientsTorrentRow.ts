import type { DownloadclientsFault } from "./downloadclientsFault.ts";
import type { DownloadclientsTorrentRowHold } from "./downloadclientsTorrentRowHold.ts";
import type { DownloadclientsTorrentRowPending } from "./downloadclientsTorrentRowPending.ts";
import type { DownloadclientsTorrentRowQuality } from "./downloadclientsTorrentRowQuality.ts";
import type { DownloadclientsTorrentRowSeverity } from "./downloadclientsTorrentRowSeverity.ts";
import type { DownloadclientsTorrentRowState } from "./downloadclientsTorrentRowState.ts";
import type { DownloadclientsTorrentRowTabsItem } from "./downloadclientsTorrentRowTabsItem.ts";
import type { DownloadclientsTrackerCell } from "./downloadclientsTrackerCell.ts";

export interface DownloadclientsTorrentRow {
  addedAt: string;
  /** The Content Type it was first grabbed for */
  contentType: string;
  /**
   * Bytes done as its Node last reported them; torrents.live carries the live figure
   * @minimum 0
   */
  done: number;
  /** Its newest Grab's Edition */
  edition?: string;
  /** The engine's words for why it failed it */
  error?: string;
  /** Why the release itself looks bad, once it does */
  fault?: DownloadclientsFault;
  /** Why its Node holds it back, as its Node last reported */
  hold?: DownloadclientsTorrentRowHold;
  id: string;
  /** When its Node last reported it, set while the Node is out of step */
  lastKnown?: string;
  /** The release's name in its metainfo, which a Torrent whose Work was deleted reads by */
  name: string;
  nodeId: string;
  /** An Admin's pause or resume its Node has yet to act on */
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
  /** Absent when its metainfo names no tracker */
  tracker?: DownloadclientsTrackerCell;
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
