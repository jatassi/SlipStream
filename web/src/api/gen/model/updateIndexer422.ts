import type { IndexersCaptchaImage } from "./indexersCaptchaImage.ts";
import type { Problem } from "./problem.ts";
import type { UpdateIndexer422Code } from "./updateIndexer422Code.ts";

export type UpdateIndexer422 = Problem & {
  captcha?: IndexersCaptchaImage;
  code?: UpdateIndexer422Code;
};
