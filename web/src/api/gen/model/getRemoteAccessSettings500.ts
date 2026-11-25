import type { GetRemoteAccessSettings500Code } from "./getRemoteAccessSettings500Code.ts";
import type { Problem } from "./problem.ts";

export type GetRemoteAccessSettings500 = Problem & {
  code?: GetRemoteAccessSettings500Code;
};
