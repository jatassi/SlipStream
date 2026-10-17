import type { Problem } from "./problem.ts";
import type { StartPasskeySignIn409Code } from "./startPasskeySignIn409Code.ts";

export type StartPasskeySignIn409 = Problem & {
  code?: StartPasskeySignIn409Code;
};
