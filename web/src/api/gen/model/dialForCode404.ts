import type { DialForCode404Code } from "./dialForCode404Code.ts";
import type { Problem } from "./problem.ts";

export type DialForCode404 = Problem & {
  code?: DialForCode404Code;
};
