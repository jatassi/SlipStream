import type { GetRemoteAccessSettings403Code } from "./getRemoteAccessSettings403Code.ts";
import type { Problem } from "./problem.ts";

export type GetRemoteAccessSettings403 = Problem & {
  code?: GetRemoteAccessSettings403Code;
};
