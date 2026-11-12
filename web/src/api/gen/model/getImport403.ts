import type { GetImport403Code } from "./getImport403Code.ts";
import type { Problem } from "./problem.ts";

export type GetImport403 = Problem & {
  code?: GetImport403Code;
};
