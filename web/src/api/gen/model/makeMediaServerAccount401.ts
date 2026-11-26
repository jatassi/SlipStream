import type { MakeMediaServerAccount401Code } from "./makeMediaServerAccount401Code.ts";
import type { Problem } from "./problem.ts";

export type MakeMediaServerAccount401 = Problem & {
  code?: MakeMediaServerAccount401Code;
};
