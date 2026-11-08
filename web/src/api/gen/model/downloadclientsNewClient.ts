import type { DownloadclientsClientFolder } from "./downloadclientsClientFolder.ts";
import type { DownloadclientsDaemon } from "./downloadclientsDaemon.ts";
import type { DownloadclientsNewClientKind } from "./downloadclientsNewClientKind.ts";

export interface DownloadclientsNewClient {
  /**
   * The client's whole address, such as http://localhost:9091/transmission/rpc; with no path it takes the kind's usual one
   * @minLength 1
   * @maxLength 2048
   */
  address: string;
  /**
   * qBittorrent 5.2 and later. Write-only
   * @maxLength 1024
   */
  apiKey?: string;
  /** Seconds; absent for the default */
  checkEvery?: number;
  /**
   * @maxItems 50
   * @nullable
   */
  clientFolders?: DownloadclientsClientFolder[] | null;
  /** Deluge's daemon; absent for the only one */
  daemon?: DownloadclientsDaemon;
  /**
   * The download folder as the client sees it; absent for the client's default
   * @maxLength 4096
   */
  folder?: string;
  /** Seconds; absent for the default */
  importDelay?: number;
  kind: DownloadclientsNewClientKind;
  /**
   * Absent for slipstream
   * @maxLength 100
   */
  label?: string;
  /**
   * Absent for the kind's name, such as Transmission
   * @maxLength 100
   */
  name?: string;
  /** A Node that holds Downloading */
  nodeId: string;
  /** Absent for on */
  on?: boolean;
  /**
   * Write-only
   * @maxLength 1024
   */
  password?: string;
  /** @maxLength 256 */
  username?: string;
}
