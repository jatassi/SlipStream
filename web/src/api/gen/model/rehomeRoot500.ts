import type { Problem } from "./problem.ts";
import type { RehomeRoot500Code } from "./rehomeRoot500Code.ts";

export type RehomeRoot500 = Problem & {
  code?: RehomeRoot500Code;
};
