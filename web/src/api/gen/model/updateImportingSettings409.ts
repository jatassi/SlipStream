import type { Problem } from "./problem.ts";
import type { UpdateImportingSettings409Code } from "./updateImportingSettings409Code.ts";

export type UpdateImportingSettings409 = Problem & {
  code?: UpdateImportingSettings409Code;
};
