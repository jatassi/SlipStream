import type { ReconcilingNodeRowLink } from "./reconcilingNodeRowLink.ts";

export interface ReconcilingNodeRow {
  /** Whether its torrents were matched with the library */
  adopted: boolean;
  /** Whether Reconciling stopped waiting on it. It still keeps every torrent until SlipStream has matched its list of torrents with the library */
  continued: boolean;
  /**
   * The line naming the Downloading Role its adoption gave it, such as vps-relay now has the Downloading Role for the 1,240 torrents it reported; null when it got none
   * @nullable
   */
  granted: string | null;
  /**
   * How many torrents its list held
   * @minimum 0
   */
  held: number;
  id: string;
  /** Its connection to this Controller. A Node on a new machine that stays offline may still be paired with the old Controller */
  link: ReconcilingNodeRowLink;
  /** When its connection last changed */
  linkSince: string;
  name: string;
  /**
   * Why SlipStream refused its list of torrents; null when it didn't. It keeps every torrent until SlipStream has matched its list of torrents with the library
   * @nullable
   */
  refused: string | null;
  /** Whether every page of its list of torrents is in */
  reported: boolean;
  /** When Reconciling began waiting on it */
  since: string;
}
