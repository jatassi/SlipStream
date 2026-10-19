import type { DeleteRoot409Code } from "./deleteRoot409Code.ts";
import type { Problem } from "./problem.ts";
import type { RootsInUse } from "./rootsInUse.ts";

export type DeleteRoot409 = Problem & {
  code?: DeleteRoot409Code;
  use?: RootsInUse;
};
