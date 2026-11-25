import type { Problem } from "./problem.ts";
import type { ShareMediaServerAccess401Code } from "./shareMediaServerAccess401Code.ts";

export type ShareMediaServerAccess401 = Problem & {
  code?: ShareMediaServerAccess401Code;
};
