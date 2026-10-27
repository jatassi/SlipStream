import type { GetIndexerCaptcha422Code } from "./getIndexerCaptcha422Code.ts";
import type { IndexersCaptchaImage } from "./indexersCaptchaImage.ts";
import type { Problem } from "./problem.ts";

export type GetIndexerCaptcha422 = Problem & {
  captcha?: IndexersCaptchaImage;
  code?: GetIndexerCaptcha422Code;
};
