import type { ActOnImportFile409Code } from "./actOnImportFile409Code.ts";
import type { Problem } from "./problem.ts";

export type ActOnImportFile409 = Problem & {
  code?: ActOnImportFile409Code;
};
