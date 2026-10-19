import type { Problem } from "./problem.ts";
import type { RehomeRoot403Code } from "./rehomeRoot403Code.ts";

export type RehomeRoot403 = Problem & {
  code?: RehomeRoot403Code;
};
