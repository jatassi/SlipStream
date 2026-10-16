import type { Problem } from "./problem.ts";
import type { StartPasskeyRegistration401Code } from "./startPasskeyRegistration401Code.ts";

export type StartPasskeyRegistration401 = Problem & {
  code?: StartPasskeyRegistration401Code;
};
