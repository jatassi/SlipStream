import type { ActOnImportFile404Code } from "./actOnImportFile404Code.ts";
import type { Problem } from "./problem.ts";

export type ActOnImportFile404 = Problem & {
  code?: ActOnImportFile404Code;
};
