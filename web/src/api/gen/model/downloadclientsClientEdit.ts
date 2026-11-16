import type { DownloadclientsClientEditKind } from "./downloadclientsClientEditKind.ts";
import type { DownloadclientsClientFolder } from "./downloadclientsClientFolder.ts";
import type { DownloadclientsDaemon } from "./downloadclientsDaemon.ts";

export interface DownloadclientsClientEdit {
  /**
   * @minLength 1
   * @maxLength 2048
   */
  address?: string;
  /**
   * Write-only; empty keeps the saved one
   * @maxLength 1024
   */
  apiKey?: string;
  /**
   * Seconds; null resets it to its default
   * @nullable
   */
  checkEvery?: number | null;
  /** @maxItems 50 */
  clientFolders?: DownloadclientsClientFolder[];
  /** Null when Deluge has only one daemon */
  daemon?: DownloadclientsDaemon | null;
  /**
   * Empty for the client's default
   * @maxLength 4096
   */
  folder?: string;
  /**
   * Seconds; null resets it to its default
   * @nullable
   */
  importDelay?: number | null;
  kind?: DownloadclientsClientEditKind;
  /**
   * @minLength 1
   * @maxLength 100
   */
  label?: string;
  /**
   * Empty for the kind's name
   * @maxLength 100
   */
  name?: string;
  /** Another Node with the Downloading Role; a client with Torrents can't move */
  nodeId?: string;
  on?: boolean;
  /**
   * Write-only; empty keeps the saved one
   * @maxLength 1024
   */
  password?: string;
  /**
   * Empty when the client has no sign-in
   * @maxLength 256
   */
  username?: string;
}
