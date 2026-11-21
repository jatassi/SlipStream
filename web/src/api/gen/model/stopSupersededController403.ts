import type { Problem } from "./problem.ts";
import type { StopSupersededController403Code } from "./stopSupersededController403Code.ts";

export type StopSupersededController403 = Problem & {
  code?: StopSupersededController403Code;
};
