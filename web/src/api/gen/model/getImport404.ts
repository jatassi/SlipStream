import type { GetImport404Code } from "./getImport404Code.ts";
import type { Problem } from "./problem.ts";

export type GetImport404 = Problem & {
  code?: GetImport404Code;
};
