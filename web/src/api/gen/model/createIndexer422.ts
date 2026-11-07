import type { CreateIndexer422Code } from "./createIndexer422Code.ts";
import type { IndexersCaptchaImage } from "./indexersCaptchaImage.ts";
import type { Problem } from "./problem.ts";

export type CreateIndexer422 = Problem & {
  captcha?: IndexersCaptchaImage;
  code?: CreateIndexer422Code;
};
