import type { GetImport409Code } from "./getImport409Code.ts";
import type { Problem } from "./problem.ts";

export type GetImport409 = Problem & {
  code?: GetImport409Code;
};
