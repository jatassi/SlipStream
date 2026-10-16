import type { Problem } from "./problem.ts";
import type { RestartNode503Code } from "./restartNode503Code.ts";

export type RestartNode503 = Problem & {
  code?: RestartNode503Code;
};
