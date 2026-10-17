import type { Problem } from "./problem.ts";
import type { StartPasskeyRegistration409Code } from "./startPasskeyRegistration409Code.ts";

export type StartPasskeyRegistration409 = Problem & {
  code?: StartPasskeyRegistration409Code;
};
