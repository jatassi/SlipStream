import type { PlaybackBlockedDimension } from "./playbackBlockedDimension.ts";

export interface PlaybackBlocked {
  /** A Target can choose this dimension */
  choosable: boolean;
  dimension: PlaybackBlockedDimension;
  /** The played file's value in release words; the reason's name for audio_track, video_shape and audio_format; empty where none applies */
  value: string;
}
