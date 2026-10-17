import type { Problem } from "./problem.ts";
import type { SendTestMail500Code } from "./sendTestMail500Code.ts";

export type SendTestMail500 = Problem & {
  code?: SendTestMail500Code;
};
