import type { IndexersCaptchaImage } from "./indexersCaptchaImage.ts";
import type { Problem } from "./problem.ts";
import type { TestIndexerDraft422Code } from "./testIndexerDraft422Code.ts";

export type TestIndexerDraft422 = Problem & {
  captcha?: IndexersCaptchaImage;
  code?: TestIndexerDraft422Code;
};
