import type { ConfirmMediaServerIdentity403Code } from "./confirmMediaServerIdentity403Code.ts";
import type { Problem } from "./problem.ts";

export type ConfirmMediaServerIdentity403 = Problem & {
  code?: ConfirmMediaServerIdentity403Code;
};
