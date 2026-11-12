import type { Problem } from "./problem.ts";
import type { UpdateImportingSettings401Code } from "./updateImportingSettings401Code.ts";

export type UpdateImportingSettings401 = Problem & {
  code?: UpdateImportingSettings401Code;
};
