import type { Problem } from "./problem.ts";
import type { StartPasskeyRegistration403Code } from "./startPasskeyRegistration403Code.ts";

export type StartPasskeyRegistration403 = Problem & {
  code?: StartPasskeyRegistration403Code;
};
