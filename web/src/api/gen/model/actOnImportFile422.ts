import type { ActOnImportFile422Code } from "./actOnImportFile422Code.ts";
import type { Problem } from "./problem.ts";

export type ActOnImportFile422 = Problem & {
  code?: ActOnImportFile422Code;
};
