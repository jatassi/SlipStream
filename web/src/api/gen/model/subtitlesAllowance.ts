import type { SubtitlesWindow } from "./subtitlesWindow.ts";

export interface SubtitlesAllowance {
  /** When the provider reported it */
  at: string;
  /** Null when the provider reported no download count */
  downloads: SubtitlesWindow | null;
  /** Null when the provider reported no search count */
  searches: SubtitlesWindow | null;
}
