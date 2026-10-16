import type { FinishPasskeySignIn401Code } from "./finishPasskeySignIn401Code.ts";
import type { Problem } from "./problem.ts";

export type FinishPasskeySignIn401 = Problem & {
  code?: FinishPasskeySignIn401Code;
};
