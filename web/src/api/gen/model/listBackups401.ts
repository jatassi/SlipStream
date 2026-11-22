import type { ListBackups401Code } from "./listBackups401Code.ts";
import type { Problem } from "./problem.ts";

export type ListBackups401 = Problem & {
  code?: ListBackups401Code;
};
