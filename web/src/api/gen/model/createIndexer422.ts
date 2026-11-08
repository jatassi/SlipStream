import type { CreateIndexer422Cause } from "./createIndexer422Cause.ts";
import type { CreateIndexer422Code } from "./createIndexer422Code.ts";
import type { IndexersCaptchaImage } from "./indexersCaptchaImage.ts";
import type { Problem } from "./problem.ts";

export type CreateIndexer422 = Problem & {
  /** A new captcha to answer, when the Test failed at a sign-in that shows one */
  captcha?: IndexersCaptchaImage;
  /** The stop cause, when the Test stopped */
  cause?: CreateIndexer422Cause;
  code?: CreateIndexer422Code;
};
