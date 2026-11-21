import type { OpenRestore422Code } from "./openRestore422Code.ts";
import type { Problem } from "./problem.ts";

export type OpenRestore422 = Problem & {
  code?: OpenRestore422Code;
};
