import type { Problem } from "./problem.ts";
import type { RehomeRoot401Code } from "./rehomeRoot401Code.ts";

export type RehomeRoot401 = Problem & {
  code?: RehomeRoot401Code;
};
