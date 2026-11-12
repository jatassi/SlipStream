import type { ActOnImportFile500Code } from "./actOnImportFile500Code.ts";
import type { Problem } from "./problem.ts";

export type ActOnImportFile500 = Problem & {
  code?: ActOnImportFile500Code;
};
