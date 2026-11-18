import type { SubtitlesSidecar } from "./subtitlesSidecar.ts";
import type { SubtitlesWantStateKind } from "./subtitlesWantStateKind.ts";
import type { SubtitlesWantStateMissing } from "./subtitlesWantStateMissing.ts";
import type { SubtitlesWantStateWhy } from "./subtitlesWantStateWhy.ts";

export interface SubtitlesWantState {
  kind: SubtitlesWantStateKind;
  /**
   * Why a missing want has no subtitle yet, the first that holds
   * @nullable
   */
  missing: SubtitlesWantStateMissing;
  /**
   * Its next search; null when none is planned
   * @nullable
   */
  next: string | null;
  /** The subtitle file that meets it */
  sidecar: SubtitlesSidecar | null;
  /**
   * The embedded track that meets it, by its position among the file's subtitle tracks
   * @nullable
   */
  track: number | null;
  /**
   * Why a not_needed want needs nothing
   * @nullable
   */
  why: SubtitlesWantStateWhy;
}
