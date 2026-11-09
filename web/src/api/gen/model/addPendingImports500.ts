import type { AddPendingImports500Code } from "./addPendingImports500Code.ts";
import type { Problem } from "./problem.ts";

export type AddPendingImports500 = Problem & {
  code?: AddPendingImports500Code;
};
