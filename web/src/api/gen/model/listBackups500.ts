import type { ListBackups500Code } from "./listBackups500Code.ts";
import type { Problem } from "./problem.ts";

export type ListBackups500 = Problem & {
  code?: ListBackups500Code;
};
