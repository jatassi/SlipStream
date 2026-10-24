import type { DownloadclientsCanceled } from "./downloadclientsCanceled.ts";

export interface DownloadclientsPreview {
  /** The Grabs at downloading that a remove fails */
  cancels: DownloadclientsCanceled[];
  /**
   * Bytes deleting the download copies frees, null while a Node can't say
   * @minimum 0
   * @nullable
   */
  frees: number | null;
  /** Some file is also a library file, as a recheck reads it */
  linked: boolean;
  /** The Works whose complete Torrents stop seeding */
  stopSeeding: string[];
}
