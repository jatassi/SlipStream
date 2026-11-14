import type { GetImport422Code } from "./getImport422Code.ts";
import type { Problem } from "./problem.ts";

export type GetImport422 = Problem & {
  code?: GetImport422Code;
};
