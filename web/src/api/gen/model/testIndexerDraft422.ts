import type { IndexersCaptchaImage } from "./indexersCaptchaImage.ts";
import type { Problem } from "./problem.ts";
import type { TestIndexerDraft422Cause } from "./testIndexerDraft422Cause.ts";
import type { TestIndexerDraft422Code } from "./testIndexerDraft422Code.ts";

export type TestIndexerDraft422 = Problem & {
  /** A new captcha to answer, when the Test failed at a sign-in that shows one */
  captcha?: IndexersCaptchaImage;
  /** The stop cause, when the Test stopped */
  cause?: TestIndexerDraft422Cause;
  code?: TestIndexerDraft422Code;
};
