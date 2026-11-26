import type { MakeMediaServerAccount403Code } from "./makeMediaServerAccount403Code.ts";
import type { Problem } from "./problem.ts";

export type MakeMediaServerAccount403 = Problem & {
  code?: MakeMediaServerAccount403Code;
};
