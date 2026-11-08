import type { GetIndexerCaptcha422Cause } from "./getIndexerCaptcha422Cause.ts";
import type { GetIndexerCaptcha422Code } from "./getIndexerCaptcha422Code.ts";
import type { IndexersCaptchaImage } from "./indexersCaptchaImage.ts";
import type { Problem } from "./problem.ts";

export type GetIndexerCaptcha422 = Problem & {
  /** A new captcha to answer, when the Test failed at a sign-in that shows one */
  captcha?: IndexersCaptchaImage;
  /** The stop cause, when the Test stopped */
  cause?: GetIndexerCaptcha422Cause;
  code?: GetIndexerCaptcha422Code;
};
