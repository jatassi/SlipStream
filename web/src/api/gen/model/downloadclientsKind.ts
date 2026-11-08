import type { DownloadclientsKindActionsItem } from "./downloadclientsKindActionsItem.ts";
import type { DownloadclientsKindKind } from "./downloadclientsKindKind.ts";
import type { DownloadclientsKindLabel } from "./downloadclientsKindLabel.ts";
import type { DownloadclientsKindPassword } from "./downloadclientsKindPassword.ts";
import type { DownloadclientsKindUsername } from "./downloadclientsKindUsername.ts";

export interface DownloadclientsKind {
  actions: DownloadclientsKindActionsItem[];
  /** It may sign in with an API key in place of the username and password */
  apiKey: boolean;
  /** Its web UI connects to a daemon the Admin may choose */
  daemon: boolean;
  /** Its remove deletes data itself */
  deletesData: boolean;
  /** The oldest version SlipStream supports, such as 4.0 */
  floor: string;
  kind: DownloadclientsKindKind;
  /** Which labels it takes: lower_word is a-z, 0-9, _ and - only */
  label: DownloadclientsKindLabel;
  /** The product's name, a new client's name by default */
  name: string;
  password: DownloadclientsKindPassword;
  /** The path an address with none takes; empty for a kind with none */
  path: string;
  username: DownloadclientsKindUsername;
}
