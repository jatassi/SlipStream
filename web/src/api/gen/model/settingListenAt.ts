import type { SettingListenAtHost } from "./settingListenAtHost.ts";

export interface SettingListenAt {
  /** The address and port, such as [::]:8484 */
  address: string;
  /** all is every network, local is this machine only, other is an address set in config.yaml or the environment */
  host: SettingListenAtHost;
  port: number;
}
