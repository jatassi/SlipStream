import type { Problem } from "./problem.ts";
import type { RehomeRoot404Code } from "./rehomeRoot404Code.ts";

export type RehomeRoot404 = Problem & {
  code?: RehomeRoot404Code;
};
