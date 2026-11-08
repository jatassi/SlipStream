import type { TrackersAttentionLevel } from "./trackersAttentionLevel.ts";

export interface TrackersAttention {
  level: TrackersAttentionLevel;
  /** One sentence */
  text: string;
}
