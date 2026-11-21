import type { OpenRestore404Code } from "./openRestore404Code.ts";
import type { Problem } from "./problem.ts";

export type OpenRestore404 = Problem & {
  code?: OpenRestore404Code;
};
