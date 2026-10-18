import type { DialForCode422Code } from "./dialForCode422Code.ts";
import type { Problem } from "./problem.ts";

export type DialForCode422 = Problem & {
  code?: DialForCode422Code;
};
