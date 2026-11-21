import type { TargetsEditions } from "./targetsEditions.ts";
import type { TargetsEntry } from "./targetsEntry.ts";
import type { TargetsWantedLanguage } from "./targetsWantedLanguage.ts";

export interface TargetsNewProfile {
  /** @pattern ^[a-z][a-z0-9]{1,15}$ */
  contentType: string;
  editions: TargetsEditions;
  /** In the Admin's order, each language once as full and once as forced only at most; none for a Content Type without subtitles */
  languages: TargetsWantedLanguage[];
  /**
   * @minLength 1
   * @maxLength 100
   */
  name: string;
  /** In Profile order; none is a Profile whose Works need no files */
  targets: TargetsEntry[];
}
