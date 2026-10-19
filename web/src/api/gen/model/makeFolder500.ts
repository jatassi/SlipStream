import type { MakeFolder500Code } from "./makeFolder500Code.ts";
import type { Problem } from "./problem.ts";

export type MakeFolder500 = Problem & {
  code?: MakeFolder500Code;
};
