import type { Problem } from "./problem.ts";
import type { StopSupersededController401Code } from "./stopSupersededController401Code.ts";

export type StopSupersededController401 = Problem & {
  code?: StopSupersededController401Code;
};
