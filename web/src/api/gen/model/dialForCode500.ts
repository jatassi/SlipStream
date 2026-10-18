import type { DialForCode500Code } from "./dialForCode500Code.ts";
import type { Problem } from "./problem.ts";

export type DialForCode500 = Problem & {
  code?: DialForCode500Code;
};
