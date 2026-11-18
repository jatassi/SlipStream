import type { SubtitlesFit } from "./subtitlesFit.ts";
import type { SubtitlesSidecarSource } from "./subtitlesSidecarSource.ts";

export interface SubtitlesSidecar {
  /** Subtitle search stops here */
  final: boolean;
  /** Its Timing sync numbers; null when SlipStream didn't check its timing */
  fit: SubtitlesFit | null;
  /** A found file named .hi, which Media Servers read as Hindi */
  hindiWarning: boolean;
  /** Its name after the video's own, such as en.sdh.srt */
  name: string;
  /**
   * The Subtitle Provider it came from
   * @nullable
   */
  provider: string | null;
  /**
   * The release name it was made for, as the provider names it
   * @nullable
   */
  release: string | null;
  source: SubtitlesSidecarSource;
  /**
   * Where it sits while it can't take its own name, relative to the root
   * @nullable
   */
  stray: string | null;
  /** SlipStream is checking its timing against a new file */
  timingDue: boolean;
}
