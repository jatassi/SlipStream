import type { TargetsDroppedLanguage } from "./targetsDroppedLanguage.ts";
import type { TargetsGainedLanguage } from "./targetsGainedLanguage.ts";

export interface TargetsSubtitles {
  /**
   * About how many days the gained searches take at the providers' last reported download allowance, 0 when none reported one
   * @minimum 0
   */
  days: number;
  /** Each subtitle language the change removes */
  dropped: TargetsDroppedLanguage[];
  /** Each subtitle language the change adds, in the Profile's order */
  gained: TargetsGainedLanguage[];
}
