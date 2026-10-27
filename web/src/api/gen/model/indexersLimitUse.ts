import type { IndexersLimitUsePer } from "./indexersLimitUsePer.ts";

export interface IndexersLimitUse {
  /**
   * 0 is no limit
   * @minimum 0
   */
  count: number;
  /**
   * When the oldest counted call leaves the window; null while under the limit
   * @nullable
   */
  freeAt: string | null;
  per: IndexersLimitUsePer;
  /**
   * The calls counted in the window now
   * @minimum 0
   */
  used: number;
}
