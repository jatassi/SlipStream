import type { OpenRestore401Code } from "./openRestore401Code.ts";
import type { Problem } from "./problem.ts";

export type OpenRestore401 = Problem & {
  code?: OpenRestore401Code;
};
