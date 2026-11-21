import type { Problem } from "./problem.ts";
import type { StopSupersededController500Code } from "./stopSupersededController500Code.ts";

export type StopSupersededController500 = Problem & {
  code?: StopSupersededController500Code;
};
