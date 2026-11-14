import type { ConfirmMediaServerIdentity401Code } from "./confirmMediaServerIdentity401Code.ts";
import type { Problem } from "./problem.ts";

export type ConfirmMediaServerIdentity401 = Problem & {
  code?: ConfirmMediaServerIdentity401Code;
};
