import type { ConfirmMediaServerIdentity500Code } from "./confirmMediaServerIdentity500Code.ts";
import type { Problem } from "./problem.ts";

export type ConfirmMediaServerIdentity500 = Problem & {
  code?: ConfirmMediaServerIdentity500Code;
};
