import type { Problem } from "./problem.ts";
import type { ShareMediaServerAccess500Code } from "./shareMediaServerAccess500Code.ts";

export type ShareMediaServerAccess500 = Problem & {
  code?: ShareMediaServerAccess500Code;
};
