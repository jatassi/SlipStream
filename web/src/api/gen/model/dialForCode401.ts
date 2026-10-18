import type { DialForCode401Code } from "./dialForCode401Code.ts";
import type { Problem } from "./problem.ts";

export type DialForCode401 = Problem & {
  code?: DialForCode401Code;
};
