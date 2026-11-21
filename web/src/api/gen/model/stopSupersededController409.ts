import type { Problem } from "./problem.ts";
import type { StopSupersededController409Code } from "./stopSupersededController409Code.ts";

export type StopSupersededController409 = Problem & {
  code?: StopSupersededController409Code;
};
