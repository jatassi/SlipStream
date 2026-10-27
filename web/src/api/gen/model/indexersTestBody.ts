import type { IndexersCaptchaAnswer } from "./indexersCaptchaAnswer.ts";

export interface IndexersTestBody {
  /** The answer to a captcha loaded for this Indexer */
  captcha?: IndexersCaptchaAnswer;
}
