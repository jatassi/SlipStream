import type { ImportFolder401Code } from "./importFolder401Code.ts";
import type { Problem } from "./problem.ts";

export type ImportFolder401 = Problem & {
  code?: ImportFolder401Code;
};
