import type { GetImportingSettings401Code } from "./getImportingSettings401Code.ts";
import type { Problem } from "./problem.ts";

export type GetImportingSettings401 = Problem & {
  code?: GetImportingSettings401Code;
};
