import type { Problem } from "./problem.ts";
import type { RestartNode401Code } from "./restartNode401Code.ts";

export type RestartNode401 = Problem & {
  code?: RestartNode401Code;
};
