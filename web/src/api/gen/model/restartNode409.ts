import type { Problem } from "./problem.ts";
import type { RestartNode409Code } from "./restartNode409Code.ts";

export type RestartNode409 = Problem & {
  code?: RestartNode409Code;
};
