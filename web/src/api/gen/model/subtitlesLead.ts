import type { SubtitlesLeadKind } from "./subtitlesLeadKind.ts";
import type { SubtitlesScans } from "./subtitlesScans.ts";

export interface SubtitlesLead {
  /** Seconds between Subtitle search runs */
  interval: number;
  /** Which sentence leads the section, the first that applies: no Profile has a Wanted Language, a Content Type with Wanted Languages has no Subtitle Provider that can search, Subtitle search is off, or ready */
  kind: SubtitlesLeadKind;
  /**
   * Wanted subtitles that no subtitle file or embedded track meets yet
   * @minimum 0
   */
  missing: number;
  /** Each Storage Node with files whose audio SlipStream still has to scan, in Node order */
  scans: SubtitlesScans[];
  /** The Content Types with Wanted Languages for which no Subtitle Provider is on with its key */
  uncovered: string[];
}
