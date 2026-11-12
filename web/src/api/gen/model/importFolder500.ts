import type { ImportFolder500Code } from "./importFolder500Code.ts";
import type { Problem } from "./problem.ts";

export type ImportFolder500 = Problem & {
  code?: ImportFolder500Code;
};
