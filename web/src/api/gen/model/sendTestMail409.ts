import type { Problem } from "./problem.ts";
import type { SendTestMail409Code } from "./sendTestMail409Code.ts";

export type SendTestMail409 = Problem & {
  code?: SendTestMail409Code;
};
