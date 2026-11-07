import type { DownloadclientsFault } from "./downloadclientsFault.ts";
import type { DownloadclientsTorrentRowClientKind } from "./downloadclientsTorrentRowClientKind.ts";
import type { DownloadclientsTorrentRowHold } from "./downloadclientsTorrentRowHold.ts";
import type { DownloadclientsTorrentRowLastKnownCause } from "./downloadclientsTorrentRowLastKnownCause.ts";
import type { DownloadclientsTorrentRowPending } from "./downloadclientsTorrentRowPending.ts";
import type { DownloadclientsTorrentRowQuality } from "./downloadclientsTorrentRowQuality.ts";
import type { DownloadclientsTorrentRowSeverity } from "./downloadclientsTorrentRowSeverity.ts";
import type { DownloadclientsTorrentRowState } from "./downloadclientsTorrentRowState.ts";
import type { DownloadclientsTorrentRowTabsItem } from "./downloadclientsTorrentRowTabsItem.ts";
import type { DownloadclientsTrackerCell } from "./downloadclientsTrackerCell.ts";

export interface DownloadclientsTorrentRow {
  addedAt: string;
  /** It was in its external client before SlipStream attached to it, so removing it only stops tracking it */
  attached?: boolean;
  /** Its external client; absent for its Node's built-in client */
  clientId?: string;
  /** Its external client's kind */
  clientKind?: DownloadclientsTorrentRowClientKind;
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
  /** Its newest Grab's Edition */
  edition?: string;
  /** The engine's error message for the failure, or an external client's own words for its error */
  error?: string;
  /** The fault found in the release itself, once there is one */
  fault?: DownloadclientsFault;
  /** Why its Node holds it back, as its Node last reported; client_error is an external client's own error, in error */
  hold?: DownloadclientsTorrentRowHold;
  id: string;
  /** When a finished external torrent imports, set while its client's Import delay runs */
  importAt?: string;
  /** When its Node last reported it, set while the row is stale for lastKnownCause */
  lastKnown?: string;
  /** Why the row is its last report: its Node is out of step, its external client isn't responding or refused the sign-in, or its external client is off */
  lastKnownCause?: DownloadclientsTorrentRowLastKnownCause;
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
  /** Its external client stopped it at its seeding limit */
  stoppedAtLimit?: boolean;
  /** The Torrents view's tabs it shows under */
  tabs: DownloadclientsTorrentRowTabsItem[];
  /** Absent when its metainfo names no announce URL */
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
