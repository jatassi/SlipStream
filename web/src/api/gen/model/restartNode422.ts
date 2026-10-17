import type { Problem } from "./problem.ts";
import type { RestartNode422Code } from "./restartNode422Code.ts";

export type RestartNode422 = Problem & {
  code?: RestartNode422Code;
};
