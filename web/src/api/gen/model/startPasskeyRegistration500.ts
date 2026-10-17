import type { Problem } from "./problem.ts";
import type { StartPasskeyRegistration500Code } from "./startPasskeyRegistration500Code.ts";

export type StartPasskeyRegistration500 = Problem & {
  code?: StartPasskeyRegistration500Code;
};
