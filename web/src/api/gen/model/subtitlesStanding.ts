import type { SubtitlesStandingKind } from "./subtitlesStandingKind.ts";

export interface SubtitlesStanding {
  /** The first that holds: off, a key is needed, the key was rejected, out of downloads or rate limited, failing, ready */
  kind: SubtitlesStandingKind;
  /**
   * When the provider rejected the key, or when a failing provider started failing
   * @nullable
   */
  since: string | null;
  /**
   * When a provider that is out takes calls again
   * @nullable
   */
  until: string | null;
  /**
   * The provider's own words for a rejected key, or the last error of a failing provider
   * @nullable
   */
  words: string | null;
}
