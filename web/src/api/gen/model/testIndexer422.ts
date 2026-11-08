import type { IndexersCaptchaImage } from "./indexersCaptchaImage.ts";
import type { Problem } from "./problem.ts";
import type { TestIndexer422Cause } from "./testIndexer422Cause.ts";
import type { TestIndexer422Code } from "./testIndexer422Code.ts";

export type TestIndexer422 = Problem & {
  /** A new captcha to answer, when the Test failed at a sign-in that shows one */
  captcha?: IndexersCaptchaImage;
  /** The stop cause, when the Test stopped */
  cause?: TestIndexer422Cause;
  code?: TestIndexer422Code;
};
