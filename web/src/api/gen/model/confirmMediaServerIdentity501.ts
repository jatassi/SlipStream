import type { ConfirmMediaServerIdentity501Code } from "./confirmMediaServerIdentity501Code.ts";
import type { Problem } from "./problem.ts";

export type ConfirmMediaServerIdentity501 = Problem & {
  code?: ConfirmMediaServerIdentity501Code;
};
