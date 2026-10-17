import type { Problem } from "./problem.ts";
import type { StartPasskeySignIn429Code } from "./startPasskeySignIn429Code.ts";

export type StartPasskeySignIn429 = Problem & {
  code?: StartPasskeySignIn429Code;
};
