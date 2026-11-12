import type { ActOnImportFile501Code } from "./actOnImportFile501Code.ts";
import type { Problem } from "./problem.ts";

export type ActOnImportFile501 = Problem & {
  code?: ActOnImportFile501Code;
};
