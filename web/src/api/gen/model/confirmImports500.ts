import type { ConfirmImports500Code } from "./confirmImports500Code.ts";
import type { Problem } from "./problem.ts";

export type ConfirmImports500 = Problem & {
  code?: ConfirmImports500Code;
};
