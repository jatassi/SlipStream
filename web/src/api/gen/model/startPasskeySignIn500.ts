import type { Problem } from "./problem.ts";
import type { StartPasskeySignIn500Code } from "./startPasskeySignIn500Code.ts";

export type StartPasskeySignIn500 = Problem & {
  code?: StartPasskeySignIn500Code;
};
