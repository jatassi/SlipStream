import type { DownloadclientsFault } from "./downloadclientsFault.ts";
import type { DownloadclientsFileDetail } from "./downloadclientsFileDetail.ts";
import type { DownloadclientsTorrentDetailHold } from "./downloadclientsTorrentDetailHold.ts";
import type { DownloadclientsTorrentDetailPending } from "./downloadclientsTorrentDetailPending.ts";
import type { DownloadclientsTorrentDetailQuality } from "./downloadclientsTorrentDetailQuality.ts";
import type { DownloadclientsTorrentDetailSeverity } from "./downloadclientsTorrentDetailSeverity.ts";
import type { DownloadclientsTorrentDetailState } from "./downloadclientsTorrentDetailState.ts";
import type { DownloadclientsTorrentDetailTabsItem } from "./downloadclientsTorrentDetailTabsItem.ts";
import type { DownloadclientsTorrentGrab } from "./downloadclientsTorrentGrab.ts";
import type { DownloadclientsTrackerCell } from "./downloadclientsTrackerCell.ts";
import type { DownloadclientsTrackerDetail } from "./downloadclientsTrackerDetail.ts";

export interface DownloadclientsTorrentDetail {
  /**
   * All-time seconds it ran
   * @minimum 0
   */
  activeSeconds: number;
  addedAt: string;
  /** The earliest every tracker accepts an announce, set while live */
  announceAt?: string;
  /** The Content Type it was first grabbed for */
  contentType: string;
  /**
   * Bytes done as its Node last reported them; torrents.live carries the live figure
   * @minimum 0
   */
  done: number;
  /**
   * All-time bytes downloaded
   * @minimum 0
   */
  downloaded: number;
  /** Its newest Grab's Edition */
  edition?: string;
  /** The engine's error message for the failure */
  error?: string;
  /** The fault found in the release itself, once there is one */
  fault?: DownloadclientsFault;
  /** In its metainfo's order */
  files: DownloadclientsFileDetail[];
  /** The Grabs that used it, newest first */
  grabs: DownloadclientsTorrentGrab[];
  /** Why its Node holds it back, as its Node last reported */
  hold?: DownloadclientsTorrentDetailHold;
  id: string;
  /** The v1 info hash in hex */
  infoHash: string;
  /** When its Node last reported it, set while the Node is out of step */
  lastKnown?: string;
  /** Its Node responded, so trackers and files are live. Otherwise they are from its last report */
  live: boolean;
  /** The release's name in its metainfo, shown for a Torrent whose Work was deleted */
  name: string;
  nodeId: string;
  /** An Admin's pause or resume its Node hasn't applied yet */
  pending?: DownloadclientsTorrentDetailPending;
  /**
   * Bytes
   * @minimum 1
   */
  pieceSize: number;
  private: boolean;
  /** The quality its newest Grab's release title names */
  quality?: DownloadclientsTorrentDetailQuality;
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
  /** Where its Node keeps it, absent until its Node reports it */
  saveFolder?: string;
  /**
   * All-time seconds seeded
   * @minimum 0
   */
  seedingSeconds: number;
  severity: DownloadclientsTorrentDetailSeverity;
  /**
   * Bytes wanted
   * @minimum 0
   */
  size: number;
  state: DownloadclientsTorrentDetailState;
  /** The Torrents view's tabs it shows under */
  tabs: DownloadclientsTorrentDetailTabsItem[];
  /** Absent when its metainfo names no tracker */
  tracker?: DownloadclientsTrackerCell;
  /**
   * The Tracker its Grab's Release reached, null when no Grab named one
   * @nullable
   */
  trackerId: string | null;
  /** In its metainfo's order */
  trackers: DownloadclientsTrackerDetail[];
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
