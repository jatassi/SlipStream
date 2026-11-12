import type { ActOnRootImport404Code } from "./actOnRootImport404Code.ts";
import type { Problem } from "./problem.ts";

export type ActOnRootImport404 = Problem & {
  code?: ActOnRootImport404Code;
};
