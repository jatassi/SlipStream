import type { GetImport500Code } from "./getImport500Code.ts";
import type { Problem } from "./problem.ts";

export type GetImport500 = Problem & {
  code?: GetImport500Code;
};
