import type { MakeMediaServerAccount404Code } from "./makeMediaServerAccount404Code.ts";
import type { Problem } from "./problem.ts";

export type MakeMediaServerAccount404 = Problem & {
  code?: MakeMediaServerAccount404Code;
};
