import type { PlaybackGroup } from "./playbackGroup.ts";

export interface PlaybackPlayer {
  /** @minimum 0 */
  directPlays: number;
  /** @minimum 0 */
  directStreams: number;
  /** Largest first */
  groups: PlaybackGroup[];
  key: string;
  lastPlayedAt: string;
  name: string;
  platform: string;
  /** @minimum 0 */
  plays: number;
  product: string;
  /** @minimum 0 */
  transcoded: number;
  /** @minimum 0 */
  unmatched: number;
}
