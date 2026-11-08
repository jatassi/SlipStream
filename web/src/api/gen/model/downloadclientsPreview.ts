import type { DownloadclientsCanceled } from "./downloadclientsCanceled.ts";
import type { DownloadclientsHitAndRun } from "./downloadclientsHitAndRun.ts";

export interface DownloadclientsPreview {
  /** The Grabs at downloading that a remove fails */
  cancels: DownloadclientsCanceled[];
  /**
   * Bytes deleting the download copies frees, null while a Node can't report it
   * @minimum 0
   * @nullable
   */
  frees: number | null;
  /** The removed torrents whose removal counts as a hit and run, each the last copy of a torrent that still owes its Tracker */
  hitAndRun: DownloadclientsHitAndRun[];
  /** Some file is also a library file, as a recheck reads it */
  linked: boolean;
  /** The Works whose complete Torrents stop seeding */
  stopSeeding: string[];
}
