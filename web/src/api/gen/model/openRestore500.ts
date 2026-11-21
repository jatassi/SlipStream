import type { OpenRestore500Code } from "./openRestore500Code.ts";
import type { Problem } from "./problem.ts";

export type OpenRestore500 = Problem & {
  code?: OpenRestore500Code;
};
