import type { ConfirmMediaServerIdentity422Code } from "./confirmMediaServerIdentity422Code.ts";
import type { Problem } from "./problem.ts";

export type ConfirmMediaServerIdentity422 = Problem & {
  code?: ConfirmMediaServerIdentity422Code;
};
