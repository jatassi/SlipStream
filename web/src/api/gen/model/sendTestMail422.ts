import type { Problem } from "./problem.ts";
import type { SendTestMail422Code } from "./sendTestMail422Code.ts";

export type SendTestMail422 = Problem & {
  code?: SendTestMail422Code;
};
