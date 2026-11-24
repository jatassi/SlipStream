import type { GetRemoteAccessSettings401Code } from "./getRemoteAccessSettings401Code.ts";
import type { Problem } from "./problem.ts";

export type GetRemoteAccessSettings401 = Problem & {
  code?: GetRemoteAccessSettings401Code;
};
