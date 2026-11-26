import type { PlaybackPlayer } from "./playbackPlayer.ts";

export interface PlaybackReport {
  /** Worst first */
  players: PlaybackPlayer[];
  /** @minimum 0 */
  plays: number;
  /** The server is Plex, which gives no reasons */
  plex: boolean;
  /** false while the server's Connect from Node is offline, when nothing is recorded */
  reachable: boolean;
  /** The report covers plays that started at or after this */
  since: string;
  /** @minimum 0 */
  transcoded: number;
  /**
   * Transcoded plays with no snapshot
   * @minimum 0
   */
  unmatched: number;
  /**
   * Keep play history for, in seconds
   * @minimum 1
   */
  window: number;
}
