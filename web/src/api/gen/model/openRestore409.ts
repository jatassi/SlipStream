import type { OpenRestore409Code } from "./openRestore409Code.ts";
import type { Problem } from "./problem.ts";

export type OpenRestore409 = Problem & {
  code?: OpenRestore409Code;
};
