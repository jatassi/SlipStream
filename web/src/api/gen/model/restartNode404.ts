import type { Problem } from "./problem.ts";
import type { RestartNode404Code } from "./restartNode404Code.ts";

export type RestartNode404 = Problem & {
  code?: RestartNode404Code;
};
