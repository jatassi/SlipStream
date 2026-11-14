import type { GetImportingSettings500Code } from "./getImportingSettings500Code.ts";
import type { Problem } from "./problem.ts";

export type GetImportingSettings500 = Problem & {
  code?: GetImportingSettings500Code;
};
