import type { FinishPasskeySignIn500Code } from "./finishPasskeySignIn500Code.ts";
import type { Problem } from "./problem.ts";

export type FinishPasskeySignIn500 = Problem & {
  code?: FinishPasskeySignIn500Code;
};
