import type { GetIndexerCaptchaDraft422Code } from "./getIndexerCaptchaDraft422Code.ts";
import type { IndexersCaptchaImage } from "./indexersCaptchaImage.ts";
import type { Problem } from "./problem.ts";

export type GetIndexerCaptchaDraft422 = Problem & {
  captcha?: IndexersCaptchaImage;
  code?: GetIndexerCaptchaDraft422Code;
};
