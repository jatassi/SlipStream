import type { EndAccountSessions401Code } from "./endAccountSessions401Code.ts";
import type { Problem } from "./problem.ts";

export type EndAccountSessions401 = Problem & {
  code?: EndAccountSessions401Code;
};
