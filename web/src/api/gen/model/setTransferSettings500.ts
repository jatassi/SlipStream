import type { Problem } from "./problem.ts";
import type { SetTransferSettings500Code } from "./setTransferSettings500Code.ts";

export type SetTransferSettings500 = Problem & {
  code?: SetTransferSettings500Code;
};
