import type { ActOnRootImport500Code } from "./actOnRootImport500Code.ts";
import type { Problem } from "./problem.ts";

export type ActOnRootImport500 = Problem & {
  code?: ActOnRootImport500Code;
};
