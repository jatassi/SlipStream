import type { Problem } from "./problem.ts";
import type { SetTransferSettings401Code } from "./setTransferSettings401Code.ts";

export type SetTransferSettings401 = Problem & {
  code?: SetTransferSettings401Code;
};
