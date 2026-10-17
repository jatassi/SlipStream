import type { EndAccountSessions404Code } from "./endAccountSessions404Code.ts";
import type { Problem } from "./problem.ts";

export type EndAccountSessions404 = Problem & {
  code?: EndAccountSessions404Code;
};
