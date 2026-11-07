import type { DownloadclientsClientFolder } from "./downloadclientsClientFolder.ts";
import type { DownloadclientsDaemon } from "./downloadclientsDaemon.ts";
import type { DownloadclientsExternalClientActionsItem } from "./downloadclientsExternalClientActionsItem.ts";
import type { DownloadclientsExternalClientKind } from "./downloadclientsExternalClientKind.ts";
import type { DownloadclientsExternalClientReach } from "./downloadclientsExternalClientReach.ts";
import type { DownloadclientsPollProblem } from "./downloadclientsPollProblem.ts";
import type { SettingDurationField } from "./settingDurationField.ts";

export interface DownloadclientsExternalClient {
  /** The actions its kind supports */
  actions: DownloadclientsExternalClientActionsItem[];
  /** The whole address, with the kind's usual path filled in */
  address: string;
  /** An API key is saved. Secrets are never returned */
  apiKeySaved: boolean;
  /** It isn't responding, refused the sign-in or needs attention while its Node is in step, so its card is red */
  away: boolean;
  checkEvery: SettingDurationField;
  clientFolders: DownloadclientsClientFolder[];
  /** Deluge's daemon; null for the only one and for every other kind */
  daemon: DownloadclientsDaemon | null;
  /** The download folder as the client sees it; empty for the client's default */
  folder: string;
  id: string;
  importDelay: SettingDurationField;
  kind: DownloadclientsExternalClientKind;
  label: string;
  name: string;
  nodeId: string;
  on: boolean;
  /** A password is saved. Secrets are never returned */
  passwordSaved: boolean;
  /** Why its last poll failed; null while it responds and before its first poll */
  problem: DownloadclientsPollProblem | null;
  /** The product's version at the last sign-in, such as 4.1.1; empty before one */
  productVersion: string;
  reach: DownloadclientsExternalClientReach;
  /**
   * Bumped on every edit
   * @minimum 1
   */
  settingsVersion: number;
  /**
   * When its poll state began; null before its Node first polled it
   * @nullable
   */
  since: string | null;
  /**
   * Its live Torrents
   * @minimum 0
   */
  torrents: number;
  /** Empty when it signs in with none */
  username: string;
  /**
   * The client's own web UI; null for a kind with none
   * @nullable
   */
  webUi: string | null;
}
