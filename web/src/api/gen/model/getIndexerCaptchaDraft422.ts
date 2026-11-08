import type { GetIndexerCaptchaDraft422Cause } from "./getIndexerCaptchaDraft422Cause.ts";
import type { GetIndexerCaptchaDraft422Code } from "./getIndexerCaptchaDraft422Code.ts";
import type { IndexersCaptchaImage } from "./indexersCaptchaImage.ts";
import type { Problem } from "./problem.ts";

export type GetIndexerCaptchaDraft422 = Problem & {
  /** A new captcha to answer, when the Test failed at a sign-in that shows one */
  captcha?: IndexersCaptchaImage;
  /** The stop cause, when the Test stopped */
  cause?: GetIndexerCaptchaDraft422Cause;
  code?: GetIndexerCaptchaDraft422Code;
};
