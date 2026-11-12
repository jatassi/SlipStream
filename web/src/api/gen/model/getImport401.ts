import type { GetImport401Code } from "./getImport401Code.ts";
import type { Problem } from "./problem.ts";

export type GetImport401 = Problem & {
  code?: GetImport401Code;
};
