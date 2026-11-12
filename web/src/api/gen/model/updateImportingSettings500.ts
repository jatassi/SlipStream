import type { Problem } from "./problem.ts";
import type { UpdateImportingSettings500Code } from "./updateImportingSettings500Code.ts";

export type UpdateImportingSettings500 = Problem & {
  code?: UpdateImportingSettings500Code;
};
