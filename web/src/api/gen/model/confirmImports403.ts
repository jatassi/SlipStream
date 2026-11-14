import type { ConfirmImports403Code } from "./confirmImports403Code.ts";
import type { Problem } from "./problem.ts";

export type ConfirmImports403 = Problem & {
  code?: ConfirmImports403Code;
};
