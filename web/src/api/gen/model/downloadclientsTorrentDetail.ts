import type { DownloadclientsFault } from "./downloadclientsFault.ts";
import type { DownloadclientsFileDetail } from "./downloadclientsFileDetail.ts";
import type { DownloadclientsSeedingView } from "./downloadclientsSeedingView.ts";
import type { DownloadclientsTorrentDetailClientKind } from "./downloadclientsTorrentDetailClientKind.ts";
import type { DownloadclientsTorrentDetailHold } from "./downloadclientsTorrentDetailHold.ts";
import type { DownloadclientsTorrentDetailLastKnownCause } from "./downloadclientsTorrentDetailLastKnownCause.ts";
import type { DownloadclientsTorrentDetailPending } from "./downloadclientsTorrentDetailPending.ts";
import type { DownloadclientsTorrentDetailQuality } from "./downloadclientsTorrentDetailQuality.ts";
import type { DownloadclientsTorrentDetailSeverity } from "./downloadclientsTorrentDetailSeverity.ts";
import type { DownloadclientsTorrentDetailState } from "./downloadclientsTorrentDetailState.ts";
import type { DownloadclientsTorrentDetailTabsItem } from "./downloadclientsTorrentDetailTabsItem.ts";
import type { DownloadclientsTorrentGrab } from "./downloadclientsTorrentGrab.ts";
import type { DownloadclientsTrackerCell } from "./downloadclientsTrackerCell.ts";
import type { DownloadclientsTrackerDetail } from "./downloadclientsTrackerDetail.ts";
import type { DownloadclientsTrackerRef } from "./downloadclientsTrackerRef.ts";

export interface DownloadclientsTorrentDetail {
  /**
   * All-time seconds it ran
   * @minimum 0
   */
  activeSeconds: number;
  addedAt: string;
  /** Absent when its metainfo names no announce URL */
  announce?: DownloadclientsTrackerCell;
  /** The earliest every announce URL accepts an announce, set while live */
  announceAt?: string;
  /** It was in its external client before SlipStream attached to it, so removing it only stops tracking it */
  attached?: boolean;
  /** Its external client; absent for its Node's built-in client */
  clientId?: string;
  /** Its external client's kind */
  clientKind?: DownloadclientsTorrentDetailClientKind;
  /** Its external client's name */
  clientName?: string;
  /** The Content Type it was first grabbed for */
  contentType: string;
  /** A remove may delete its download copy; false when removing it only stops tracking it, attached or in an off client */
  deleteFiles: boolean;
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
  /** The engine's error message for the failure, or an external client's own words for its error */
  error?: string;
  /** The fault found in the release itself, once there is one */
  fault?: DownloadclientsFault;
  /** In its metainfo's order */
  files: DownloadclientsFileDetail[];
  /** The Grabs that used it, newest first */
  grabs: DownloadclientsTorrentGrab[];
  /** Why its Node holds it back, as its Node last reported; client_error is an external client's own error, in error */
  hold?: DownloadclientsTorrentDetailHold;
  id: string;
  /** When a finished external torrent imports, set while its client's Import delay runs */
  importAt?: string;
  /** The v1 info hash in hex */
  infoHash: string;
  /** When its Node last reported it, set while the row is stale for lastKnownCause */
  lastKnown?: string;
  /** Why the row is its last report: its Node is out of step, its external client isn't responding or refused the sign-in, or its external client is off */
  lastKnownCause?: DownloadclientsTorrentDetailLastKnownCause;
  /** Its Node responded, so announce URLs and files are live. Otherwise they are from its last report */
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
  /** Its seeding against its Tracker's rule and its seeding goal, null in an external client, whose own limits govern its seeding */
  seeding: DownloadclientsSeedingView | null;
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
  /** Its external client stopped it at its seeding limit */
  stoppedAtLimit?: boolean;
  /** The Torrents view's tabs it shows under */
  tabs: DownloadclientsTorrentDetailTabsItem[];
  /** The Tracker its first Grab's Release reached, null when none did or it is in an external client */
  tracker: DownloadclientsTrackerRef | null;
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
