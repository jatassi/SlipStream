import type { Problem } from "./problem.ts";
import type { ShareMediaServerAccess409Code } from "./shareMediaServerAccess409Code.ts";

export type ShareMediaServerAccess409 = Problem & {
  code?: ShareMediaServerAccess409Code;
};
