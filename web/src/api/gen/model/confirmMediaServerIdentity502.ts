import type { ConfirmMediaServerIdentity502Code } from "./confirmMediaServerIdentity502Code.ts";
import type { Problem } from "./problem.ts";

export type ConfirmMediaServerIdentity502 = Problem & {
  code?: ConfirmMediaServerIdentity502Code;
};
