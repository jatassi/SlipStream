import type { FilesAudioTrack } from "./filesAudioTrack.ts";
import type { FilesSubtitleTrack } from "./filesSubtitleTrack.ts";
import type { FilesVideoInfo } from "./filesVideoInfo.ts";

export interface FilesMediaInfo {
  /** In stream order */
  audio: FilesAudioTrack[];
  /**
   * Bits per second; 0 when the container states none
   * @minimum 0
   */
  bitrate: number;
  /** ffprobe's format name, such as matroska,webm */
  container: string;
  /**
   * 0 when the container states none
   * @minimum 0
   */
  durationSeconds: number;
  /** In stream order */
  subtitles: FilesSubtitleTrack[];
  /** Its first video stream that isn't cover art */
  video: FilesVideoInfo;
}
