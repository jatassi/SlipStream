import type { MakeFolder404Code } from "./makeFolder404Code.ts";
import type { Problem } from "./problem.ts";

export type MakeFolder404 = Problem & {
  code?: MakeFolder404Code;
};
