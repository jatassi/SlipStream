import type { MakeMediaServerAccount500Code } from "./makeMediaServerAccount500Code.ts";
import type { Problem } from "./problem.ts";

export type MakeMediaServerAccount500 = Problem & {
  code?: MakeMediaServerAccount500Code;
};
