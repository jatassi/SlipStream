import type { ConfirmImports404Code } from "./confirmImports404Code.ts";
import type { Problem } from "./problem.ts";

export type ConfirmImports404 = Problem & {
  code?: ConfirmImports404Code;
};
