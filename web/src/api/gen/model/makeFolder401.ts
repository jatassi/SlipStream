import type { MakeFolder401Code } from "./makeFolder401Code.ts";
import type { Problem } from "./problem.ts";

export type MakeFolder401 = Problem & {
  code?: MakeFolder401Code;
};
