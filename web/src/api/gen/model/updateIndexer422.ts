import type { IndexersCaptchaImage } from "./indexersCaptchaImage.ts";
import type { Problem } from "./problem.ts";
import type { UpdateIndexer422Cause } from "./updateIndexer422Cause.ts";
import type { UpdateIndexer422Code } from "./updateIndexer422Code.ts";

export type UpdateIndexer422 = Problem & {
  /** A new captcha to answer, when the Test failed at a sign-in that shows one */
  captcha?: IndexersCaptchaImage;
  /** The stop cause, when the Test stopped */
  cause?: UpdateIndexer422Cause;
  code?: UpdateIndexer422Code;
};
