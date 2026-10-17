import type { Problem } from "./problem.ts";
import type { SendTestMail403Code } from "./sendTestMail403Code.ts";

export type SendTestMail403 = Problem & {
  code?: SendTestMail403Code;
};
