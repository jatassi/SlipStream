import type { SubtitlesExtra } from "./subtitlesExtra.ts";
import type { SubtitlesTrack } from "./subtitlesTrack.ts";
import type { SubtitlesWantView } from "./subtitlesWantView.ts";

export interface SubtitlesFileSubtitles {
  /** Subtitle files beside the video that meet no want */
  companions: SubtitlesExtra[];
  /** The media record's subtitle tracks, in stream order */
  embedded: SubtitlesTrack[];
  /** An episode file's Aired numbers */
  episodes: number[];
  fileId: string;
  /** Relative to the root */
  path: string;
  /**
   * An episode file's Aired season
   * @nullable
   */
  season: number | null;
  /** In Profile order */
  wants: SubtitlesWantView[];
}
