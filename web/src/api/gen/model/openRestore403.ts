import type { OpenRestore403Code } from "./openRestore403Code.ts";
import type { Problem } from "./problem.ts";

export type OpenRestore403 = Problem & {
  code?: OpenRestore403Code;
};
