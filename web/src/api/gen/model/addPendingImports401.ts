import type { AddPendingImports401Code } from "./addPendingImports401Code.ts";
import type { Problem } from "./problem.ts";

export type AddPendingImports401 = Problem & {
  code?: AddPendingImports401Code;
};
