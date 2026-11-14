import type { ConfirmImports401Code } from "./confirmImports401Code.ts";
import type { Problem } from "./problem.ts";

export type ConfirmImports401 = Problem & {
  code?: ConfirmImports401Code;
};
