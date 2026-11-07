import type { IndexersCaptchaAnswer } from "./indexersCaptchaAnswer.ts";

export interface IndexersTestBody {
  /** The answer to a captcha loaded for this Indexer */
  captcha?: IndexersCaptchaAnswer;
  /**
   * The current two-factor code, for a site whose sign-in asks for one
   * @maxLength 16
   */
  code?: string;
}
