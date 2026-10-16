import type { Problem } from "./problem.ts";
import type { RestartNode500Code } from "./restartNode500Code.ts";

export type RestartNode500 = Problem & {
  code?: RestartNode500Code;
};
