import type { Problem } from "./problem.ts";
import type { SendTestMail401Code } from "./sendTestMail401Code.ts";

export type SendTestMail401 = Problem & {
  code?: SendTestMail401Code;
};
