import type { Problem } from "./problem.ts";
import type { ShareMediaServerAccess502Code } from "./shareMediaServerAccess502Code.ts";

export type ShareMediaServerAccess502 = Problem & {
  code?: ShareMediaServerAccess502Code;
};
