import type { Problem } from "./problem.ts";
import type { ShareMediaServerAccess403Code } from "./shareMediaServerAccess403Code.ts";

export type ShareMediaServerAccess403 = Problem & {
  code?: ShareMediaServerAccess403Code;
};
